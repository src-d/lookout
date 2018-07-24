package lookout

import (
	"context"
	"sync"

	log "gopkg.in/src-d/go-log.v1"
)

type reqSent func(AnalyzerClient) ([]*Comment, error)

// Server implements glue between providers / data-server / analyzers
type Server struct {
	watcher    Watcher
	poster     Poster
	fileGetter FileGetter
	analyzers  map[string]AnalyzerClient
}

// NewServer creates new Server
func NewServer(w Watcher, p Poster, fileGetter FileGetter, analyzers map[string]AnalyzerClient) *Server {
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

	send := func(a AnalyzerClient) ([]*Comment, error) {
		resp, err := a.NotifyReviewEvent(ctx, e)
		if err != nil {
			return nil, err
		}
		return resp.Comments, nil
	}
	comments := s.concurrentRequest(ctx, logger, send)

	s.post(ctx, logger, e, comments.Get())
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

	send := func(a AnalyzerClient) ([]*Comment, error) {
		resp, err := a.NotifyPushEvent(ctx, e)
		if err != nil {
			return nil, err
		}
		return resp.Comments, nil
	}
	comments := s.concurrentRequest(ctx, logger, send)

	s.post(ctx, logger, e, comments.Get())
	return nil
}

// FIXME(max): it's better to hold logger inside context
func (s *Server) concurrentRequest(ctx context.Context, logger log.Logger, send reqSent) commentsList {
	var comments commentsList

	var wg sync.WaitGroup
	for name, a := range s.analyzers {
		wg.Add(1)
		go func(name string, a AnalyzerClient) {
			defer wg.Done()

			aLogger := logger.With(log.Fields{
				"analyzer": name,
			})

			cs, err := send(a)
			if err != nil {
				aLogger.Errorf(err, "analysis failed")
				return
			}

			if len(cs) == 0 {
				aLogger.Infof("no comments were produced")
			}

			comments.Add(cs...)
		}(name, a)
	}
	wg.Wait()

	return comments
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
