package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/pb"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/store/models"
	log "gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

// AnalyzerConfig is a configuration of analyzer
type AnalyzerConfig struct {
	Name string
	// Addr is gRPC URL.
	// can be defined only in global config, repository-scoped configuration is ignored
	Addr string
	// Disabled repository-scoped configuration can accept only true value, false value is ignored
	Disabled bool
	// Feedback is a url to be linked after each comment
	Feedback string
	// Settings any configuration for an analyzer
	Settings map[string]interface{}
}

// ServerConfig is a server configuration
type ServerConfig struct {
	Analyzers []AnalyzerConfig
}

// Analyzer is a struct of analyzer client and config
type Analyzer struct {
	Client lookout.AnalyzerClient
	Config AnalyzerConfig
}

// AnalyzerComments contains a group of comments and the config for the
// analyzer that created them
type AnalyzerComments struct {
	Config   AnalyzerConfig
	Comments []*lookout.Comment
}

type reqSent func(client lookout.AnalyzerClient, settings map[string]interface{}) ([]*lookout.Comment, error)

// Server implements glue between providers / data-server / analyzers
type Server struct {
	watcher    lookout.Watcher
	poster     lookout.Poster
	fileGetter lookout.FileGetter
	analyzers  map[string]Analyzer
	eventOp    store.EventOperator
	commentOp  store.CommentOperator
}

// NewServer creates new Server
func NewServer(w lookout.Watcher, p lookout.Poster, fileGetter lookout.FileGetter,
	analyzers map[string]Analyzer, eventOp store.EventOperator, commentOp store.CommentOperator) *Server {
	return &Server{w, p, fileGetter, analyzers, eventOp, commentOp}
}

// Run starts server
func (s *Server) Run(ctx context.Context) error {
	// FIXME(max): we most probably want to change interface of EventHandler instead of it
	return s.watcher.Watch(ctx, func(e lookout.Event) error {
		return s.handleEvent(ctx, e)
	})
}

func (s *Server) handleEvent(ctx context.Context, e lookout.Event) (err error) {
	logger := log.With(log.Fields{
		"event-type": e.Type(),
		"event-id":   e.ID(),
	})

	status, err := s.eventOp.Save(ctx, e)
	if err != nil {
		logger.Errorf(err, "can't save event to database")
		return
	}

	if status == models.EventStatusProcessed {
		logger.Infof("event successfully processed, skipping...")
		return
	}

	defer func() {
		if err == nil {
			status = models.EventStatusProcessed
		} else {
			logger.Errorf(err, "event processing failed")
			// don't fail on event processing error, just skip it
			err = nil
			status = models.EventStatusFailed
		}
		if updateErr := s.eventOp.UpdateStatus(ctx, e, status); updateErr != nil {
			logger.Errorf(updateErr, "can't update status in database")
		}
	}()

	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		err = s.HandleReview(ctx, ev)
	case *lookout.PushEvent:
		err = s.HandlePush(ctx, ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
	}

	return
}

// HandleReview sends request to analyzers concurrently
func (s *Server) HandleReview(ctx context.Context, e *lookout.ReviewEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing pull request")

	if err := e.Validate(); err != nil {
		return err
	}

	conf, err := s.getConfig(ctx, logger, e)
	if err != nil {
		return err
	}

	s.status(ctx, logger, e, lookout.PendingAnalysisStatus)

	send := func(a lookout.AnalyzerClient, settings map[string]interface{}) ([]*lookout.Comment, error) {
		st := pb.ToStruct(settings)
		if st != nil {
			e.Configuration = *st
		}
		resp, err := a.NotifyReviewEvent(ctx, e)
		if err != nil {
			return nil, err
		}
		return resp.Comments, nil
	}
	comments := s.concurrentRequest(ctx, logger, conf, send)

	if err := s.post(ctx, logger, e, comments); err != nil {
		return fmt.Errorf("posting analysis failed: %s", err)
	}

	s.status(ctx, logger, e, lookout.SuccessAnalysisStatus)

	return nil
}

