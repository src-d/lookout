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
	"github.com/src-d/lookout/util/cache"

	"github.com/gregjones/httpcache"
	"github.com/stretchr/testify/suite"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

func init() {
	// make everything faster for tests
	minInterval = 10 * time.Millisecond
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

var emptyArrayHandler = func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `[]`)
}

const (
	pullID = "fd84071093b69f9aac25fb5dfeea1a870e3e19cf"
	pushID = "d1f57cc4e520766576c5f1d9e7655aeea5fbccfa"
)

func (s *WatcherTestSuite) newWatcher(repoURLs []string) *Watcher {
	pool := newTestPool(s.Suite, repoURLs, s.githubURL, s.cache)
	w, err := NewWatcher(pool)

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
	err := w.Watch(ctx, func(ctx context.Context, e lookout.Event) error {
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

func (s *WatcherTestSuite) TestWatch_CallbackError_Pull() {
	var calls int32

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", emptyArrayHandler)

	w := s.newWatcher([]string{"github.com/mock/test"})
	err := w.Watch(context.TODO(), func(ctx context.Context, e lookout.Event) error {
		s.Equal(pb.ReviewEventType, e.Type())
		s.Equal(pullID, e.ID().String())

		return fmt.Errorf("foo")
	})

	s.EqualError(err, "foo")
}

func (s *WatcherTestSuite) TestWatch_CallbackError_Event() {
	var calls int32

	s.mux.HandleFunc("/repos/mock/test/pulls", emptyArrayHandler)
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	w := s.newWatcher([]string{"github.com/mock/test"})
	err := w.Watch(context.TODO(), func(ctx context.Context, e lookout.Event) error {
		s.Equal(pb.PushEventType, e.Type())
		s.Equal(pushID, e.ID().String())

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
	err := w.Watch(ctx, func(ctx context.Context, e lookout.Event) error {
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
	sleepTime := RequestTimeout * 2
	defer func() { RequestTimeout = prevRequestTimeout }()

	errCodeHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callsErr, 1)
		time.Sleep(sleepTime)
	}

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&calls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&calls))

	s.mux.HandleFunc("/repos/mock/test-err/pulls", errCodeHandler)
	s.mux.HandleFunc("/repos/mock/test-err/events", errCodeHandler)

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test", "github.com/mock/test-err"})
	err := w.Watch(ctx, func(ctx context.Context, e lookout.Event) error {
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
	err := w.Watch(ctx, func(ctx context.Context, e lookout.Event) error {
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
	err := w.Watch(ctx, func(ctx context.Context, e lookout.Event) error {
		prEvents++
		s.Equal("fd84071093b69f9aac25fb5dfeea1a870e3e19cf", e.ID().String())

		return nil
	})

	s.EqualValues(1, atomic.LoadInt32(&calls))
	s.EqualValues(1, prEvents)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestCustomMinInterval() {
	var pullCalls, eventCalls int32

	s.mux.HandleFunc("/repos/mock/test/pulls", pullsHandler(&pullCalls))
	s.mux.HandleFunc("/repos/mock/test/events", eventsHandler(&eventCalls))

	clientMinInterval := 200 * time.Millisecond
	client := NewClient(nil, s.cache, clientMinInterval.String())
	client.BaseURL = s.githubURL
	client.UploadURL = s.githubURL

	repo, _ := vcsurl.Parse("github.com/mock/test")

	pool := &ClientPool{
		Changes: make(chan ClientPoolEvent),
		byClients: map[*Client][]*lookout.RepositoryInfo{
			client: []*lookout.RepositoryInfo{repo},
		},
		byRepo: map[string]*Client{"mock/test": client},
	}

	w, err := NewWatcher(pool)
	s.NoError(err)

	globalTimeout := clientMinInterval * 3
	ctx, cancel := context.WithTimeout(context.TODO(), globalTimeout)
	defer cancel()

	err = w.Watch(ctx, func(ctx context.Context, e lookout.Event) error { return nil })

	s.EqualValues(globalTimeout/clientMinInterval, atomic.LoadInt32(&pullCalls))
	s.EqualValues(globalTimeout/clientMinInterval, atomic.LoadInt32(&eventCalls))
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestAddRepo() {
	var callsA, callsB int32

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test-a"})
	// add new repo
	go func() {
		time.Sleep(minInterval) // make sure we add new repo after some calls were done already
		c, _ := w.pool.Client("mock", "test-a")
		repo, _ := vcsurl.Parse("github.com/mock/test-b")
		w.pool.Update(c, append(w.pool.ReposByClient(c), repo))
	}()

	err := w.Watch(ctx, func(e lookout.Event) error {
		return nil
	})

	s.True(atomic.LoadInt32(&callsA) > 2)
	s.True(atomic.LoadInt32(&callsB) > 2)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestRemoveRepo() {
	var callsA, callsB int32

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*20)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test-a", "github.com/mock/test-b"})
	// remove repo
	go func() {
		time.Sleep(minInterval * 5) // make sure we add new repo after some calls were done already
		c, _ := w.pool.Client("mock", "test-a")
		var repos []*lookout.RepositoryInfo
		for _, r := range w.pool.ReposByClient(c) {
			if r.FullName == "mock/test-a" {
				repos = append(repos, r)
				break
			}
		}
		w.pool.Update(c, repos)
	}()

	err := w.Watch(ctx, func(e lookout.Event) error {
		return nil
	})

	s.True(atomic.LoadInt32(&callsA) > 10) // check that watching didn't stop
	s.True(atomic.LoadInt32(&callsB) > 2)  // check that calls were made
	s.True(atomic.LoadInt32(&callsB) < 10) // check that watching did stop
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestAddClient() {
	var callsA, callsB int32

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*10)
	defer cancel()

	w := s.newWatcher([]string{"github.com/mock/test-a"})
	// add new client
	go func() {
		time.Sleep(minInterval) // make sure we add new repo after some calls were done already
		c := newClient(s.githubURL, s.cache)
		repo, _ := vcsurl.Parse("github.com/mock/test-b")
		w.pool.Update(c, []*lookout.RepositoryInfo{repo})
	}()

	err := w.Watch(ctx, func(e lookout.Event) error {
		return nil
	})

	s.True(atomic.LoadInt32(&callsA) > 2)
	s.True(atomic.LoadInt32(&callsB) > 2)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestRemoveClient() {
	var callsA, callsB int32

	s.mux.HandleFunc("/repos/mock/test-a/pulls", pullsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-a/events", eventsHandler(&callsA))
	s.mux.HandleFunc("/repos/mock/test-b/pulls", pullsHandler(&callsB))
	s.mux.HandleFunc("/repos/mock/test-b/events", eventsHandler(&callsB))

	ctx, cancel := context.WithTimeout(context.TODO(), minInterval*20)
	defer cancel()

	// construct watcher
	repo1, _ := vcsurl.Parse("github.com/mock/test-a")
	repo2, _ := vcsurl.Parse("github.com/mock/test-b")

	client1 := newClient(s.githubURL, s.cache)
	client2 := newClient(s.githubURL, s.cache)
	byClients := map[*Client][]*lookout.RepositoryInfo{
		client1: []*lookout.RepositoryInfo{repo1},
		client2: []*lookout.RepositoryInfo{repo2},
	}
	byRepo := map[string]*Client{
		repo1.FullName: client1,
		repo2.FullName: client2,
	}

	pool := &ClientPool{
		Changes:   make(chan ClientPoolEvent),
		byClients: byClients,
		byRepo:    byRepo,
	}

	w, _ := NewWatcher(pool)

	// remove client
	go func() {
		time.Sleep(minInterval * 5) // make sure we add new repo after some calls were done already
		pool.RemoveClient(client2)
	}()

	err := w.Watch(ctx, func(e lookout.Event) error {
		return nil
	})

	s.True(atomic.LoadInt32(&callsA) > 10) // check that we didn't stop watching
	s.True(atomic.LoadInt32(&callsB) > 2)  // check that we did some calls
	s.True(atomic.LoadInt32(&callsB) < 20) // check that we stopped watching
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

func newClient(githubURL *url.URL, cache *cache.ValidableCache) *Client {
	client := NewClient(nil, cache, "")
	client.BaseURL = githubURL
	client.UploadURL = githubURL
	return client
}

func newTestPool(s suite.Suite, repoURLs []string, githubURL *url.URL, cache *cache.ValidableCache) *ClientPool {
	client := newClient(githubURL, cache)

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
		Changes:   make(chan ClientPoolEvent),
		byClients: byClients,
		byRepo:    byRepo,
	}
}
