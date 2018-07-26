package json

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/pb"

	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite
}

func (s *WatcherTestSuite) SetupTest() {

}

var (
	pushJSON   = `{"event":"push", "commit_revision":{"base":{"internal_repository_url":"http://github.com/foo/bar","ReferenceName":"refs/heads/master","Hash":"hash1"},"head":{"internal_repository_url":"http://github.com/foo/bar","ReferenceName":"refs/heads/my-branch","Hash":"hash2"}}}`
	reviewJSON = `{"event":"review", "commit_revision":{"base":{"internal_repository_url":"http://github.com/foo/bar","ReferenceName":"refs/heads/master","Hash":"hash1"},"head":{"internal_repository_url":"http://github.com/foo/bar","ReferenceName":"refs/heads/my-branch","Hash":"hash2"}}}`
	badEvent   = `{"event":"none"}`
	badJSON    = `{"event":"push", { ...`
)

func (s *WatcherTestSuite) TestWatch() {
	var events int

	w, err := NewWatcher(strings.NewReader(pushJSON+"\n"+reviewJSON), &lookout.WatchOptions{})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	expectedTypes := []pb.EventType{pb.PushEventType, pb.ReviewEventType}

	err = w.Watch(ctx, func(e lookout.Event) error {
		s.Equal(expectedTypes[events], e.Type())
		s.Equal("http://github.com/foo/bar", e.Revision().Base.InternalRepositoryURL)

		events++
		return nil
	})

	s.Equal(2, events)
	s.Error(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_WrongEvent() {
	var events int

	w, err := NewWatcher(strings.NewReader(badEvent), &lookout.WatchOptions{})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++
		return nil
	})

	s.Equal(0, events)
	s.Error(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_BadJSON() {
	var events int

	w, err := NewWatcher(strings.NewReader(badEvent), &lookout.WatchOptions{})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++
		return nil
	})

	s.Equal(0, events)
	s.Error(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_WithError() {
	w, err := NewWatcher(strings.NewReader(pushJSON), &lookout.WatchOptions{})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		s.Equal(pb.PushEventType, e.Type())
		s.Equal("http://github.com/foo/bar", e.Revision().Base.InternalRepositoryURL)
		return fmt.Errorf("foo")
	})

	s.Error(err)
	s.Equal("foo", err.Error())
}

func (s *WatcherTestSuite) TearDownSuite() {
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
