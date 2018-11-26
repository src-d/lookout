package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	uuid "github.com/satori/go.uuid"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/store"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-log.v1"
)

func init() {
	if _, err := app.AddCommand("review-all", "provides simple data server and triggers analyzer", "",
		&ReviewAllCommand{}); err != nil {
		panic(err)
	}
}

type ReviewAllCommand struct {
	EventCommand
}

func (c *ReviewAllCommand) Execute(args []string) error {
	fullGitPath, err := filepath.Abs(c.GitDir)
	if err != nil {
		return fmt.Errorf("can't resolve '%s' full path: %s", c.GitDir, err)
	}

	if err := c.openRepository(); err != nil {
		return err
	}

	conf, err := c.parseConfig()
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

	commitIter, err := c.repo.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	srv := server.NewServer(
		&server.LogPoster{log.DefaultLogger}, dataSrv.FileGetter,
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

	err = commitIter.ForEach(func(c *object.Commit) error {
		//TODO(smola): we may want to serve initial commits
		if c.NumParents() != 1 {
			return nil
		}

		parent, err := c.Parent(0)
		if err != nil {
			return err
		}

		fromRef := &lookout.ReferencePointer{
			InternalRepositoryURL: "file://" + fullGitPath,
			ReferenceName:         plumbing.HEAD,
			Hash:                  parent.Hash.String(),
		}

		toRef := &lookout.ReferencePointer{
			InternalRepositoryURL: "file://" + fullGitPath,
			ReferenceName:         plumbing.HEAD,
			Hash:                  c.Hash.String(),
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
			Configuration: conf})

		return err
	})
	if err != nil {
		return err
	}

	commitIter.Close()
	grpcSrv.GracefulStop()
	return <-serveResult
}
