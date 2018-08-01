package store

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/store/models"
)

// EventOperator manages persistence of Events
type EventOperator interface {
	// Save persists Event in a store and returns Status if event was persisted already
	Save(context.Context, lookout.Event) (models.EventStatus, error)
	// UpdateStatus updates Status of event in a store
	UpdateStatus(context.Context, lookout.Event, models.EventStatus) error
}

// NoopEventOperator satisfies EventOperator interface but does nothing
type NoopEventOperator struct{}

var _ EventOperator = &NoopEventOperator{}

// Save implements EventOperator interface and always returns New status
func (o *NoopEventOperator) Save(context.Context, lookout.Event) (models.EventStatus, error) {
	return models.EventStatusNew, nil
}

// UpdateStatus implements EventOperator interface and does nothing
func (o *NoopEventOperator) UpdateStatus(context.Context, lookout.Event, models.EventStatus) error {
	return nil
}
