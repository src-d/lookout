package lookout

import (
	"context"

	"gopkg.in/src-d/go-errors.v1"
)

var (
	// NoErrStopWatcher if a new error of this kind is returned by EventHandler
	// the Watcher.Watch function exits without error.
	NoErrStopWatcher = errors.NewKind("Stop watcher")
)

// Watcher watch for new events in given provider.
type Watcher interface {
	// Watch for new events triggering the EventHandler for each new issue,
	// it stops until an error is returned by the EventHandler. Network errors
	// or other temporal errors are handled as non-fatal errors, just logging it.
	Watch(context.Context, EventHandler) error
}

// EventHandler funciton to be called when a new event happends.
type EventHandler func(Event) error

// WatchOptions options to use in the Watcher constructors.
type WatchOptions struct {
	URLs []string
}
