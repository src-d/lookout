package store

import (
	"context"

	"github.com/src-d/lookout"
)

type EventStatus string

const (
	EventStatusNew       = EventStatus("new")
	EventStatusProcessed = EventStatus("processed")
	EventStatusFailed    = EventStatus("failed")
)

type EventOperator interface {
	Save(context.Context, lookout.Event) (EventStatus, error)
	UpdateStatus(context.Context, lookout.Event, EventStatus) error
}

// NoopEventOperator satisfies EventOperator interface but does nothing
type NoopEventOperator struct{}

var _ EventOperator = &NoopEventOperator{}

func (o *NoopEventOperator) Save(context.Context, lookout.Event) (EventStatus, error) {
	return EventStatusNew, nil
}

func (o *NoopEventOperator) UpdateStatus(context.Context, lookout.Event, EventStatus) error {
	return nil
}
