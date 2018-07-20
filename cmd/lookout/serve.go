package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/provider/github"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"

	"google.golang.org/grpc"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	if _, err := parser.AddCommand("serve", "run server", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}
}

// AnalyzerConfig is a configuration of analyzer
type AnalyzerConfig struct {
	Name string
	Addr string
}

// Config is a server configuration
type Config struct {
	Analyzers []AnalyzerConfig
}

type ServeCommand struct {
	ConfigFile  string `long:"config" short:"c" default:"config.yml" env:"CONFIG_FILE" description:"path to configuration file"`
	GithubUser  string `long:"github-user" env:"GITHUB_USER" description:"user for the GitHub API"`
	GithubToken string `long:"github-token" env:"GITHUB_TOKEN" description:"access token for the GitHub API"`
	DataServer  string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd     string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	DryRun      bool   `long:"dry-run" env:"LOOKOUT_DRY_RUN" description:"analyze repositories and log the result without posting code reviews to GitHub"`
	Library     string `long:"library" default:"/tmp/lookout" env:"LOOKOUT_LIBRARY" description:"path to the lookout library"`
	Positional  struct {
		Repository string `positional-arg-name:"repository"`
	} `positional-args:"yes" required:"yes"`

	poster    lookout.Poster
	analyzers map[string]lookout.AnalyzerClient
}

func (c *ServeCommand) Execute(args []string) error {
	var conf Config
	data, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("Can't open configuration file: %s", err)
	}
	if err := yaml.Unmarshal([]byte(data), &conf); err != nil {
		return fmt.Errorf("Can't parse configuration file: %s", err)
	}

	if err := c.startServer(); err != nil {
		return err
	}

	c.analyzers = make(map[string]lookout.AnalyzerClient, len(conf.Analyzers))
	for _, a := range conf.Analyzers {
		if err := c.startAnalyzer(a); err != nil {
			return err
		}
	}

	if err := c.initPoster(); err != nil {
		return err
	}

	t := &roundTripper{
		Log:      log.DefaultLogger,
		User:     c.GithubUser,
		Password: c.GithubToken,
	}
	watcher, err := github.NewWatcher(t, &lookout.WatchOptions{
		URL: c.Positional.Repository,
	})
	if err != nil {
		return err
	}

	return watcher.Watch(context.Background(), c.handleEvent)
}

func (c *ServeCommand) initPoster() error {
	if c.DryRun {
		c.poster = &LogPoster{log.DefaultLogger}
	} else {
		c.poster = github.NewPoster(&roundTripper{
			Log:      log.DefaultLogger,
			User:     c.GithubUser,
			Password: c.GithubToken,
		})
	}

	return nil
}

func (c *ServeCommand) startAnalyzer(conf AnalyzerConfig) error {
	addr, err := lookout.ToGoGrpcAddress(conf.Addr)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}

	c.analyzers[conf.Name] = lookout.NewAnalyzerClient(conn)
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

	gitService := git.NewService(loader)
	bblfshService := bblfsh.NewService(gitService, gitService, bblfshConn)

	srv := &lookout.DataServerHandler{
		ChangeGetter: bblfshService,
		FileGetter:   bblfshService,
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
	case *lookout.ReviewEvent:
		return c.handlePR(ev)
	default:
		log.Debugf("ignoring unsupported event: %s", ev)
		return nil
	}
}

func (c *ServeCommand) handlePR(e *lookout.ReviewEvent) error {
	logger := log.DefaultLogger.With(log.Fields{
		"provider":   e.Provider,
		"repository": e.Head.InternalRepositoryURL,
		"head":       e.Head.ReferenceName,
	})
	logger.Infof("processing pull request")

	var comments []*lookout.Comment

	for name, a := range c.analyzers {
		aLogger := logger.With(log.Fields{
			"analyzer": name,
		})

		resp, err := a.NotifyReviewEvent(context.TODO(), e)
		if err != nil {
			aLogger.Errorf(err, "analysis failed")
			return nil
		}

		if len(resp.Comments) == 0 {
			aLogger.Infof("no comments were produced")
			continue
		}

		comments = append(comments, resp.Comments...)
	}

	if len(comments) == 0 {
		logger.With(log.Fields{
			"comments": len(comments),
		}).Infof("posting analysis")

		if err := c.poster.Post(context.TODO(), e, comments); err != nil {
			logger.Errorf(err, "posting analysis failed")
		}
	}

	return nil
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

type roundTripper struct {
	Log      log.Logger
	Base     http.RoundTripper
	User     string
	Password string
}

func (t *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Log.With(log.Fields{
		"url":  req.URL.String(),
		"user": t.User,
	}).Debugf("http request")

	if t.User != "" {
		req.SetBasicAuth(t.User, t.Password)
	}

	rt := t.Base
	if rt == nil {
		rt = http.DefaultTransport
	}

	return rt.RoundTrip(req)
}

var _ http.RoundTripper = &roundTripper{}
