package main

import (
	"context"

	"github.com/src-d/lookout"
	"google.golang.org/grpc"
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

	c.Args.Analyzer, err = lookout.ToGoGrpcAddress(c.Args.Analyzer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.Args.Analyzer, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	client := lookout.NewAnalyzerClient(conn)

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
