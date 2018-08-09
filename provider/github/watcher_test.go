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

func (s *WatcherTestSuite) TestWatch() {
	var callsA, callsB, events, prEvents, pushEvents int32

	pullsHandler := func(calls *int32) func(w http.ResponseWriter, r *http.Request) {
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

	eventsHandler := func(calls *int32) func(w http.ResponseWriter, r *http.Request) {
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

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	repoURLs := []string{"github.com/mock/test-a", "github.com/mock/test-b"}
	poll := newTestPool(repoURLs, s.githubURL, s.cache)
	w, err := NewWatcher(poll, &lookout.WatchOptions{
		URLs: repoURLs,
	})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++

		switch e.Type() {
		case pb.ReviewEventType:
			prEvents++
			s.Equal("fd84071093b69f9aac25fb5dfeea1a870e3e19cf", e.ID().String())
		case pb.PushEventType:
			pushEvents++
			s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
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

func (s *WatcherTestSuite) TestWatch_WithError() {
	s.mux.HandleFunc("/repos/mock/test/pulls", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":1}]`)
	})
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	})

	repoURLs := []string{"github.com/mock/test"}
	poll := newTestPool(repoURLs, s.githubURL, s.cache)
	w, err := NewWatcher(poll, &lookout.WatchOptions{
		URLs: repoURLs,
	})

	s.NoError(err)

	err = w.Watch(context.TODO(), func(e lookout.Event) error {
		s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())

		return fmt.Errorf("foo")
	})

	s.EqualError(err, "foo")
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

	repoURLs := []string{"github.com/mock/test"}
	poll := newTestPool(repoURLs, s.githubURL, s.cache)
	w, err := NewWatcher(poll, &lookout.WatchOptions{
		URLs: repoURLs,
	})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		prEvents++
		s.Equal("02b508226b9c2f38be7d589fe765a119ddf4452b", e.ID().String())

		return nil
	})

	s.Equal(1, atomic.LoadInt32(&calls))
	s.Equal(1, prEvents)
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

func newTestPool(repoURLs []string, githubURL *url.URL, cache *cache.ValidableCache) *ClientPool {
	client := NewClient(nil, cache, log.New(log.Fields{}))
	client.BaseURL = githubURL
	client.UploadURL = githubURL

	byClients := map[*Client][]*lookout.RepositoryInfo{
		client: []*lookout.RepositoryInfo{},
	}
	byRepo := make(map[string]*Client, len(repoURLs))

	for _, url := range repoURLs {
		repo, err := vcsurl.Parse(url)
		if err != nil {
			panic(err)
		}

		byClients[client] = append(byClients[client], repo)
		byRepo[repo.FullName] = client
	}

	return &ClientPool{
		byClients: byClients,
		byRepo:    byRepo,
	}
}
