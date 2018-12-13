package main

import (
	"context"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/grpchelper"

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

	dataHandler, err := c.makeDataServerHandler()
	if err != nil {
		return err
	}

	startDataServer, stopDataServer := c.initDataServer(dataHandler)
	go func() {
		stopCh <- startDataServer()
	}()

	analyzer, err := c.analyzer()
	if err != nil {
		return err
	}

	srv := server.NewServer(
		&server.LogPoster{log.DefaultLogger}, dataHandler.FileGetter,
		map[string]lookout.Analyzer{
			analyzer.Config.Name: analyzer,
		},
		&store.NoopEventOperator{}, &store.NoopCommentOperator{},
		0, 0)

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	ev := lookout.ReviewEvent{
		InternalID:  id.String(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsMergeable: true,
		Source:      *toRef,
		Merge:       *toRef,
		CommitRevision: lookout.CommitRevision{
			Base: *fromRef,
			Head: *toRef,
		}}

	st := grpchelper.ToPBStruct(analyzer.Config.Settings)
	if st != nil {
		ev.Configuration = *st
	}

	err = srv.HandleReview(context.TODO(), &ev, false)

	if err != nil {
		return err
	}

	stopDataServer()

	return <-stopCh
}
