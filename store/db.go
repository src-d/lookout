package store

import (
	"context"
	"fmt"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/store/models"
	"github.com/src-d/lookout/util/ctxlog"
	kallax "gopkg.in/src-d/go-kallax.v1"
)

// DBEventOperator operates on event database store
type DBEventOperator struct {
	reviewsStore      *models.ReviewEventStore
	reviewTargetStore *models.ReviewTargetStore
	pushStore         *models.PushEventStore
}

// NewDBEventOperator creates new DBEventOperator using kallax as storage
func NewDBEventOperator(
	r *models.ReviewEventStore,
	rt *models.ReviewTargetStore,
	p *models.PushEventStore,
) *DBEventOperator {
	return &DBEventOperator{r, rt, p}
}

var _ EventOperator = &DBEventOperator{}

// Save implements EventOperator interface
func (o *DBEventOperator) Save(ctx context.Context, e lookout.Event) (models.EventStatus, error) {
	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		return o.saveReview(ctx, ev)
	case *lookout.PushEvent:
		return o.savePush(ctx, ev)
	default:
		ctxlog.Get(ctx).Debugf("ignoring unsupported event: %s", ev)
	}

	return models.EventStatusNew, nil
}

// UpdateStatus implements EventOperator interface
func (o *DBEventOperator) UpdateStatus(ctx context.Context, e lookout.Event, status models.EventStatus) error {
	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		return o.updateReviewStatus(ctx, ev, status)
	case *lookout.PushEvent:
		return o.updatePushStatus(ctx, ev, status)
	default:
		ctxlog.Get(ctx).Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

func (o *DBEventOperator) saveReview(ctx context.Context, e *lookout.ReviewEvent) (models.EventStatus, error) {
	m, err := o.getReview(ctx, e)
	if err == kallax.ErrNotFound {
		m = models.NewReviewEvent(e)
		target, err := o.getOrCreateReviewTarget(ctx, e)
		if err != nil {
			return models.EventStatusNew, err
		}

		m.ReviewTarget = target
		// kallax will save both event and target models
		return models.EventStatusNew, o.reviewsStore.Insert(m)
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
	// select with joins don't work in kallax
	// https://github.com/src-d/go-kallax/issues/250
	//
	// q := models.NewReviewEventQuery().
	// 	WithReviewTarget().
	// 	Where(kallax.Eq(models.Schema.ReviewTarget.Provider, e.Provider)).
	// 	FindByOldInternalID(e.InternalID)
	//
	// use 2 queries instead

	qTarget := models.NewReviewTargetQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID)
	target, err := o.reviewTargetStore.FindOne(qTarget)
	if err != nil {
		return nil, err
	}

	q := models.NewReviewEventQuery().FindByReviewTarget(target.ID)

	return o.reviewsStore.FindOne(q)
}

func (o *DBEventOperator) getReviewTarget(ctx context.Context, e *lookout.ReviewEvent) (*models.ReviewTarget, error) {
	q := models.NewReviewTargetQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID)

	return o.reviewTargetStore.FindOne(q)
}

func (o *DBEventOperator) getOrCreateReviewTarget(ctx context.Context, e *lookout.ReviewEvent) (*models.ReviewTarget, error) {
	m, err := o.getReviewTarget(ctx, e)
	if err == kallax.ErrNotFound {
		return models.NewReviewTarget(e), nil
	}

	return m, err
}

