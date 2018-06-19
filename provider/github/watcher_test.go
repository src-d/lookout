package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite
	mux    *http.ServeMux
	server *httptest.Server
	client *github.Client
}

func (s *WatcherTestSuite) SetupTest() {
	s.mux = http.NewServeMux()
	s.server = httptest.NewServer(s.mux)

	s.client = github.NewClient(nil)
	url, _ := url.Parse(s.server.URL + "/")
	s.client.BaseURL = url
	s.client.UploadURL = url
}

func (s *WatcherTestSuite) TestWatch() {
	s.mux.HandleFunc("/repos/mock/test/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"1", "type":"PushEvent", "payload":{"push_id": 1}}]`)
	})

	w, err := NewWatcher(&lookout.WatchOptions{
		URL: "github.com/mock/test",
	})

	w.c = s.client

	s.NoError(err)

	err = w.Watch(context.TODO(), func(e lookout.Event) error {
		s.Equal(lookout.PushEventType, e.Type())
		s.Equal("d1f57cc4e520766576c5f1d9e7655aeea5fbccfa", e.ID().String())
		return lookout.NoErrStopWatcher.New()
	})

	s.NoError(err)
}

func (s *WatcherTestSuite) TearDownSuite() {
	s.server.Close()
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
