package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/pb"
	"github.com/src-d/lookout/util/cache"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/stretchr/testify/suite"
)

func init() {
	// make everything faster for tests
	minInterval = time.Millisecond
}

type WatcherTestSuite struct {
	suite.Suite
	mux    *http.ServeMux
	server *httptest.Server
	client *github.Client
	cache  *cache.ValidableCache
}

func (s *WatcherTestSuite) SetupTest() {
	s.mux = http.NewServeMux()
	s.server = httptest.NewServer(s.mux)

	s.cache = cache.NewValidableCache(httpcache.NewMemoryCache())
	s.client = github.NewClient(&http.Client{
		Transport: httpcache.NewTransport(s.cache),
	})

	url, _ := url.Parse(s.server.URL + "/")
	s.client.BaseURL = url
	s.client.UploadURL = url
}

func (s *WatcherTestSuite) TestWatch() {
	var callsA, callsB, events, prEvents, pushEvents int

	pullsHandler := func(calls *int) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			*calls++
			etag := "124567"
			if r.Header.Get("if-none-match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("etag", etag)
			fmt.Fprint(w, `[{"id":5}]`)
		}
	}

	eventsHandler := func(calls *int) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			*calls++
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

	w, err := NewWatcher(&NoopTransport{}, &lookout.WatchOptions{
		URLs: []string{"github.com/mock/test-a", "github.com/mock/test-b"},
	})

	w.clients["mock/test-a"] = s.client
	w.clients["mock/test-b"] = s.client
	w.cache = s.cache

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++

		switch e.Type() {
		case pb.ReviewEventType:
			prEvents++
			s.Equal("02b508226b9c2f38be7d589fe765a119ddf4452b", e.ID().String())
		case pb.PushEventType:
			pushEvents++
			s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
		}

		return nil
	})

	s.True(callsA > 2)
	s.True(callsB > 2)
	s.Equal(4, events)
	s.Equal(2, prEvents)
	s.Equal(2, pushEvents)
	s.EqualError(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_WithError() {
	s.mux.HandleFunc("/repos/mock/test/pulls", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":1}]`)
	})
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	})

	w, err := NewWatcher(&NoopTransport{}, &lookout.WatchOptions{
		URLs: []string{"github.com/mock/test"},
	})

	w.clients["mock/test"] = s.client
	w.cache = s.cache

	s.NoError(err)

	err = w.Watch(context.TODO(), func(e lookout.Event) error {
		switch e.Type() {
		case pb.ReviewEventType:
			s.Equal("96de6d9d321aa94faa0f053c72e2684a44961e5b", e.ID().String())
		case pb.PushEventType:
			s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
		}
		return fmt.Errorf("foo")
	})

	s.Error(err, "foo")
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
