package main

import (
	"context"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	if _, err := app.AddCommand("review", "provides simple data server and triggers analyzer", "",
		&ReviewCommand{}); err != nil {
		panic(err)
	}
}

type ReviewCommand struct {
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
		&store.NoopEventOperator{}, &store.NoopCommentOperator{},
		0, 0)

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

	if err != nil {
		return err
	}

	stopDataServer()

	return <-stopCh
}
