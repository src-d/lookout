package main

import (
	"context"

	"github.com/src-d/lookout/util/cli"
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
	c.initHealthProbes()

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

	c.probeReadiness = true

	ctx := context.Background()
	return c.runEventEnqueuer(ctx, c.QueueOptions, watcher)
}
