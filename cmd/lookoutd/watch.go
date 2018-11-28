package main

import (
	"context"

	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/ctxlog"
)

func init() {
	if _, err := app.AddCommand("watch", "run a watcher for a distributed environment", "",
		&WatchCommand{}); err != nil {
		panic(err)
	}
}

type WatchCommand struct {
	lookoutdCommand
	cli.QueueOptions
}

func (c *WatchCommand) Execute(args []string) error {
	ctx, stopCtx := context.WithCancel(context.Background())
	stopCh := make(chan error, 1)

	go func() {
		err := c.startHealthProbes()
		ctxlog.Get(ctx).Errorf(err, "health probes server stopped")

		stopCh <- err
	}()

	conf, err := c.initConfig()
	if err != nil {
		return err
	}

	err = c.initProvider(conf)
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher(conf)
	if err != nil {
		return err
	}

	err = c.InitQueue()
	if err != nil {
		return err
	}

	go func() {
		err := c.runEventEnqueuer(ctx, c.QueueOptions, watcher)
		ctxlog.Get(ctx).Errorf(err, "event enqueuer stopped")
		stopCh <- err
	}()

	go func() {
		stopCh <- stopOnSignal(ctx)
	}()

	c.probeReadiness = true

	err = <-stopCh

	// stop servers gracefully
	stopCtx()

	return err
}
