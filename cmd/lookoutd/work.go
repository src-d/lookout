package main

import (
	"context"
	"fmt"

	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/util/cli"
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

	err = c.InitQueue()
	if err != nil {
		return err
	}

	ctx := context.Background()
	server := server.NewServer(poster, dataHandler.FileGetter, analyzers, eventOp, commentsOp)

	c.probeReadiness = true

	return c.runEventDequeuer(ctx, c.QueueOptions, server)
}