// HandlePush sends request to analyzers concurrently
func (s *Server) HandlePush(ctx context.Context, e *lookout.PushEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing push")

	if err := e.Validate(); err != nil {
		return err
	}

	conf, err := s.getConfig(ctx, logger, e)
	if err != nil {
		return err
	}

	s.status(ctx, logger, e, lookout.PendingAnalysisStatus)

	send := func(a lookout.AnalyzerClient, settings map[string]interface{}) ([]*lookout.Comment, error) {
		st := pb.ToStruct(settings)
		if st != nil {
			e.Configuration = *st
		}
		resp, err := a.NotifyPushEvent(ctx, e)
		if err != nil {
			return nil, err
		}
		return resp.Comments, nil
	}
	comments := s.concurrentRequest(ctx, logger, conf, send)

	if err := s.post(ctx, logger, e, comments); err != nil {
		return fmt.Errorf("posting analysis failed: %s", err)
	}
	s.status(ctx, logger, e, lookout.SuccessAnalysisStatus)

	return nil
}

// FIXME(max): it's better to hold logger inside context
func (s *Server) getConfig(ctx context.Context, logger log.Logger, e lookout.Event) (map[string]AnalyzerConfig, error) {
	rev := e.Revision()
	scanner, err := s.fileGetter.GetFiles(ctx, &lookout.FilesRequest{
		Revision:       &rev.Head,
		IncludePattern: `^\.lookout\.yml$`,
		WantContents:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("Can't get .lookout.yml in revision %s: %s", rev.Head, err)
	}
	var configContent []byte
	if scanner.Next() {
		configContent = scanner.File().Content
	}
	scanner.Close()
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(configContent) == 0 {
		logger.Infof("repository config is not found")
		return nil, nil
	}

	var conf ServerConfig
	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return nil, fmt.Errorf("Can't parse configuration file: %s", err)
	}

	res := make(map[string]AnalyzerConfig, len(s.analyzers))
	for name, a := range s.analyzers {
		res[name] = a.Config
	}
	for _, aConf := range conf.Analyzers {
		if _, ok := s.analyzers[aConf.Name]; !ok {
			logger.Warningf("analyzer '%s' required by local config isn't enabled on server", aConf.Name)
			continue
		}
		res[aConf.Name] = aConf
	}

	return res, nil
}

// FIXME(max): it's better to hold logger inside context
func (s *Server) concurrentRequest(ctx context.Context, logger log.Logger, conf map[string]AnalyzerConfig, send reqSent) []AnalyzerComments {
	var comments commentsList

	var wg sync.WaitGroup
	for name, a := range s.analyzers {
		if a.Config.Disabled {
			continue
		}

		wg.Add(1)
		go func(name string, a Analyzer) {
			defer wg.Done()

			aLogger := logger.With(log.Fields{
				"analyzer": name,
			})

			settings := mergeSettings(a.Config.Settings, conf[name].Settings)
			cs, err := send(a.Client, settings)
			if err != nil {
				aLogger.Errorf(err, "analysis failed")
				return
			}

			if len(cs) == 0 {
				aLogger.Infof("no comments were produced")
			}

			comments.Add(a.Config, cs...)
		}(name, a)
	}
	wg.Wait()

	return comments.Get()
}

func mergeSettings(global, local map[string]interface{}) map[string]interface{} {
	if local == nil {
		return global
	}

	if global == nil {
		return local
	}

	return mergeMaps(global, local)
}

func mergeMaps(global, local map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range local {
		if subMap, ok := v.(map[string]interface{}); ok {
			gv, ok := merged[k]
			if ok {
				if gvMap, ok := gv.(map[string]interface{}); ok {
					merged[k] = mergeMaps(gvMap, subMap)
					continue
				}
			}
		}
		merged[k] = v
	}

	return merged
}

func (s *Server) post(ctx context.Context, logger log.Logger, e Event, comments []AnalyzerComments) error {
	if len(comments) == 0 {
		return nil
	}
	logger.With(log.Fields{
		"comments": len(comments),
	}).Infof("posting analysis")

	if err := s.poster.Post(ctx, e, comments); err != nil {
		return err
	}

	for _, c := range comments {
		if err := s.commentOp.Save(ctx, e, c); err != nil {
			log.Errorf(err, "can't save comment")
		}
	}

	return nil
}

func (s *Server) status(ctx context.Context, logger log.Logger, e lookout.Event, st lookout.AnalysisStatus) {
	if err := s.poster.Status(ctx, e, st); err != nil {
		logger.With(log.Fields{"status": st}).Errorf(err, "posting status failed")
	}
}

type commentsList struct {
	sync.Mutex
	list []AnalyzerComments
}

func (l *commentsList) Add(conf AnalyzerConfig, cs ...*lookout.Comment) {
	l.Lock()
	l.list = append(l.list, AnalyzerComments{conf, cs})
	l.Unlock()
}

func (l *commentsList) Get() []AnalyzerComments {
	return l.list
}
