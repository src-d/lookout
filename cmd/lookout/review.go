package main

import (
	"context"

	"github.com/src-d/lookout"
)

func init() {
	if _, err := parser.AddCommand("review", "provides simple data server and triggers analyzer", "",
		&ReviewCommand{}); err != nil {
		panic(err)
	}
}

type ReviewCommand struct {
	EventCommand
}

func (c *ReviewCommand) Execute(args []string) error {
	if err := c.openRepository(); err != nil {
		return err
	}

	fromRef, toRef, err := c.resolveRefs()
	if err != nil {
		return err
	}

	grpcSrv, err := c.makeDataServer()
	if err != nil {
		return err
	}

	lis, err := lookout.Listen(c.DataServer)
	if err != nil {
		return err
	}

	serveResult := make(chan error)
	go func() { serveResult <- grpcSrv.Serve(lis) }()

	client, err := c.analyzerClient()
	if err != nil {
		return err
	}

	resp, err := client.NotifyReviewEvent(context.TODO(),
		&lookout.ReviewEvent{
			IsMergeable: true,
			Source:      *toRef,
			Merge:       *toRef,
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
