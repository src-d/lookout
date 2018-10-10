package cli

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
)

// RunWatcher starts the watcher with the given handler callback. On failure
// the watcher is restarted
func RunWatcher(ctx context.Context, watcher lookout.Watcher, eventHandler lookout.EventHandler) error {
	errCh := make(chan error, 1)
	for {
		go func() {
			err := watcher.Watch(ctx, eventHandler)
			errCh <- err
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "error during watcher.Watch, restarting watcher")
			} else {
				return nil
			}
		}
	}
}
