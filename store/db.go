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
	q := models.NewReviewEventQuery().FindByInternalID(e.ID().String())

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
	q := models.NewReviewEventQuery().FindByInternalID(e.ID().String())

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

	// get review target (pull request)
	qTarget := models.NewReviewTargetQuery().
		FindByProvider(e.Provider).
		FindByInternalID(e.InternalID).
		Select(models.Schema.ReviewTarget.ID)
	target, err := o.reviewTargetStore.FindOne(qTarget)
	if err != nil {
		return false, err
	}

	// get all review events for this target (pull request)
	reviewIdsQ := models.NewReviewEventQuery().
		FindByReviewTarget(target.ID).
		Select(models.Schema.ReviewEvent.ID)
	reviews, err := o.reviewsStore.FindAll(reviewIdsQ)
	if err != nil {
		return false, err
	}

	reviewIds := make([]interface{}, len(reviews))
	for i, r := range reviews {
		reviewIds[i] = r.ID
	}

	// make sure we didn't post such comment in any of previous events
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

// DBOrganizationOperator operates on an organization database store
type DBOrganizationOperator struct {
	organizationStore *models.OrganizationStore
}

// NewDBOrganizationOperator creates new DBOrganizationOperator using kallax as storage
func NewDBOrganizationOperator(store *models.OrganizationStore) *DBOrganizationOperator {
	return &DBOrganizationOperator{store}
}

var _ OrganizationOperator = &DBOrganizationOperator{}

func (o *DBOrganizationOperator) getOrganization(ctx context.Context, provider string, orgID string) (*models.Organization, error) {
	q := models.NewOrganizationQuery().FindByProvider(provider).FindByInternalID(orgID)
	return o.organizationStore.FindOne(q)
}

// Save persists the given config, updating the current one if it exists
// for the given (provider, orgID)
func (o *DBOrganizationOperator) Save(ctx context.Context, provider string, orgID string, config string) error {
	m, err := o.getOrganization(ctx, provider, orgID)
	if err != nil && err != kallax.ErrNotFound {
		return err
	}

	if err == kallax.ErrNotFound {
		m = models.NewOrganization(provider, orgID, config)
	} else {
		m.Config = config
	}

	_, err = o.organizationStore.Save(m)
	return err
}

// Config returns the stored config for the given (provider, orgID). If there
// are no records in the DB, it returns "" without error.
func (o *DBOrganizationOperator) Config(ctx context.Context, provider string, orgID string) (string, error) {
	m, err := o.getOrganization(ctx, provider, orgID)
	if err == kallax.ErrNotFound {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return m.Config, nil
}
