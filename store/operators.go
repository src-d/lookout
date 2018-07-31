package store

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/store/models"
)

type EventOperator interface {
	Save(context.Context, lookout.Event) (models.EventStatus, error)
	UpdateStatus(context.Context, lookout.Event, models.EventStatus) error
}

// NoopEventOperator satisfies EventOperator interface but does nothing
type NoopEventOperator struct{}

var _ EventOperator = &NoopEventOperator{}

func (o *NoopEventOperator) Save(context.Context, lookout.Event) (models.EventStatus, error) {
	return models.EventStatusNew, nil
}

func (o *NoopEventOperator) UpdateStatus(context.Context, lookout.Event, models.EventStatus) error {
	return nil
}
