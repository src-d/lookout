package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/grpchelper"

	uuid "github.com/satori/go.uuid"
	gocli "gopkg.in/src-d/go-cli.v0"
	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	app.AddCommand(&PushCommand{})
}

type PushCommand struct {
	gocli.PlainCommand `name:"push" short-description:"trigger a push event" long-description:"Provides a simple data server and triggers an analyzer push event"`
	EventCommand
}

func (c *PushCommand) Execute(args []string) error {
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

	log, err := c.repo.Log(&gogit.LogOptions{From: plumbing.NewHash(toRef.Hash)})
	var commits uint32
	for {
		commit, err := log.Next()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("revision %s is not a parent of %s",
					fromRef.Hash, toRef.Hash)
			}

			return err
		}
		if commit.Hash.String() == fromRef.Hash {
			break
		}
		commits++
	}

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	ev := lookout.PushEvent{
		InternalID: id.String(),
		CreatedAt:  time.Now(),
		Commits:    commits,
		CommitRevision: lookout.CommitRevision{
			Base: *fromRef,
			Head: *toRef,
		}}

	st := grpchelper.ToPBStruct(analyzer.Config.Settings)
	if st != nil {
		ev.Configuration = *st
	}

	err = srv.HandlePush(context.TODO(), &ev, false)

	if err != nil {
		return err
	}

	stopDataServer()

	return <-stopCh
}
