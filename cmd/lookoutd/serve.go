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

	dataHandler, err := c.initDataHandler(conf)
	if err != nil {
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

	server := server.NewServer(
		poster, dataHandler.FileGetter, analyzers,
		eventOp, commentsOp,
		conf.Timeout.AnalyzerReview, conf.Timeout.AnalyzerPush)

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		err := startDataServer()
		ctxlog.Get(ctx).Errorf(err, "data server stopped")
		stopCh <- err
	}()

	go func() {
		err := c.runEventDequeuer(ctx, qOpt, server)
		ctxlog.Get(ctx).Errorf(err, "event dequeuer stopped")
		stopCh <- err
	}()

	go func() {
		err := c.runEventEnqueuer(ctx, qOpt, watcher)
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
	stopDataServer()

	return err
}
