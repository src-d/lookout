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
	var calls, events int
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		calls++
		etag := "124567"
		if r.Header.Get("if-none-match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("etag", etag)
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	})

	w, err := NewWatcher(nil, &lookout.WatchOptions{
		URL: "github.com/mock/test",
	})

	w.c = s.client
	w.cache = s.cache

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++

		s.Equal(pb.PushEventType, e.Type())
		s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
		return nil
	})

	s.True(calls > 1)
	s.Equal(1, events)
	s.Error(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_WithError() {
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	})

	w, err := NewWatcher(nil, &lookout.WatchOptions{
		URL: "github.com/mock/test",
	})

	w.c = s.client
	w.cache = s.cache

	s.NoError(err)

	err = w.Watch(context.TODO(), func(e lookout.Event) error {
		s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
		return fmt.Errorf("foo")
	})

	s.Error(err, "foo")

	err = w.Watch(context.TODO(), func(e lookout.Event) error {
		s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
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
