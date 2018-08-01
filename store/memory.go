package store

import (
	"context"
	"errors"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/store/models"
)

// MemEventOperator satisfies EventOperator interface keeps events in memory
type MemEventOperator struct {
	events map[string]models.EventStatus
}

// NewMemEventOperator creates new MemEventOperator
func NewMemEventOperator() *MemEventOperator {
	return &MemEventOperator{events: make(map[string]models.EventStatus)}
}

var _ EventOperator = &MemEventOperator{}

// Save implements EventOperator interface and always returns New status
func (o *MemEventOperator) Save(ctx context.Context, e lookout.Event) (models.EventStatus, error) {
	id := e.ID().String()
	s, ok := o.events[id]
	if !ok {
		s = models.EventStatusNew
		o.events[id] = s
	}

	return s, nil
}

// UpdateStatus implements EventOperator interface and does nothing
func (o *MemEventOperator) UpdateStatus(ctx context.Context, e lookout.Event, s models.EventStatus) error {
	id := e.ID().String()
	if _, ok := o.events[id]; !ok {
		return errors.New("event not found")
	}

	o.events[id] = s
	return nil
}
