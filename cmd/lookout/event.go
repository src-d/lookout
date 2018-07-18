package main

import (
	"fmt"
	"path/filepath"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"
	"google.golang.org/grpc"

	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type EventCommand struct {
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd    string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	GitDir     string `long:"git-dir" default:"." env:"GIT_DIR" description:"path to the .git directory to analyze"`
	RevFrom    string `long:"from" default:"HEAD^" description:"name of the base revision for event"`
	RevTo      string `long:"to" default:"HEAD" description:"name of the head revision for event"`
	Args       struct {
		Analyzer string `positional-arg-name:"analyzer" description:"gRPC URL of the analyzer to use"`
	} `positional-args:"yes" required:"yes"`

	repo *gogit.Repository
}

func (c *EventCommand) openRepository() error {
	var err error

	c.repo, err = gogit.PlainOpenWithOptions(c.GitDir, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})

	return err
}

func (c *EventCommand) resolveRefs() (*lookout.ReferencePointer, *lookout.ReferencePointer, error) {
	baseHash, err := getCommitHashByRev(c.repo, c.RevFrom)
	if err != nil {
		return nil, nil, fmt.Errorf("base revision error: %s", err)
	}

	headHash, err := getCommitHashByRev(c.repo, c.RevTo)
	if err != nil {
		return nil, nil, fmt.Errorf("head revision error: %s", err)
	}

	fullGitPath, err := filepath.Abs(c.GitDir)
	if err != nil {
		return nil, nil, fmt.Errorf("can't resolve full path: %s", err)
	}

	fromRef := lookout.ReferencePointer{
		InternalRepositoryURL: "file://" + fullGitPath,
		ReferenceName:         plumbing.ReferenceName(c.RevFrom),
		Hash:                  baseHash,
	}

	toRef := lookout.ReferencePointer{
		InternalRepositoryURL: "file://" + fullGitPath,
		ReferenceName:         plumbing.ReferenceName(c.RevTo),
		Hash:                  headHash,
	}

	return &fromRef, &toRef, nil
}

func (c *EventCommand) makeDataServer() (*grpc.Server, error) {
	var err error

	c.Bblfshd, err = lookout.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return nil, err
	}

	bblfshConn, err := grpc.Dial(c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	loader := git.NewStorerCommitLoader(c.repo.Storer)
	gitService := git.NewService(loader)
	bblfshService := bblfsh.NewService(gitService, gitService, bblfshConn)

	srv := &lookout.DataServerHandler{
		ChangeGetter: bblfshService,
		FileGetter:   bblfshService,
	}
	grpcSrv := grpc.NewServer()
	lookout.RegisterDataServer(grpcSrv, srv)

	return grpcSrv, nil
}

func (c *EventCommand) printComments(cs []*lookout.Comment) {
	fmt.Println("BEGIN RESULT")
	for _, comment := range cs {
		if comment.File == "" {
			fmt.Printf("GLOBAL: %s\n", comment.Text)
			continue
		}

		if comment.Line == 0 {
			fmt.Printf("%s: %s\n", comment.File, comment.Text)
			continue
		}

		fmt.Printf("%s:%d: %s\n", comment.File, comment.Line, comment.Text)
	}

	fmt.Println("END RESULT")
}

func getCommitHashByRev(r *gogit.Repository, revName string) (string, error) {
	h, err := r.ResolveRevision(plumbing.Revision(revName))
	if err != nil {
		return "", err
	}

	return h.String(), nil
}
