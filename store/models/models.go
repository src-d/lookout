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
}

func newReviewEvent(e *lookout.ReviewEvent) *ReviewEvent {
	return &ReviewEvent{ID: kallax.NewULID(), Status: EventStatusNew, ReviewEvent: *e}
}

// Comment is a persisted model for comment
type Comment struct {
	kallax.Model    `pk:"id"`
	ID              kallax.ULID
	lookout.Comment `kallax:",inline"`
}

func newComment(c lookout.Comment) *Comment {
	return &Comment{ID: kallax.NewULID(), Comment: c}
}
