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

// Save implements EventOperator interface
func (o *MemEventOperator) Save(ctx context.Context, e lookout.Event) (models.EventStatus, error) {
	id := e.ID().String()
	s, ok := o.events[id]
	if !ok {
		s = models.EventStatusNew
		o.events[id] = s
	}

	return s, nil
}

// UpdateStatus implements EventOperator interface
func (o *MemEventOperator) UpdateStatus(ctx context.Context, e lookout.Event, s models.EventStatus) error {
	id := e.ID().String()
	if _, ok := o.events[id]; !ok {
		return errors.New("event not found")
	}

	o.events[id] = s
	return nil
}

// MemCommentOperator satisfies CommentOperator interface but does nothing
type MemCommentOperator struct {
	comments map[uint32][]*lookout.Comment
}

// NewMemCommentOperator creates new MemCommentOperator
func NewMemCommentOperator() *MemCommentOperator {
	return &MemCommentOperator{comments: make(map[uint32][]*lookout.Comment)}
}

var _ CommentOperator = &MemCommentOperator{}

// Save implements EventOperator interface
func (o *MemCommentOperator) Save(ctx context.Context, e lookout.Event, c *lookout.Comment) error {
	re := e.(*lookout.ReviewEvent)
	o.comments[re.Number] = append(o.comments[re.Number], c)

	return nil
}

// Posted implements EventOperator interface
func (o *MemCommentOperator) Posted(ctx context.Context, e lookout.Event, c *lookout.Comment) (bool, error) {
	re := e.(*lookout.ReviewEvent)

	comments, ok := o.comments[re.Number]
	if !ok {
		return false, nil
	}

	for _, sc := range comments {
		if sc.File == c.File && sc.Line == c.Line && sc.Text == c.Text {
			return true, nil
		}
	}

	return false, nil
}
