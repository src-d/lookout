package lookout

import (
	"context"
	"fmt"
	"sync"

	"github.com/src-d/lookout/pb"
	log "gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

// AnalyzerConfig is a configuration of analyzer
type AnalyzerConfig struct {
	Name string
	// Addr is gRPC URL.
	// can be defined only in global config, repository-scoped configuration is ignored
	Addr string
	// Settings any configuration for an analyzer
	Settings map[string]interface{}
}

// ServerConfig is a server configuration
type ServerConfig struct {
	Analyzers []AnalyzerConfig
}

// Analyzer is a struct of analyzer client and config
type Analyzer struct {
	Client AnalyzerClient
	Config AnalyzerConfig
}

type reqSent func(client AnalyzerClient, settings map[string]interface{}) ([]*Comment, error)

// Server implements glue between providers / data-server / analyzers
type Server struct {
	watcher    Watcher
	poster     Poster
	fileGetter FileGetter
	analyzers  map[string]Analyzer
}

// NewServer creates new Server
func NewServer(w Watcher, p Poster, fileGetter FileGetter, analyzers map[string]Analyzer) *Server {
	return &Server{w, p, fileGetter, analyzers}
}

// Run starts server
func (s *Server) Run(ctx context.Context) error {
	// FIXME(max): we most probably want to change interface of EventHandler instead of it
	return s.watcher.Watch(ctx, func(e Event) error {
		return s.handleEvent(ctx, e)
	})
}

func (s *Server) handleEvent(ctx context.Context, e Event) error {
	switch ev := e.(type) {
	case *ReviewEvent:
		return s.HandleReview(ctx, ev)
	case *PushEvent:
		return s.HandlePush(ctx, ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

// HandleReview sends request to analyzers concurrently
func (s *Server) HandleReview(ctx context.Context, e *ReviewEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing pull request")

	if err := e.Validate(); err != nil {
		logger.Errorf(err, "processing pull request failed")
		return nil
	}

	conf, err := s.getConfig(ctx, logger, e)
	if err != nil {
		return err
	}

	send := func(a AnalyzerClient, settings map[string]interface{}) ([]*Comment, error) {
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

	s.post(ctx, logger, e, comments)
	return nil
}

// HandlePush sends request to analyzers concurrently
func (s *Server) HandlePush(ctx context.Context, e *PushEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing push")

	if err := e.Validate(); err != nil {
		logger.Errorf(err, "processing push failed")
		return nil
	}

	conf, err := s.getConfig(ctx, logger, e)
	if err != nil {
		return err
	}

	send := func(a AnalyzerClient, settings map[string]interface{}) ([]*Comment, error) {
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

	s.post(ctx, logger, e, comments)
	return nil
}

// FIXME(max): it's better to hold logger inside context
func (s *Server) getConfig(ctx context.Context, logger log.Logger, e Event) (map[string]AnalyzerConfig, error) {
	rev := e.Revision()
	scanner, err := s.fileGetter.GetFiles(ctx, &FilesRequest{
		Revision:       &rev.Head,
		IncludePattern: `^\.lookout\.yml$`,
		WantContents:   true,
	})
	if err != nil {
		return nil, err
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
func (s *Server) concurrentRequest(ctx context.Context, logger log.Logger, conf map[string]AnalyzerConfig, send reqSent) []*Comment {
	var comments commentsList

	var wg sync.WaitGroup
	for name, a := range s.analyzers {
		wg.Add(1)
		go func(name string, a AnalyzerClient) {
			defer wg.Done()

			aLogger := logger.With(log.Fields{
				"analyzer": name,
			})

			cs, err := send(a, conf[name].Settings)
			if err != nil {
				aLogger.Errorf(err, "analysis failed")
				return
			}

			if len(cs) == 0 {
				aLogger.Infof("no comments were produced")
			}

			comments.Add(cs...)
		}(name, a.Client)
	}
	wg.Wait()

	return comments.Get()
}

func (s *Server) post(ctx context.Context, logger log.Logger, e Event, comments []*Comment) {
	if len(comments) == 0 {
		return
	}
	logger.With(log.Fields{
		"comments": len(comments),
	}).Infof("posting analysis")

	if err := s.poster.Post(ctx, e, comments); err != nil {
		logger.Errorf(err, "posting analysis failed")
	}
}

type commentsList struct {
	sync.Mutex
	list []*Comment
}

func (l *commentsList) Add(cs ...*Comment) {
	l.Lock()
	l.list = append(l.list, cs...)
	l.Unlock()
}

func (l *commentsList) Get() []*Comment {
	return l.list
}
