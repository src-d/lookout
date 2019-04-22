package main

import (
	"context"

	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/ctxlog"
	gocli "gopkg.in/src-d/go-cli.v0"
)

func init() {
	app.AddCommand(&WatchCommand{})
}

type WatchCommand struct {
	gocli.PlainCommand `name:"watch" short-description:"run a watcher for a distributed environment" long-description:"Run a watcher for a distributed environment"`
	lookoutdCommand
	cli.QueueOptions
}

func (c *WatchCommand) ExecuteContext(ctx context.Context, args []string) error {
	ctx, stopCtx := context.WithCancel(ctx)
	stopCh := make(chan error, 1)

	go func() {
		err := c.startHealthProbes()
		ctxlog.Get(ctx).Errorf(err, "health probes server stopped")

		stopCh <- err
	}()

	err := c.initProvider(c.conf)
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher(c.conf)
	if err != nil {
		return err
	}

	err = c.InitQueue()
	if err != nil {
		return err
	}

	go func() {
		err := c.runEventEnqueuer(ctx, c.QueueOptions.Q, watcher)
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "event enqueuer stopped")
		}
		stopCh <- err
	}()

	c.probeReadiness = true

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-stopCh:
		// stop the other servers that did not fail
		stopCtx()
	}

	if err != context.Canceled {
		return err
	}

	return nil
}
