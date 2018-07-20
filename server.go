package lookout

import (
	"context"
	"sync"

	log "gopkg.in/src-d/go-log.v1"
)

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
		return s.HandlePR(ctx, ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

// HandlePR sends request to analyzers concurrently
func (s *Server) HandlePR(ctx context.Context, e *ReviewEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing pull request")

	var comments commentsList

	var wg sync.WaitGroup
	for name, a := range s.analyzers {
		wg.Add(1)
		go func(name string, a AnalyzerClient) {
			defer wg.Done()

			aLogger := logger.With(log.Fields{
				"analyzer": name,
			})

			resp, err := a.NotifyReviewEvent(context.TODO(), e)
			if err != nil {
				aLogger.Errorf(err, "analysis failed")
				return
			}

			if len(resp.Comments) == 0 {
				aLogger.Infof("no comments were produced")
			}

			comments.Add(resp.Comments...)
		}(name, a)
	}
	wg.Wait()

	if !comments.Empty() {
		logger.With(log.Fields{
			"comments": comments.Len(),
		}).Infof("posting analysis")

		if err := s.poster.Post(ctx, e, comments.Get()); err != nil {
			logger.Errorf(err, "posting analysis failed")
		}
	}

	return nil
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

func (l *commentsList) Len() int {
	return len(l.list)
}

func (l *commentsList) Empty() bool {
	return len(l.list) == 0
}
