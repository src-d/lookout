package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/pb"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/store/models"
	"github.com/src-d/lookout/util/ctxlog"

	log "gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

// Config is a server configuration
type Config struct {
	Analyzers []lookout.AnalyzerConfig
}

type reqSent func(client lookout.AnalyzerClient, settings map[string]interface{}) ([]*lookout.Comment, error)

// Server implements glue between providers / data-server / analyzers
type Server struct {
	watcher    lookout.Watcher
	poster     lookout.Poster
	fileGetter lookout.FileGetter
	analyzers  map[string]lookout.Analyzer
	eventOp    store.EventOperator
	commentOp  store.CommentOperator
}

// NewServer creates new Server
func NewServer(w lookout.Watcher, p lookout.Poster, fileGetter lookout.FileGetter,
	analyzers map[string]lookout.Analyzer, eventOp store.EventOperator, commentOp store.CommentOperator) *Server {
	return &Server{w, p, fileGetter, analyzers, eventOp, commentOp}
}

// Run starts server
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	for {
		go func() {
			// FIXME(max): we most probably want to change interface of EventHandler instead of it
			err := s.watcher.Watch(ctx, func(e lookout.Event) error {
				return s.handleEvent(ctx, e)
			})
			errCh <- err
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "error during watching, restart watching")
			} else {
				return nil
			}
		}
	}
}

func (s *Server) handleEvent(ctx context.Context, e lookout.Event) error {
	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{
		"event-type": e.Type(),
		"event-id":   e.ID().String(),
	})

	status, err := s.eventOp.Save(ctx, e)
	if err != nil {
		logger.Errorf(err, "can't save event to database")
		return err
	}

	if status == models.EventStatusProcessed {
		logger.Infof("event successfully processed, skipping...")
		return nil
	}

	// TODO(max): we need some retry policy here depends on errors
	if status == models.EventStatusFailed {
		logger.Infof("event processing failed, skipping...")
		return nil
	}

	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		err = s.HandleReview(ctx, ev)
	case *lookout.PushEvent:
		err = s.HandlePush(ctx, ev)
	default:
		logger.Debugf("ignoring unsupported event: %s", ev)
	}

	if err == nil {
		status = models.EventStatusProcessed
	} else {
		logger.Errorf(err, "event processing failed")
		status = models.EventStatusFailed
	}

	if updateErr := s.eventOp.UpdateStatus(ctx, e, status); updateErr != nil {
		logger.Errorf(updateErr, "can't update status in database")
	}

	// don't fail on event processing error, just skip it
	return nil
}

// HandleReview sends request to analyzers concurrently
func (s *Server) HandleReview(ctx context.Context, e *lookout.ReviewEvent) error {
	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing pull request")

	if err := e.Validate(); err != nil {
		return err
	}

	conf, err := s.getConfig(ctx, e)
	if err != nil {
		return err
	}

	s.status(ctx, e, lookout.PendingAnalysisStatus)

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
	comments := s.concurrentRequest(ctx, conf, send)

	if err := s.post(ctx, e, comments); err != nil {
		s.status(ctx, e, lookout.ErrorAnalysisStatus)
		return fmt.Errorf("posting analysis failed: %s", err)
	}

	s.status(ctx, e, lookout.SuccessAnalysisStatus)

	return nil
}

// HandlePush sends request to analyzers concurrently
func (s *Server) HandlePush(ctx context.Context, e *lookout.PushEvent) error {
	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing push")

	if err := e.Validate(); err != nil {
		return err
	}

	conf, err := s.getConfig(ctx, e)
	if err != nil {
		return err
	}

	s.status(ctx, e, lookout.PendingAnalysisStatus)

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
	comments := s.concurrentRequest(ctx, conf, send)

	if err := s.post(ctx, e, comments); err != nil {
		s.status(ctx, e, lookout.ErrorAnalysisStatus)
		return fmt.Errorf("posting analysis failed: %s", err)
	}
	s.status(ctx, e, lookout.SuccessAnalysisStatus)

	return nil
}

