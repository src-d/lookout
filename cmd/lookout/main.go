package main

import (
	"context"
	"fmt"
	"os"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/grpclog/glogger"
	gogit "gopkg.in/src-d/go-git.v4"
	gogitsrv "gopkg.in/src-d/go-git.v4/plumbing/transport/server"
)

var parser = flags.NewParser(nil, flags.Default)

type AnalyzeCommand struct {
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL of the analyzer to use"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd    string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	GitDir     string `long:"git-dir" default:"." env:"GIT_DIR" description:"path to the .git directory to analyze"`
}

func (c *AnalyzeCommand) Execute(args []string) error {
	r, err := gogit.PlainOpenWithOptions(c.GitDir, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return err
	}

	headRef, err := r.Head()
	if err != nil {
		return err
	}

	headCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}

	headHash := headCommit.Hash.String()

	var parentHash string
	if len(headCommit.ParentHashes) > 0 {
		parentCommit, err := headCommit.Parent(0)
		if err != nil {
			return err
		}

		parentHash = parentCommit.Hash.String()
	}

	c.Bblfshd, err = lookout.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return err
	}

	bblfshConn, err := grpc.Dial(c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return err
	}

	l := gogitsrv.MapLoader{
		"repo:///repo": r.Storer,
	}
	srv := &lookout.DataServerHandler{
		ChangeGetter: bblfsh.NewService(
			git.NewService(l),
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

	c.Analyzer, err = lookout.ToGoGrpcAddress(c.Analyzer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.Analyzer, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	client := lookout.NewAnalyzerClient(conn)
	resp, err := client.NotifyPullRequestEvent(context.TODO(),
		&lookout.PullRequestEvent{
			CommitRevision: lookout.CommitRevision{
				Base: lookout.ReferencePointer{
					InternalRepositoryURL: "file:///repo",
					Hash: parentHash,
				},
				Head: lookout.ReferencePointer{
					InternalRepositoryURL: "file:///repo",
					Hash: headHash,
				},
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

func main() {
	if _, err := parser.AddCommand("analyze", "analyzes HEAD", "",
		&AnalyzeCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.Parse(); err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				os.Exit(0)
			}

			parser.WriteHelp(os.Stdout)
		}

		os.Exit(1)
	}
}
