package models

//go:generate kallax gen

import (
	"github.com/src-d/lookout"
	kallax "gopkg.in/src-d/go-kallax.v1"
)

// ReviewEvent is a persisted model for review event
type ReviewEvent struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	Status       EventStatus

	// can't be pointer or kallax panics
	lookout.ReviewEvent `kallax:",inline"`

	ReviewTarget *ReviewTarget `fk:",inverse"`
}

func newReviewEvent(e *lookout.ReviewEvent) *ReviewEvent {
	return &ReviewEvent{ID: kallax.NewULID(), Status: EventStatusNew, ReviewEvent: *e}
}

// ReviewTarget is a persisted for a pull request
type ReviewTarget struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	Provider     string
	InternalID   string
	RepositoryID uint32
	Number       uint32
}

// PushEvent is a persisted model for review event
type PushEvent struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	Status       EventStatus

	// can't be pointer or kallax panics
	lookout.PushEvent `kallax:",inline"`
}

func newPushEvent(e *lookout.PushEvent) *PushEvent {
	return &PushEvent{ID: kallax.NewULID(), Status: EventStatusNew, PushEvent: *e}
}

// Comment is a persisted model for comment
type Comment struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	ReviewEvent  *ReviewEvent `fk:",inverse"`

	lookout.Comment `kallax:",inline"`
	Analyzer        string
}

func newComment(r *ReviewEvent, c *lookout.Comment) *Comment {
	return &Comment{ID: kallax.NewULID(), ReviewEvent: r, Comment: *c}
}
