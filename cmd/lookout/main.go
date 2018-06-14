package main

import (
	"context"
	"fmt"
	"os"

	"github.com/src-d/lookout/api"
	"github.com/src-d/lookout/git"
	"github.com/src-d/lookout/server"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	gogit "gopkg.in/src-d/go-git.v4"
	gogitsrv "gopkg.in/src-d/go-git.v4/plumbing/transport/server"
)

var parser = flags.NewParser(nil, flags.Default)

type AnalyzeCommand struct {
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL of the analyzer to use"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
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

	l := gogitsrv.MapLoader{
		"repo:///repo": r.Storer,
	}
	srv := server.NewServer(git.NewService(l))
	grpcSrv := grpc.NewServer()
	api.RegisterDataServer(grpcSrv, srv)
	lis, err := server.Listen(c.DataServer)
	if err != nil {
		return err
	}

	serveResult := make(chan error)
	go func() { serveResult <- grpcSrv.Serve(lis) }()

	c.Analyzer, err = server.ToGoGrpcAddress(c.Analyzer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.Analyzer, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	client := api.NewAnalyzerClient(conn)
	resp, err := client.Analyze(context.TODO(), &api.AnalysisRequest{
		Repository: "repo:///repo",
		BaseHash:   parentHash,
		NewHash:    headHash,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Result: %#v\n", resp)

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