func (s *Server) getConfig(ctx context.Context, e lookout.Event) (map[string]lookout.AnalyzerConfig, error) {
	rev := e.Revision()
	ctxlog.Get(ctx).Infof("getting .lookout.yml")
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
		ctxlog.Get(ctx).Infof("repository config is not found")
		return nil, nil
	}

	var conf Config
	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return nil, fmt.Errorf("Can't parse configuration file: %s", err)
	}

	res := make(map[string]lookout.AnalyzerConfig, len(s.analyzers))
	for name, a := range s.analyzers {
		res[name] = a.Config
	}
	for _, aConf := range conf.Analyzers {
		if _, ok := s.analyzers[aConf.Name]; !ok {
			ctxlog.Get(ctx).Warningf("analyzer '%s' required by local config isn't enabled on server", aConf.Name)
			continue
		}
		res[aConf.Name] = aConf
	}

	return res, nil
}

func (s *Server) concurrentRequest(ctx context.Context, conf map[string]lookout.AnalyzerConfig, send reqSent) []lookout.AnalyzerComments {
	var comments commentsList

	var wg sync.WaitGroup
	for name, a := range s.analyzers {
		if a.Config.Disabled || conf[name].Disabled {
			ctxlog.Get(ctx).Infof("analyzer %s disabled by local .lookout.yml", name)
			continue
		}

		wg.Add(1)
		go func(name string, a lookout.Analyzer) {
			defer wg.Done()

			aLogger := ctxlog.Get(ctx).With(log.Fields{
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

func (s *Server) post(ctx context.Context, e lookout.Event, comments []lookout.AnalyzerComments) error {
	var filtered []lookout.AnalyzerComments
	for _, cg := range comments {
		var filteredComments []*lookout.Comment
		for _, c := range cg.Comments {
			yes, err := s.commentOp.Posted(ctx, e, c)
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "comment posted check failed")
			}
			if yes {
				continue
			}
			filteredComments = append(filteredComments, c)
		}
		if len(filteredComments) > 0 {
			filtered = append(filtered, lookout.AnalyzerComments{
				Config:   cg.Config,
				Comments: filteredComments,
			})
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	ctxlog.Get(ctx).With(log.Fields{
		"comments": len(comments),
	}).Infof("posting analysis")

	if err := s.poster.Post(ctx, e, comments); err != nil {
		return err
	}

	for _, cg := range comments {
		for _, c := range cg.Comments {
			if err := s.commentOp.Save(ctx, e, c); err != nil {
				ctxlog.Get(ctx).Errorf(err, "can't save comment")
			}
		}
	}

	return nil
}

func (s *Server) status(ctx context.Context, e lookout.Event, st lookout.AnalysisStatus) {
	if err := s.poster.Status(ctx, e, st); err != nil {
		ctxlog.Get(ctx).With(log.Fields{"status": st}).Errorf(err, "posting status failed")
	}
}

type commentsList struct {
	sync.Mutex
	list []lookout.AnalyzerComments
}

func (l *commentsList) Add(conf lookout.AnalyzerConfig, cs ...*lookout.Comment) {
	l.Lock()
	l.list = append(l.list, lookout.AnalyzerComments{conf, cs})
	l.Unlock()
}

func (l *commentsList) Get() []lookout.AnalyzerComments {
	return l.list
}

type LogPoster struct {
	Log log.Logger
}

func (p *LogPoster) Post(ctx context.Context, e lookout.Event,
	aCommentsList []lookout.AnalyzerComments) error {
	for _, aComments := range aCommentsList {
		for _, c := range aComments.Comments {
			logger := p.Log.With(log.Fields{
				"text": c.Text,
			})
			if c.File == "" {
				logger.Infof("global comment")
				continue
			}

			logger = logger.With(log.Fields{"file": c.File})
			if c.Line == 0 {
				logger.Infof("file comment")
				continue
			}

			logger.With(log.Fields{"line": c.Line}).Infof("line comment")
		}
	}

	return nil
}

func (p *LogPoster) Status(ctx context.Context, e lookout.Event,
	status lookout.AnalysisStatus) error {
	p.Log.Infof("status: %s", status)
	return nil
}
