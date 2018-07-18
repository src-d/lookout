package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"

	"google.golang.org/grpc"
	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func init() {
	if _, err := parser.AddCommand("review", "provides simple data server and triggers analyzer", "",
		&ReviewCommand{}); err != nil {
		panic(err)
	}
}

type ReviewCommand struct {
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd    string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	GitDir     string `long:"git-dir" default:"." env:"GIT_DIR" description:"path to the .git directory to analyze"`
	RevFrom    string `long:"from" default:"HEAD^" description:"name of the base revision for review event"`
	RevTo      string `long:"to" default:"HEAD" description:"name of the head revision for review event"`
	Args       struct {
		Analyzer string `positional-arg-name:"analyzer" description:"gRPC URL of the analyzer to use"`
	} `positional-args:"yes" required:"yes"`
}

func (c *ReviewCommand) Execute(args []string) error {
	r, err := gogit.PlainOpenWithOptions(c.GitDir, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return err
	}

	baseHash, err := getCommitHashByRev(r, c.RevFrom)
	if err != nil {
		return fmt.Errorf("base revision error: %s", err)
	}

	headHash, err := getCommitHashByRev(r, c.RevTo)
	if err != nil {
		return fmt.Errorf("head revision error: %s", err)
	}

	c.Bblfshd, err = lookout.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return err
	}

	bblfshConn, err := grpc.Dial(c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return err
	}

	loader := git.NewStorerCommitLoader(r.Storer)
	srv := &lookout.DataServerHandler{
		ChangeGetter: bblfsh.NewService(
			git.NewService(loader),
			bblfshConn,
		),
	}
	grpcSrv := grpc.NewServer()
	lookout.RegisterDataServer(grpcSrv, srv)
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

	fullGitPath, err := filepath.Abs(c.GitDir)
	if err != nil {
		return fmt.Errorf("can't resolve full path: %s", err)
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

	client := lookout.NewAnalyzerClient(conn)
	resp, err := client.NotifyReviewEvent(context.TODO(),
		&lookout.ReviewEvent{
			IsMergeable: true,
			Source:      toRef,
			Merge:       toRef,
			CommitRevision: lookout.CommitRevision{
				Base: fromRef,
				Head: toRef,
			}})
	if err != nil {
		return err
	}

	fmt.Println("BEGIN RESULT")
	for _, comment := range resp.Comments {
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

	grpcSrv.GracefulStop()
	return <-serveResult
}

func getCommitHashByRev(r *gogit.Repository, revName string) (string, error) {
	h, err := r.ResolveRevision(plumbing.Revision(revName))
	if err != nil {
		return "", err
	}

	return h.String(), nil
}
