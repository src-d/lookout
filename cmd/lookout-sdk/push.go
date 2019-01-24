package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"

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

	err = srv.HandlePush(context.TODO(), &lookout.PushEvent{
		InternalID: id.String(),
		CreatedAt:  time.Now(),
		Commits:    commits,
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
