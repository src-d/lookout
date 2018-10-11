package main

import (
	"context"
	"fmt"

	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/ctxlog"
)

func init() {
	if _, err := app.AddCommand("serve", "run a standalone server", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}
}

type ServeCommand struct {
	queueConsumerCommand
}

func (c *ServeCommand) Execute(args []string) error {
	c.initHealthProbes()

	conf, err := c.initConfig()
	if err != nil {
		return err
	}

	dataHandler, err := c.initDataHandler()
	if err != nil {
		return err
	}

	if err := c.startServer(dataHandler); err != nil {
		return err
	}

	db, err := c.InitDB()
	if err != nil {
		return fmt.Errorf("Can't connect to the DB: %s", err)
	}

	eventOp, commentsOp := c.initDBOperators(db)

	analyzers, err := c.initAnalyzers(conf)
	if err != nil {
		return err
	}

	err = c.initProvider(conf)
	if err != nil {
		return err
	}

	poster, err := c.initPoster(conf)
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher(conf)
	if err != nil {
		return err
	}

	qOpt := cli.QueueOptions{
		Queue:  "mem-queue",
		Broker: "memory://",
	}

	err = qOpt.InitQueue()
	if err != nil {
		return err
	}

	ctx := context.Background()
	server := server.NewServer(poster, dataHandler.FileGetter, analyzers, eventOp, commentsOp)

	c.probeReadiness = true

	deqErrCh := make(chan error, 1)
	enqErrCh := make(chan error, 1)

	go func() {
		deqErrCh <- c.runEventDequeuer(ctx, qOpt, server)
	}()

	go func() {
		enqErrCh <- c.runEventEnqueuer(ctx, qOpt, watcher)
	}()

	select {
	case err := <-deqErrCh:
		ctxlog.Get(ctx).Errorf(err, "error from the event dequeuer")
		return err
	case err := <-enqErrCh:
		ctxlog.Get(ctx).Errorf(err, "error from the event enqueuer")
		return err
	}
}
