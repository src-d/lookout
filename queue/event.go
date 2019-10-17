package queue

import (
	"fmt"
	"reflect"

	"github.com/src-d/lookout"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

// Event allows to marshal and unmarshal lookout.Event interface
type Event struct {
	EventType lookout.EventType
	// Exported so go-queue marshals it, but should be accessed through ToInterface()
	ReviewEvent *lookout.ReviewEvent
	// Exported so go-queue marshals it, but should be accessed through ToInterface()
	PushEvent *lookout.PushEvent
}

// NewEvent creates new marshallable event from lookout.Event
func NewEvent(ev lookout.Event) (*Event, error) {
	if ev == nil {
		return nil, fmt.Errorf("nil event isn't supported")
	}

	e := &Event{}
	e.EventType = ev.Type()

	switch evt := ev.(type) {
	case *lookout.ReviewEvent:
		e.ReviewEvent = evt
	case *lookout.PushEvent:
		e.PushEvent = evt
	default:
		return nil, fmt.Errorf("unsupported event type %s", reflect.TypeOf(ev).String())
	}

	return e, nil
}

// ToInterface return lookout.Event
func (e Event) ToInterface() (lookout.Event, error) {
	switch e.EventType {
	case pb.PushEventType:
		return e.PushEvent, nil
	case pb.ReviewEventType:
		return e.ReviewEvent, nil
	}

	return nil, fmt.Errorf("unknown lookout event")
}
