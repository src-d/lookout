package main

import (
	"context"

	"github.com/src-d/lookout"
	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func init() {
	if _, err := parser.AddCommand("push", "provides simple data server and triggers analyzer", "",
		&PushCommand{}); err != nil {
		panic(err)
	}
}

type PushCommand struct {
	EventCommand
}

func (c *PushCommand) Execute(args []string) error {
	if err := c.openRepository(); err != nil {
		return err
	}

	fromRef, toRef, err := c.resolveRefs()
	if err != nil {
		return err
	}

	dataSrv, err := c.makeDataServerHandler()
	if err != nil {
		return err
	}

	serveResult := make(chan error)
	grpcSrv, err := c.bindDataServer(dataSrv, serveResult)
	if err != nil {
		return err
	}

	client, err := c.analyzerClient()
	if err != nil {
		return err
	}

	log, err := c.repo.Log(&gogit.LogOptions{From: plumbing.NewHash(toRef.Hash)})
	var commits uint32
	for {
		commit, err := log.Next()
		if err != nil {
			return err
		}
		if commit.Hash.String() == fromRef.Hash {
			break
		}
		commits++
	}

	resp, err := client.NotifyPushEvent(context.TODO(),
		&lookout.PushEvent{
			Commits: commits,
			CommitRevision: lookout.CommitRevision{
				Base: *fromRef,
				Head: *toRef,
			}})
	if err != nil {
		return err
	}

	c.printComments(resp.Comments)

	grpcSrv.GracefulStop()
	return <-serveResult
}
