package main

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/provider/github"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"

	"google.golang.org/grpc"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-log.v1"
)

func init() {
	if _, err := parser.AddCommand("serve", "run server", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}
}

type ServeCommand struct {
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL of the analyzer to use"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd    string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	Library    string `long:"library" default:"/tmp/lookout" env:"LOOKOUT_LIBRARY" description:"path to the lookout library"`
	Positional struct {
		Repository string `positional-arg-name:"repository"`
	} `positional-args:"yes" required:"yes"`

	analyzer lookout.AnalyzerClient
}

func (c *ServeCommand) Execute(args []string) error {
	if err := c.startServer(); err != nil {
		return err
	}

	if err := c.initAnalyzer(); err != nil {
		return err
	}

	watcher, err := github.NewWatcher(&lookout.WatchOptions{
		URL: c.Positional.Repository,
	})
	if err != nil {
		return err
	}

	return watcher.Watch(context.Background(), c.handleEvent)
}

func (c *ServeCommand) initAnalyzer() error {
	var err error
	c.Analyzer, err = lookout.ToGoGrpcAddress(c.Analyzer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.Analyzer, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	c.analyzer = lookout.NewAnalyzerClient(conn)
	return nil

}

func (c *ServeCommand) startServer() error {
	var err error
	c.Bblfshd, err = lookout.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return err
	}

	bblfshConn, err := grpc.Dial(c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return err
	}

	lib := git.NewLibrary(osfs.New(c.Library))
	sync := git.NewSyncer(lib)
	loader := git.NewLibraryCommitLoader(lib, sync)

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

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Errorf(err, "data server failed")
		}
	}()
	return nil
}

func (c *ServeCommand) handleEvent(e lookout.Event) error {
	switch ev := e.(type) {
	case *lookout.PullRequestEvent:
		return c.handlePR(ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

func (c *ServeCommand) handlePR(e *lookout.PullRequestEvent) error {
	log := log.New(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	log.Infof("processing pull request")
	resp, err := c.analyzer.NotifyPullRequestEvent(context.TODO(), e)
	if err != nil {
		log.Errorf(err, "analysis failed")
		return nil
	}

	poster := &LogPoster{log}
	return poster.Post(context.TODO(), e, resp.Comments)
}

type LogPoster struct {
	Log log.Logger
}

func (p *LogPoster) Post(ctx context.Context, e lookout.Event,
	comments []*lookout.Comment) error {
	for _, c := range comments {
		logger := p.Log.With(log.Fields{
			"text": c.Text,
		})
		if c.File == "" {
			logger.Infof("global comment")
			continue
		}

		logger = logger.With(log.Fields{"file": c.File})
		if c.Line == 0 {
			logger.Infof("file comment")
			continue
		}

		logger.With(log.Fields{"line": c.Line}).Infof("line comment")
	}

	return nil
}
