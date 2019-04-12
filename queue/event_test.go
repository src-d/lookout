package queue

import (
	"testing"

	"github.com/src-d/lookout"
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

	le := &unknownEvent{}
	e, err := NewEvent(le)
	require.EqualError(err, "unsupported event type *queue.unknownEvent")
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

type unknownEvent struct{ lookout.Event }

func (e *unknownEvent) Type() lookout.EventType { return 0 }
