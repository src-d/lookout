package store

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/store/models"
	kallax "gopkg.in/src-d/go-kallax.v1"
	log "gopkg.in/src-d/go-log.v1"
)

// DBEventOperator operates on event database store
type DBEventOperator struct {
	reviewsStore *models.ReviewEventStore
}

func NewDBEventOperator(s *models.ReviewEventStore) *DBEventOperator {
	return &DBEventOperator{s}
}

var _ EventOperator = &DBEventOperator{}

func (o *DBEventOperator) Save(ctx context.Context, e lookout.Event) (models.EventStatus, error) {
	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		return o.saveReview(ctx, ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
	}

	return models.EventStatusNew, nil
}

func (o *DBEventOperator) UpdateStatus(ctx context.Context, e lookout.Event, status models.EventStatus) error {
	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		return o.updateReviewStatus(ctx, ev, status)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

func (o *DBEventOperator) saveReview(ctx context.Context, e *lookout.ReviewEvent) (models.EventStatus, error) {
	m, err := o.getReview(ctx, e)
	if err == kallax.ErrNotFound {
		return models.EventStatusNew, o.reviewsStore.Insert(models.NewReviewEvent(e))
	}
	if err != nil {
		return models.EventStatusNew, err
	}

	status := models.EventStatusNew
	if m.Status != "" {
		status = m.Status
	}
	return status, nil
}

func (o *DBEventOperator) updateReviewStatus(ctx context.Context, e *lookout.ReviewEvent, s models.EventStatus) error {
	m, err := o.getReview(ctx, e)
	if err != nil {
		return err
	}

	m.Status = s

	_, err = o.reviewsStore.Update(m, models.Schema.ReviewEvent.Status)

	return err
}

func (o *DBEventOperator) getReview(ctx context.Context, e *lookout.ReviewEvent) (*models.ReviewEvent, error) {
	q := models.NewReviewEventQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID)

	return o.reviewsStore.FindOne(q)
}
