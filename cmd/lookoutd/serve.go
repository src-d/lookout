package main

import (
	"context"
	"fmt"

	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/ctxlog"

	gocli "gopkg.in/src-d/go-cli.v0"
)

func init() {
	app.AddCommand(&ServeCommand{})
}

type ServeCommand struct {
	gocli.PlainCommand `name:"serve" short-description:"run a standalone server" long-description:"Run a standalone server"`
	queueConsumerCommand
}

func (c *ServeCommand) ExecuteContext(ctx context.Context, args []string) error {
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

	dataHandler, err := c.initDataHandler(c.conf)
	if err != nil {
		return err
	}

	db, err := c.InitDB()
	if err != nil {
		return fmt.Errorf("Can't connect to the DB: %s", err)
	}

	eventOp, commentsOp, organizationsOp := c.initDBOperators(db)

	analyzers, err := c.initAnalyzers(c.conf)
	if err != nil {
		return err
	}

	poster, err := c.initPoster(c.conf)
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher(c.conf)
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
		eventOp, commentsOp, organizationsOp,
		c.conf.Timeout.AnalyzerReview, c.conf.Timeout.AnalyzerPush)

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		err := startDataServer()
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "data server stopped")
		}
		stopCh <- err
	}()

	go func() {
		err := c.runEventDequeuer(ctx, qOpt, server)
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "event dequeuer stopped")
		}
		stopCh <- err
	}()

	go func() {
		err := c.runEventEnqueuer(ctx, qOpt, watcher)
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

	// stop data server, it does not stop with context
	stopDataServer()

	if err != context.Canceled {
		return err
	}

	return nil
}