func (o *DBEventOperator) savePush(ctx context.Context, e *lookout.PushEvent) (models.EventStatus, error) {
	m, err := o.getPush(ctx, e)
	if err == kallax.ErrNotFound {
		return models.EventStatusNew, o.pushStore.Insert(models.NewPushEvent(e))
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

func (o *DBEventOperator) updatePushStatus(ctx context.Context, e *lookout.PushEvent, s models.EventStatus) error {
	m, err := o.getPush(ctx, e)
	if err != nil {
		return err
	}

	m.Status = s

	_, err = o.pushStore.Update(m, models.Schema.PushEvent.Status)

	return err
}

func (o *DBEventOperator) getPush(ctx context.Context, e *lookout.PushEvent) (*models.PushEvent, error) {
	q := models.NewPushEventQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID)

	return o.pushStore.FindOne(q)
}

// DBCommentOperator operates on comments database store
type DBCommentOperator struct {
	store             *models.CommentStore
	reviewsStore      *models.ReviewEventStore
	reviewTargetStore *models.ReviewTargetStore
}

// NewDBCommentOperator creates new DBCommentOperator using kallax as storage
func NewDBCommentOperator(
	c *models.CommentStore,
	r *models.ReviewEventStore,
	rt *models.ReviewTargetStore,
) *DBCommentOperator {
	return &DBCommentOperator{c, r, rt}
}

var _ CommentOperator = &DBCommentOperator{}

// Save implements EventOperator interface
func (o *DBCommentOperator) Save(ctx context.Context, e lookout.Event, c *lookout.Comment, analyzerName string) error {
	ev, ok := e.(*lookout.ReviewEvent)
	if !ok {
		return fmt.Errorf("comments can belong only to review event but %v is given", e.Type())
	}

	return o.save(ctx, ev, c, analyzerName)
}

// Posted implements EventOperator interface
func (o *DBCommentOperator) Posted(ctx context.Context, e lookout.Event, c *lookout.Comment) (bool, error) {
	ev, ok := e.(*lookout.ReviewEvent)
	if !ok {
		return false, fmt.Errorf("comments can belong only to review event but %v is given", e.Type())
	}

	return o.posted(ctx, ev, c)
}

func (o *DBCommentOperator) save(ctx context.Context, e *lookout.ReviewEvent, c *lookout.Comment, analyzerName string) error {
	// select with joins don't work in kallax
	// https://github.com/src-d/go-kallax/issues/250
	//
	// q := models.NewReviewEventQuery().
	// 	WithReviewTarget().
	// 	Where(kallax.Eq(models.Schema.ReviewTarget.Provider, e.Provider)).
	// 	FindByOldInternalID(e.InternalID)
	//
	// use 2 queries instead

	qTarget := models.NewReviewTargetQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID)
	target, err := o.reviewTargetStore.FindOne(qTarget)
	if err != nil {
		return err
	}

	q := models.NewReviewEventQuery().FindByReviewTarget(target.ID)

	r, err := o.reviewsStore.FindOne(q)
	if err != nil {
		return err
	}

	m := models.NewComment(r, c)
	m.Analyzer = analyzerName
	_, err = o.store.Save(m)
	return err
}

func (o *DBCommentOperator) posted(ctx context.Context, e *lookout.ReviewEvent, c *lookout.Comment) (bool, error) {
	// select with joins don't work in kallax
	// https://github.com/src-d/go-kallax/issues/250
	//
	// reviewIdsQ := models.NewReviewEventQuery().
	// 	WithReviewTarget().
	// 	Where(kallax.Eq(models.Schema.ReviewTarget.Provider, e.Provider)).
	// 	Where(kallax.Eq(models.Schema.ReviewTarget.RepositoryID, e.RepositoryID)).
	// 	Where(kallax.Eq(models.Schema.ReviewTarget.Number, e.Number)).
	// 	Select(models.Schema.ReviewEvent.ID)
	//
	// use 2 queries instead

	qTarget := models.NewReviewTargetQuery().
		FindByProvider(e.Provider).
		FindByRepositoryID(kallax.Eq, e.RepositoryID).
		FindByNumber(kallax.Eq, e.Number)
	target, err := o.reviewTargetStore.FindOne(qTarget)
	if err != nil {
		return false, err
	}

	reviewIdsQ := models.NewReviewEventQuery().FindByReviewTarget(target.ID)

	reviews, err := o.reviewsStore.FindAll(reviewIdsQ)
	if err != nil {
		return false, err
	}

	reviewIds := make([]interface{}, len(reviews))
	for i, r := range reviews {
		reviewIds[i] = r.ID
	}

	q := models.NewCommentQuery().
		Where(kallax.In(models.Schema.Comment.ReviewEventFK, reviewIds...)).
		FindByFile(c.File).
		FindByLine(kallax.Eq, c.Line).
		FindByText(c.Text)

	count, err := o.store.Count(q)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
