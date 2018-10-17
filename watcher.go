package lookout

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
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

// EventHandler is the function to be called when a new event happens.
type EventHandler func(context.Context, Event) error

const cacheSize = 100000

// CachedHandler wraps an EventHandler, keeping a cache to skip successfully
// processed Events
func CachedHandler(fn EventHandler) EventHandler {
	cache, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, e Event) error {
		if _, ok := cache.Get(e.ID()); ok {
			return nil
		}

		err := fn(ctx, e)

		if err == nil {
			cache.Add(e.ID(), nil)
		}

		return err
	}
}
