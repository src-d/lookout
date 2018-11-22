package main

import (
	"context"
	"fmt"

	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/ctxlog"
)

func init() {
	if _, err := app.AddCommand("work", "run a worker for a distributed environment", "",
		&WorkCommand{}); err != nil {
		panic(err)
	}
}

type WorkCommand struct {
	queueConsumerCommand
	cli.QueueOptions
}

func (c *WorkCommand) Execute(args []string) error {
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

	err = c.InitQueue()
	if err != nil {
		return err
	}

	server := server.NewServer(
		poster, dataHandler.FileGetter,
		analyzers,
		eventOp, commentsOp,
		conf.Timeout.AnalyzerReview, conf.Timeout.AnalyzerPush,
	)

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		err := startDataServer()
		ctxlog.Get(ctx).Errorf(err, "data server stopped")
		stopCh <- err
	}()

	go func() {
		err := c.runEventDequeuer(ctx, c.QueueOptions, server)
		ctxlog.Get(ctx).Errorf(err, "event dequeuer stopped")
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
