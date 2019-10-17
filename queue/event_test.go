package queue

import (
	"testing"

	"github.com/src-d/lookout"
	lookout_mock "github.com/src-d/lookout/mock"
	"github.com/stretchr/testify/suite"
)

type EventSuite struct {
	suite.Suite
}

func TestEventSuite(t *testing.T) {
	suite.Run(t, new(EventSuite))
}

func (s *EventSuite) TestReviewEvent() {
	require := s.Require()

	le := &lookout.ReviewEvent{}
	le.InternalID = "test"
	e, err := NewEvent(le)
	require.NoError(err)

	ue, err := e.ToInterface()
	require.NoError(err)
	require.Equal(le, ue)
}

func (s *EventSuite) TestPushEvent() {
	require := s.Require()

	le := &lookout.PushEvent{}
	le.InternalID = "test"
	e, err := NewEvent(le)
	require.NoError(err)

	ue, err := e.ToInterface()
	require.NoError(err)
	require.Equal(le, ue)
}

func (s *EventSuite) TestUnknownEvent() {
	require := s.Require()

	le := &lookout_mock.FakeEvent{}
	e, err := NewEvent(le)
	require.EqualError(err, "unsupported event type *mock.FakeEvent")
	require.Nil(e)

	e = &Event{
		EventType: 0,
	}
	ue, err := e.ToInterface()
	require.EqualError(err, "unknown lookout event")
	require.Nil(ue)
}

func (s *EventSuite) TestEmptyEvent() {
	require := s.Require()

	var le lookout.Event
	e, err := NewEvent(le)
	require.EqualError(err, "nil event isn't supported")
	require.Nil(e)

	e = &Event{}
	ue, err := e.ToInterface()
	require.EqualError(err, "unknown lookout event")
	require.Nil(ue)
}
