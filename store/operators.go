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

// CommentOperator manages persistence of Comments
type CommentOperator interface {
	// Save persists Comment in a store
	Save(context.Context, lookout.Event, *lookout.Comment, string) error
	// Posted checks if a comment was already posted for review
	Posted(context.Context, lookout.Event, *lookout.Comment) (bool, error)
}

// OrganizationOperator manages persistence of default config for organizations
type OrganizationOperator interface {
	// Save persists the given config, updating the current one if it exists
	// for the given (provider, orgID)
	Save(ctx context.Context, provider string, orgID string, config string) error
	// Config returns the stored config for the given (provider, orgID). If there
	// are no records in the DB, it returns "" without error.
	Config(ctx context.Context, provider string, orgID string) (string, error)
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

// NoopCommentOperator satisfies CommentOperator interface but does nothing
type NoopCommentOperator struct{}

var _ CommentOperator = &NoopCommentOperator{}

// Save implements EventOperator interface and does nothing
func (o *NoopCommentOperator) Save(context.Context, lookout.Event, *lookout.Comment, string) error {
	return nil
}

// Posted implements EventOperator interface and always returns false
func (o *NoopCommentOperator) Posted(context.Context, lookout.Event, *lookout.Comment) (bool, error) {
	return false, nil
}

// NoopOrganizationOperator satisfies OrganizationOperator interface but does nothing
type NoopOrganizationOperator struct{}

var _ OrganizationOperator = &NoopOrganizationOperator{}

// Save persists the given config, updating the current one if it exists
// for the given (provider, orgID)
func (o *NoopOrganizationOperator) Save(ctx context.Context, provider string, orgID string, config string) error {
	return nil
}

// Config returns the stored config for the given (provider, orgID). If there
// are no records in the DB, it returns "" without error.
func (o *NoopOrganizationOperator) Config(ctx context.Context, provider string, orgID string) (string, error) {
	return "", nil
}

