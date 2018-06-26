package lookout

import "context"

// Poster can post comments about an event.
type Poster interface {
	// Post posts comments about an event.
	Post(context.Context, Event, []*Comment) error
}
