package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/provider/github"
	"github.com/src-d/lookout/provider/json"
	"github.com/src-d/lookout/server"
	"github.com/src-d/lookout/service/bblfsh"
	"github.com/src-d/lookout/service/enry"
	"github.com/src-d/lookout/service/git"
	"github.com/src-d/lookout/service/purge"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/store/models"
	"github.com/src-d/lookout/util/cache"
	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/grpchelper"

	"github.com/golang-migrate/migrate"
	"github.com/gregjones/httpcache/diskcache"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	if _, err := app.AddCommand("serve", "run server", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}
}

type ServeCommand struct {
	cli.CommonOptions
	cli.DBOptions
	ConfigFile  string `long:"config" short:"c" default:"config.yml" env:"LOOKOUT_CONFIG_FILE" description:"path to configuration file"`
	GithubUser  string `long:"github-user" env:"GITHUB_USER" description:"user for the GitHub API"`
	GithubToken string `long:"github-token" env:"GITHUB_TOKEN" description:"access token for the GitHub API"`
	DataServer  string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd     string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	DryRun      bool   `long:"dry-run" env:"LOOKOUT_DRY_RUN" description:"analyze repositories and log the result without posting code reviews to GitHub"`
	Library     string `long:"library" default:"/tmp/lookout" env:"LOOKOUT_LIBRARY" description:"path to the lookout library"`
	Provider    string `long:"provider" default:"github" env:"LOOKOUT_PROVIDER" description:"provider name: github, json"`
	Positional  struct {
		Repository string `positional-arg-name:"repository"`
	} `positional-args:"yes" required:"yes"`

	analyzers map[string]lookout.AnalyzerClient
	pool      *github.ClientPool
}

// Config holds the main configuration
type Config struct {
	server.Config `yaml:",inline"`
	Providers     struct {
		Github github.ProviderConfig
	}
	Repositories []RepoConfig
}

// RepoConfig holds configuration for repository, support only github provider
type RepoConfig struct {
	URL    string
	Client github.ClientConfig
}

func (c *ServeCommand) Execute(args []string) error {
	var conf Config
	configData, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("Can't open configuration file: %s", err)
	}
	if err := yaml.Unmarshal([]byte(configData), &conf); err != nil {
		return fmt.Errorf("Can't parse configuration file: %s", err)
	}

	dataHandler, err := c.initDataHandler()
	if err != nil {
		return err
	}

	if err := c.startServer(dataHandler); err != nil {
		return err
	}

	db, err := c.initDB()
	if err != nil {
		return fmt.Errorf("Can't connect to the DB: %s", err)
	}

	reviewStore := models.NewReviewEventStore(db)
	eventOp := store.NewDBEventOperator(
		reviewStore,
		models.NewPushEventStore(db),
	)
	commentsOp := store.NewDBCommentOperator(
		models.NewCommentStore(db),
		reviewStore,
	)

	analyzers := make(map[string]lookout.Analyzer)
	for _, aConf := range conf.Analyzers {
		if aConf.Disabled {
			continue
		}
		client, err := c.startAnalyzer(aConf)
		if err != nil {
			return err
		}
		analyzers[aConf.Name] = lookout.Analyzer{
			Client: client,
			Config: aConf,
		}
	}

	err = c.initProvider(conf)
	if err != nil {
		return err
	}

	poster, err := c.initPoster(conf)
	if err != nil {
		return err
	}

	watcher, err := c.initWatcher(conf)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return server.NewServer(watcher, poster, dataHandler.FileGetter, analyzers, eventOp, commentsOp).Run(ctx)
}

func (c *ServeCommand) initProvider(conf Config) error {
	noDefaultAuth := c.GithubUser == "" || c.GithubToken == ""
	defaultConfig := github.ClientConfig{
		User:  c.GithubUser,
		Token: c.GithubToken,
	}

	switch c.Provider {
	case github.Provider:
		urls := strings.Split(c.Positional.Repository, ",")
		urlToConfig := make(map[string]github.ClientConfig, len(urls))
		repoToConfig := make(map[string]github.ClientConfig, len(conf.Repositories))
		for _, repo := range conf.Repositories {
			if !repo.Client.IsZero() {
				repoToConfig[repo.URL] = repo.Client
			}
		}

		for _, url := range urls {
			conf, ok := repoToConfig[url]
			if !ok {
				if noDefaultAuth {
					// Empty github auth is only useful for --dry-run,
					// we may want to enforce this as an error
					log.Warningf("missing authentication for repository %s, and no default provided", url)
				} else {
					log.Infof("using default authentication for repository %s", url)
				}

				conf = defaultConfig
			}
			urlToConfig[url] = conf
		}

		cache := cache.NewValidableCache(diskcache.New("/tmp/github"))
		pool, err := github.NewClientPoolFromTokens(urlToConfig, cache)
		if err != nil {
			return err
		}

		c.pool = pool
	}

	return nil
}

func (c *ServeCommand) initPoster(conf Config) (lookout.Poster, error) {
	if c.DryRun {
		return &server.LogPoster{log.DefaultLogger}, nil
	}

	switch c.Provider {
	case github.Provider:
		return github.NewPoster(c.pool, conf.Providers.Github), nil
	case json.Provider:
		return json.NewPoster(os.Stdout), nil
	default:
		return nil, fmt.Errorf("provider %s not supported", c.Provider)
	}
}

func (c *ServeCommand) initWatcher(conf Config) (lookout.Watcher, error) {
	switch c.Provider {
	case github.Provider:
		watcher, err := github.NewWatcher(c.pool, &lookout.WatchOptions{
			URLs: strings.Split(c.Positional.Repository, ","),
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

	ctx := context.Background()
	conn, err := grpchelper.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	go grpchelper.LogConnStatusChanges(ctx, log.With(log.Fields{
		"analyzer": conf.Name,
		"addr":     conf.Addr,
	}), conn)

	return lookout.NewAnalyzerClient(conn), nil
}

func (c *ServeCommand) initDataHandler() (*lookout.DataServerHandler, error) {
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
	enryService := enry.NewService(gitService, gitService)
	bblfshService := bblfsh.NewService(enryService, enryService, bblfshConn)
	purgeService := purge.NewService(bblfshService, bblfshService)

	srv := &lookout.DataServerHandler{
		ChangeGetter: purgeService,
		FileGetter:   purgeService,
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

func (c *ServeCommand) initDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.DB)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	m, err := store.NewMigrateInstance(db)
	if err != nil {
		return nil, err
	}

	dbVersion, _, err := m.Version()

	// The DB is not initialized
	if err == migrate.ErrNilVersion {
		return nil, fmt.Errorf("the DB is empty, it needs to be initialized with the 'lookout migrate' command")
	}

	if err != nil {
		return nil, err
	}

	maxVersion, err := store.MaxMigrateVersion()
	if err != nil {
		return nil, err
	}

	if dbVersion != maxVersion {
		return nil, fmt.Errorf(
			"database version mismatch. Current version is %v, but this binary (version %s, built on %s) needs version %v. "+
				"Use 'lookout migrate' to upgrade your database", dbVersion, version, build, maxVersion)
	}

	log.Debugf("the DB version is up to date, %v", dbVersion)
	log.Infof("connection with the DB established")
	return db, nil
}
