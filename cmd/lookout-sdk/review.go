package main

import (
	"context"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"

	uuid "github.com/satori/go.uuid"
	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	app.AddCommand(&ReviewCommand{})
}

type ReviewCommand struct {
	gocli.PlainCommand `name:"review" short-description:"trigger a review event" long-description:"Provides a simple data server and triggers an analyzer review event"`
	EventCommand
}

func (c *ReviewCommand) Execute(args []string) error {
	stopCh := make(chan error, 1)

	if err := c.openRepository(); err != nil {
		return err
	}

	fromRef, toRef, err := c.resolveRefs()
	if err != nil {
		return err
	}

	conf, err := c.parseConfig()
	if err != nil {
		return err
	}

	dataHandler, err := c.makeDataServerHandler()
	if err != nil {
		return err
	}

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		stopCh <- startDataServer()
	}()

	client, err := c.analyzerClient()
	if err != nil {
		return err
	}

	srv := server.NewServer(
		&server.LogPoster{log.DefaultLogger}, dataHandler.FileGetter,
		map[string]lookout.Analyzer{
			"test-analyzes": lookout.Analyzer{
				Client: client,
			},
		},
		&store.NoopEventOperator{},
		&store.NoopCommentOperator{},
		&store.NoopOrganizationOperator{},
		0, 0)
	srv.ExitOnError = true

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	err = srv.HandleReview(context.TODO(), &lookout.ReviewEvent{
		InternalID:  id.String(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsMergeable: true,
		Source:      *toRef,
		Merge:       *toRef,
		CommitRevision: lookout.CommitRevision{
			Base: *fromRef,
			Head: *toRef,
		},
		Configuration: conf}, false)

	stopDataServer()

	if err != nil {
		return err
	}

	return <-stopCh
}
