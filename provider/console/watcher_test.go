package console

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

var pushStr = "PUSH http://github.com/foo/bar master hash1 my-branch hash2"
var badStr = "NOPE foo bar"

func (s *WatcherTestSuite) TestWatch() {
	var events int

	w, err := NewWatcher(&WatchOptions{
		Reader: strings.NewReader(pushStr),
	})

	s.NoError(err)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err = w.Watch(ctx, func(e lookout.Event) error {
		events++

		s.Equal(pb.PushEventType, e.Type())
		s.Equal("http://github.com/foo/bar", e.Revision().Base.InternalRepositoryURL)
		return nil
	})

	s.Equal(1, events)
	s.Error(err, "context deadline exceeded")
}

func (s *WatcherTestSuite) TestWatch_WrongEvent() {
	var events int

	w, err := NewWatcher(&WatchOptions{
		Reader: strings.NewReader(badStr),
	})

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
	w, err := NewWatcher(&WatchOptions{
		Reader: strings.NewReader(pushStr),
	})

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
