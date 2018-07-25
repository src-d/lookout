package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/provider/github"
	"github.com/src-d/lookout/provider/json"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/git"
	"github.com/src-d/lookout/util/flags"
	"github.com/src-d/lookout/util/grpchelper"

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

type ServeCommand struct {
	flags.CommonOptions
	ConfigFile  string `long:"config" short:"c" default:"config.yml" env:"LOOKOUT_CONFIG_FILE" description:"path to configuration file"`
	GithubUser  string `long:"github-user" env:"GITHUB_USER" description:"user for the GitHub API"`
	GithubToken string `long:"github-token" env:"GITHUB_TOKEN" description:"access token for the GitHub API"`
	DataServer  string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd     string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	DryRun      bool   `long:"dry-run" env:"LOOKOUT_DRY_RUN" description:"analyze repositories and log the result without posting code reviews to GitHub"`
	Library     string `long:"library" default:"/tmp/lookout" env:"LOOKOUT_LIBRARY" description:"path to the lookout library"`
	Provider    string `long:"provider" default:"github" env:"LOOKOUT_PROVIDER" description:"provider name: github, console"`
	Positional  struct {
		Repository string `positional-arg-name:"repository"`
	} `positional-args:"yes" required:"yes"`

	analyzers map[string]lookout.AnalyzerClient
}

func (c *ServeCommand) Execute(args []string) error {
	var conf lookout.ServerConfig
	configData, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("Can't open configuration file: %s", err)
	}
	if err := yaml.Unmarshal([]byte(configData), &conf); err != nil {
		return fmt.Errorf("Can't parse configuration file: %s", err)
	}

	setGrpcLogger()

	dataHandler, err := c.initDataHadler()
	if err != nil {
		return err
	}

	if err := c.startServer(dataHandler); err != nil {
		return err
	}

	analyzers := make(map[string]lookout.Analyzer, len(conf.Analyzers))
	for _, aConf := range conf.Analyzers {
		client, err := c.startAnalyzer(aConf)
		if err != nil {
			return err
		}
		analyzers[aConf.Name] = lookout.Analyzer{
			Client: client,
			Config: aConf,
		}
	}

	poster, err := c.initPoster()
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher()
	if err != nil {
		return err
	}

	ctx := context.Background()
	return lookout.NewServer(watcher, poster, dataHandler.FileGetter, analyzers).Run(ctx)
}

func (c *ServeCommand) initPoster() (lookout.Poster, error) {
	if c.DryRun {
		return &LogPoster{log.DefaultLogger}, nil
	}

	switch c.Provider {
	case github.Provider:
		return github.NewPoster(&roundTripper{
			Log:      log.DefaultLogger,
			User:     c.GithubUser,
			Password: c.GithubToken,
		}), nil
	case json.Provider:
		return json.NewPoster(os.Stdout), nil
	default:
		return nil, fmt.Errorf("provider %s not supported", c.Provider)
	}
}

func (c *ServeCommand) initWatcher() (lookout.Watcher, error) {
	switch c.Provider {
	case github.Provider:
		t := &roundTripper{
			Log:      log.DefaultLogger,
			User:     c.GithubUser,
			Password: c.GithubToken,
		}

		watcher, err := github.NewWatcher(t, &lookout.WatchOptions{
			URL: c.Positional.Repository,
		})
		if err != nil {
			return nil, err
		}

		return watcher, nil
	case json.Provider:
		return json.NewWatcher(os.Stdin, &lookout.WatchOptions{})
	default:
		return nil, fmt.Errorf("provider %s not supported", c.Provider)
	}
}

func (c *ServeCommand) startAnalyzer(conf lookout.AnalyzerConfig) (lookout.AnalyzerClient, error) {
	addr, err := grpchelper.ToGoGrpcAddress(conf.Addr)
	if err != nil {
		return nil, err
	}

	conn, err := grpchelper.DialContext(context.Background(), addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return lookout.NewAnalyzerClient(conn), nil
}

func (c *ServeCommand) initDataHadler() (*lookout.DataServerHandler, error) {
	var err error
	c.Bblfshd, err = grpchelper.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return nil, err
	}

	bblfshConn, err := grpchelper.DialContext(context.Background(), c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return nil, err
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

	return srv, nil
}

func (c *ServeCommand) startServer(srv *lookout.DataServerHandler) error {
	grpcSrv := grpchelper.NewServer()
	lookout.RegisterDataServer(grpcSrv, srv)
	lis, err := grpchelper.Listen(c.DataServer)
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
