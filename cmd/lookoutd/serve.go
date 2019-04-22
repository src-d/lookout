package main

import (
	"context"
	"fmt"

	lookoutQueue "github.com/src-d/lookout/queue"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/util/ctxlog"

	gocli "gopkg.in/src-d/go-cli.v0"
	queue "gopkg.in/src-d/go-queue.v1"
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

	posterQ, err := newMemQueue("poster-queue")
	if err != nil {
		return err
	}

	poster, err := c.initPoster(c.conf)
	if err != nil {
		return err
	}

	posterInQueue := lookoutQueue.NewPoster(poster, posterQ)

	watcher, err := c.initWatcher(c.conf)
	if err != nil {
		return err
	}

	eventsQ, err := newMemQueue("events-queue")
	if err != nil {
		return err
	}

	server := server.NewServer(server.Options{
		Poster:         posterInQueue,
		FileGetter:     dataHandler.FileGetter,
		Analyzers:      analyzers,
		EventOp:        eventOp,
		CommentOp:      commentsOp,
		OrganizationOp: organizationsOp,
		ReviewTimeout:  c.conf.Timeout.AnalyzerReview,
		PushTimeout:    c.conf.Timeout.AnalyzerPush,
	})

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		err := startDataServer()
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "data server stopped")
		}
		stopCh <- err
	}()

	go func() {
		err := c.runEventDequeuer(ctx, eventsQ, server)
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "event dequeuer stopped")
		}
		stopCh <- err
	}()

	go func() {
		err := c.runEventEnqueuer(ctx, eventsQ, watcher)
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "event enqueuer stopped")
		}
		stopCh <- err
	}()

	go func() {
		// TODO number of workers should be configurable
		err := posterInQueue.Consume(ctx, 1)
		if err != context.Canceled {
			ctxlog.Get(ctx).Errorf(err, "poster consumer stopped")
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

func newMemQueue(name string) (queue.Queue, error) {
	b, err := queue.NewBroker("memory://")
	if err != nil {
		return nil, err
	}

	return b.Queue(name)
}
