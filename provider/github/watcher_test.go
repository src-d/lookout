package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/pb"
	"github.com/src-d/lookout/util/cache"

	"github.com/gregjones/httpcache"
	"github.com/stretchr/testify/suite"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	// make everything faster for tests
	minInterval = time.Millisecond
	log.DefaultLogger = log.New(log.Fields{"app": "lookout"})
}

type WatcherTestSuite struct {
	suite.Suite
	mux       *http.ServeMux
	server    *httptest.Server
	githubURL *url.URL
	cache     *cache.ValidableCache
}

func (s *WatcherTestSuite) SetupTest() {
	s.mux = http.NewServeMux()
	s.server = httptest.NewServer(s.mux)

	s.cache = cache.NewValidableCache(httpcache.NewMemoryCache())
	s.githubURL, _ = url.Parse(s.server.URL + "/")
}

var pullsHandler = func(calls *int32) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)

		etag := "124567"
		if r.Header.Get("if-none-match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("etag", etag)
		fmt.Fprint(w, `[{"id":5}]`)
	}
}

var eventsHandler = func(calls *int32) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)

		etag := "124567"
		if r.Header.Get("if-none-match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("etag", etag)
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	}
}

const (
	pullID = "fd84071093b69f9aac25fb5dfeea1a870e3e19cf"
	pushID = "d1f57cc4e520766576c5f1d9e7655aeea5fbccfa"
)

func (s *WatcherTestSuite) newWatcher(repoURLs []string) *Watcher {
	pool := newTestPool(s.Suite, repoURLs, s.githubURL, s.cache)
	w, err := NewWatcher(pool, &lookout.WatchOptions{
		URLs: repoURLs,
	})

	s.NoError(err)

	return w
}

func (s *WatcherTestSuite) TestWatch() {
	var callsA, callsB, events, prEvents, pushEvents int32

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test-a", "github.com/mock/test-b"})
	err := w.Watch(ctx, func(e lookout.Event) error {
		atomic.AddInt32(&events, 1)

		switch e.Type() {
		case pb.ReviewEventType:
			prEvents++
			s.Equal(pullID, e.ID().String())
		case pb.PushEventType:
			pushEvents++
			s.Equal(pushID, e.ID().String())
		}

		return nil
	})

	s.True(atomic.LoadInt32(&callsA) > 2)
	s.True(atomic.LoadInt32(&callsB) > 2)
	s.EqualValues(4, events)
	s.EqualValues(2, prEvents)
	s.EqualValues(2, pushEvents)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_CallbackError() {
	var calls int32

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	w := s.newWatcher([]string{"github.com/mock/test"})
	err := w.Watch(context.TODO(), func(e lookout.Event) error {
		switch e.Type() {
		case pb.ReviewEventType:
			s.Equal(pullID, e.ID().String())
		case pb.PushEventType:
			s.Equal(pushID, e.ID().String())
		}

		return fmt.Errorf("foo")
	})

	s.EqualError(err, "foo")
}

func (s *WatcherTestSuite) TestWatch_HttpError() {
	var calls, callsErr int32

	errCodeHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callsErr, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	s.mux.HandleFunc("/repos/mock/test-err/pulls", errCodeHandler)
	s.mux.HandleFunc("/repos/mock/test-err/events", errCodeHandler)

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test", "github.com/mock/test-err"})
	err := w.Watch(ctx, func(e lookout.Event) error {
		s.Equal("git://github.com/mock/test.git", e.Revision().Head.InternalRepositoryURL)
		return nil
	})

	s.True(atomic.LoadInt32(&calls) > 1)
	s.True(atomic.LoadInt32(&callsErr) > 1)

	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_HttpTimeout() {
	var calls, callsErr int32

	// Change the request timeout for this test
	prevRequestTimeout := RequestTimeout
	RequestTimeout = 5 * minInterval
	defer func() { RequestTimeout = prevRequestTimeout }()

	errCodeHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callsErr, 1)
		time.Sleep(RequestTimeout * 2)
	}

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	s.mux.HandleFunc("/repos/mock/test-err/pulls", errCodeHandler)
	s.mux.HandleFunc("/repos/mock/test-err/events", errCodeHandler)

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test", "github.com/mock/test-err"})
	err := w.Watch(ctx, func(e lookout.Event) error {
		s.Equal("git://github.com/mock/test.git", e.Revision().Head.InternalRepositoryURL)
		return nil
	})

	s.True(atomic.LoadInt32(&calls) > 1)
	s.True(atomic.LoadInt32(&callsErr) > 1)

	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_JSONError() {
	var calls, callsErr int32

	errCodeHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callsErr, 1)
		fmt.Fprint(w, `[{"key":"value", "broken json!`)
	}

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	s.mux.HandleFunc("/repos/mock/test-err/pulls", errCodeHandler)
	s.mux.HandleFunc("/repos/mock/test-err/events", errCodeHandler)

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test", "github.com/mock/test-err"})
	err := w.Watch(ctx, func(e lookout.Event) error {
		return nil
	})

	s.True(atomic.LoadInt32(&calls) > 1)
	s.True(atomic.LoadInt32(&callsErr) > 1)

	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatchLimit() {
	var calls, prEvents int32

	reset := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)

	pullsHandler := func(calls *int32) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(calls, 1)

			w.Header().Set(headerRateReset, reset)
			w.Header().Set(headerRateRemaining, "1")
			fmt.Fprint(w, `[{"id":5}]`)
		}
	}
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerRateRemaining, "0")
		fmt.Fprint(w, `[]`)
	})

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test"})
	err := w.Watch(ctx, func(e lookout.Event) error {
		prEvents++
		s.Equal("fd84071093b69f9aac25fb5dfeea1a870e3e19cf", e.ID().String())

		return nil
	})

	s.EqualValues(1, atomic.LoadInt32(&calls))
	s.EqualValues(1, prEvents)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TearDownSuite() {
	s.server.Close()
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

type NoopTransport struct{}

func (t *NoopTransport) Get(repo string) http.RoundTripper {
	return nil
}

func newTestPool(s suite.Suite, repoURLs []string, githubURL *url.URL, cache *cache.ValidableCache) *ClientPool {
	client := NewClient(nil, cache, log.New(log.Fields{}))
	client.BaseURL = githubURL
	client.UploadURL = githubURL

	byClients := map[*Client][]*lookout.RepositoryInfo{
		client: []*lookout.RepositoryInfo{},
	}
	byRepo := make(map[string]*Client, len(repoURLs))

	for _, url := range repoURLs {
		repo, err := vcsurl.Parse(url)
		s.NoError(err)

		byClients[client] = append(byClients[client], repo)
		byRepo[repo.FullName] = client
	}

	return &ClientPool{
		byClients: byClients,
		byRepo:    byRepo,
	}
}
