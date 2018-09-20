package models

//go:generate kallax gen

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/src-d/lookout"
	kallax "gopkg.in/src-d/go-kallax.v1"
)

// ReviewEvent is a persisted model for review event
type ReviewEvent struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	Status       EventStatus
	// temporary column for migration
	// should be removed in #259
	OldInternalID string

	// those fields can change with each push
	IsMergeable   bool
	Source        lookout.ReferencePointer
	Merge         lookout.ReferencePointer
	Configuration types.Struct
	Base          lookout.ReferencePointer
	Head          lookout.ReferencePointer
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// static part of review
	ReviewTarget *ReviewTarget `fk:",inverse"`
}

func newReviewEvent(e *lookout.ReviewEvent) *ReviewEvent {
	return &ReviewEvent{
		ID:            kallax.NewULID(),
		Status:        EventStatusNew,
		OldInternalID: e.InternalID,
		IsMergeable:   e.IsMergeable,
		Source:        e.Source,
		Merge:         e.Merge,
		Configuration: e.Configuration,
		Base:          e.Base,
		Head:          e.Head,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

// ReviewTarget is a persisted model for a pull request
type ReviewTarget struct {
	kallax.Model `pk:"id"`
	ID           kallax.ULID
	kallax.Timestamps

	Provider     string
	InternalID   string
	RepositoryID uint32
	Number       uint32
}

func newReviewTarget(e *lookout.ReviewEvent) *ReviewTarget {
	return &ReviewTarget{
		ID:           kallax.NewULID(),
		Provider:     e.Provider,
		InternalID:   e.InternalID,
		RepositoryID: e.RepositoryID,
		Number:       e.Number,
	}
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
	kallax.Timestamps
	ID          kallax.ULID
	ReviewEvent *ReviewEvent `fk:",inverse"`

	lookout.Comment `kallax:",inline"`
	Analyzer        string
}

func newComment(r *ReviewEvent, c *lookout.Comment) *Comment {
	return &Comment{ID: kallax.NewULID(), ReviewEvent: r, Comment: *c}
}
