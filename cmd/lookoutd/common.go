package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/provider/github"
	"github.com/src-d/lookout/provider/json"
	queue_util "github.com/src-d/lookout/queue"
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

	"github.com/gregjones/httpcache/diskcache"
	"github.com/jinzhu/copier"
	"github.com/sanity-io/litter"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-billy.v4/osfs"
	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
	yaml "gopkg.in/yaml.v2"
)

type startFunc func() error
type stopFunc func()

type lookoutdBaseCommand struct {
	cli.LogOptions
	ConfigFile string `long:"config" short:"c" default:"config.yml" env:"LOOKOUT_CONFIG_FILE" description:"path to configuration file"`
}

// lookoutdCommand represents the common options for serve, watch, work
type lookoutdCommand struct {
	lookoutdBaseCommand

	GithubUser  string `long:"github-user" env:"GITHUB_USER" description:"user for the GitHub API"`
	GithubToken string `long:"github-token" env:"GITHUB_TOKEN" description:"access token for the GitHub API"`
	Provider    string `long:"provider" choice:"github" choice:"json" default:"github" env:"LOOKOUT_PROVIDER" description:"provider name: github, json"`
	ProbesAddr  string `long:"probes-addr" default:"0.0.0.0:8090" env:"LOOKOUT_PROBES_ADDRESS" description:"TCP address to bind the health probe endpoints"`

	pool           *github.ClientPool
	probeReadiness bool
	conf           Config
}

// Init implements the go-cli initializer interface. Initializes logs
// and Config file based on the cli options
func (c *lookoutdCommand) Init(a *gocli.App) error {
	c.lookoutdBaseCommand.Init(a)

	var err error
	c.conf, err = c.initConfig()
	return err
}

// queueConsumerCommand represents the common options for serve, work
type queueConsumerCommand struct {
	lookoutdCommand
	cli.DBOptions

	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd    string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
	DryRun     bool   `long:"dry-run" env:"LOOKOUT_DRY_RUN" description:"analyze repositories and log the result without posting code reviews to GitHub"`
	Library    string `long:"library" default:"/tmp/lookout" env:"LOOKOUT_LIBRARY" description:"path to the lookout library"`
	Workers    int    `long:"workers" env:"LOOKOUT_WORKERS" default:"1" description:"number of concurrent workers processing events, 0 means the same number as processors"`

	analyzers map[string]lookout.AnalyzerClient
}

var defaultInstallationsSyncInterval = 5 * time.Minute

// Config holds the main configuration
type Config struct {
	server.Config `yaml:",inline"`
	Providers     struct {
		Github github.ProviderConfig
	}
	Repositories []RepoConfig
	Timeout      TimeoutConfig
}

// RepoConfig holds configuration for repository, support only github provider
type RepoConfig struct {
	URL    string
	Client github.ClientConfig
}

// TimeoutConfig holds configuration for timeouts
type TimeoutConfig struct {
	AnalyzerReview time.Duration `yaml:"analyzer_review"`
	AnalyzerPush   time.Duration `yaml:"analyzer_push"`
	GithubRequest  time.Duration `yaml:"github_request"`
	GitFetch       time.Duration `yaml:"git_fetch"`
	BblfshParse    time.Duration `yaml:"bblfsh_parse"`
}

func (c *lookoutdCommand) initConfig() (Config, error) {
	var conf Config
	configData, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		// Special case for #289. When using docker-compose, if 'config.yml' does
		// not exist the volume will be mounted as an empty directory
		// named 'config.yml'
		fi, errStat := os.Stat(c.ConfigFile)
		if c.ConfigFile == "config.yml" && errStat == nil && fi.IsDir() {
			return conf, fmt.Errorf("Can't open configuration file. If you are using docker-compose, make sure './config.yml' exists")
		}

		return conf, fmt.Errorf("Can't open configuration file: %s", err)
	}

	// Set default timeouts
	conf.Timeout = TimeoutConfig{
		AnalyzerReview: 10 * time.Minute,
		AnalyzerPush:   60 * time.Minute,
		GithubRequest:  time.Minute,
		GitFetch:       20 * time.Minute,
		BblfshParse:    2 * time.Minute,
	}

	if err := yaml.Unmarshal([]byte(configData), &conf); err != nil {
		return conf, fmt.Errorf("Can't parse configuration file: %s", err)
	}

	c.logConfig(conf)

	return conf, nil
}

func logConfig(options interface{}, conf Config) {
	var confCp Config
	copier.Copy(&confCp, conf)

	confCp.Repositories = make([]RepoConfig, len(conf.Repositories))
	for i := range conf.Repositories {
		var repoConfigCp RepoConfig
		copier.Copy(&repoConfigCp, conf.Repositories[i])
		if repoConfigCp.Client.Token != "" {
			repoConfigCp.Client.Token = "****"
		}
		confCp.Repositories[i] = repoConfigCp
	}

	lt := litter.Options{
		Compact: true,
	}

	log.With(log.Fields{
		"options": lt.Sdump(options),
		"conf":    lt.Sdump(confCp),
		"version": version,
		"build":   build,
	}).Infof("starting %s", name)
}

func (c *lookoutdCommand) logConfig(conf Config) {
	var cCp lookoutdCommand
	copier.Copy(&cCp, c)

	cCp.GithubToken = "****"

	logConfig(cCp, conf)
}

func (c *queueConsumerCommand) logConfig(conf Config) {
	var cCp queueConsumerCommand
	copier.Copy(&cCp, c)

	cCp.DBOptions.DB = "****"
	cCp.GithubToken = "****"

	logConfig(cCp, conf)
}

func (c *lookoutdCommand) initProvider(conf Config) error {
	switch c.Provider {
	case github.Provider:
		if conf.Providers.Github.PrivateKey != "" || conf.Providers.Github.AppID != 0 {
			return c.initProviderGithubApp(conf)
		}

		return c.initProviderGithubToken(conf)
	}

	return nil
}

func (c *lookoutdCommand) initProviderGithubToken(conf Config) error {
	noDefaultAuth := c.GithubUser == "" || c.GithubToken == ""
	defaultConfig := github.ClientConfig{
		User:  c.GithubUser,
		Token: c.GithubToken,
	}

	repoToConfig := make(map[string]github.ClientConfig, len(conf.Repositories))
	for _, repo := range conf.Repositories {
		repoToConfig[repo.URL] = repo.Client
	}

	for url, config := range repoToConfig {
		if config.IsZero() {
			if noDefaultAuth {
				// Empty github auth is only useful for --dry-run,
				// we may want to enforce this as an error
				log.Warningf("missing authentication for repository %s, and no default provided", url)
			} else {
				log.Infof("using default authentication for repository %s", url)
			}

			repoToConfig[url] = defaultConfig
		}
	}

	cache := cache.NewValidableCache(diskcache.New("/tmp/github"))
	pool, err := github.NewClientPoolFromTokens(repoToConfig, cache, conf.Timeout.GithubRequest)
	if err != nil {
		return err
	}

	c.pool = pool
	return nil
}

func (c *lookoutdCommand) initProviderGithubApp(conf Config) error {
	if conf.Providers.Github.PrivateKey == "" {
		return fmt.Errorf("missing GitHub App private key filepath in config")
	}
	if conf.Providers.Github.AppID == 0 {
		return fmt.Errorf("missing GitHub App ID in config")
	}

	installationsSyncInterval := defaultInstallationsSyncInterval
	if conf.Providers.Github.InstallationSyncInterval != "" {
		var err error
		installationsSyncInterval, err = time.ParseDuration(conf.Providers.Github.InstallationSyncInterval)
		if err != nil {
			return fmt.Errorf("can't parse sync interval: %s", err)
		}
	}

	cache := cache.NewValidableCache(diskcache.New("/tmp/github"))
	insts, err := github.NewInstallations(
		conf.Providers.Github.AppID, conf.Providers.Github.PrivateKey,
		cache, conf.Providers.Github.WatchMinInterval, conf.Timeout.GithubRequest)
	if err != nil {
		return err
	}

	c.pool = insts.Pool

	go func() {
		for {
			if err := insts.Sync(); err != nil {
				log.Errorf(err, "can't sync installations with github")
			}
			time.Sleep(installationsSyncInterval)
		}
	}()

	return nil
}

func (c *lookoutdCommand) initWatcher(conf Config) (lookout.Watcher, error) {
	switch c.Provider {
	case github.Provider:
		watcher, err := github.NewWatcher(c.pool)
		if err != nil {
			return nil, err
		}

		return watcher, nil
	case json.Provider:
		return json.NewWatcher(os.Stdin)
	default:
		return nil, fmt.Errorf("provider %s not supported", c.Provider)
	}
}

func (c *lookoutdCommand) startHealthProbes() error {
	livenessPath := "/health/liveness"
	http.HandleFunc(livenessPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	readinessPath := "/health/readiness"
	http.HandleFunc(readinessPath, func(w http.ResponseWriter, r *http.Request) {
		if c.probeReadiness {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		} else {
			w.WriteHeader(500)
			w.Write([]byte("starting up"))
		}
	})

	log.With(log.Fields{
		"addr":  c.ProbesAddr,
		"paths": []string{livenessPath, readinessPath},
	}).Infof("listening to health probe HTTP requests")

	return http.ListenAndServe(c.ProbesAddr, nil)
}

func (c *queueConsumerCommand) initPoster(conf Config) (lookout.Poster, error) {
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

func (c *queueConsumerCommand) startAnalyzer(conf lookout.AnalyzerConfig) (lookout.AnalyzerClient, error) {
	if conf.Name == "" {
		return nil, fmt.Errorf("missing 'name' in analyzer config")
	}

	if conf.Addr == "" {
		return nil, fmt.Errorf("missing 'addr' in config for analyzer %s", conf.Name)
	}
	addr, err := pb.ToGoGrpcAddress(conf.Addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address '%s' in config for analyzer %s: %s", conf.Addr, conf.Name, err)
	}

	ctx := context.Background()
	conn, err := grpchelper.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create a client connection to address '%s' in config for analyzer %s: %s", conf.Addr, conf.Name, err)
	}

	go grpchelper.LogConnStatusChanges(ctx, log.DefaultLogger.With(log.Fields{
		"analyzer": conf.Name,
		"addr":     conf.Addr,
	}), conn)

	return lookout.NewAnalyzerClient(conn), nil
}

func (c *queueConsumerCommand) initDataHandler(conf Config) (*lookout.DataServerHandler, error) {
	var err error
	c.Bblfshd, err = pb.ToGoGrpcAddress(c.Bblfshd)
	if err != nil {
		return nil, err
	}

	bblfshConn, err := grpchelper.DialContext(context.Background(), c.Bblfshd, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	var authProvider git.AuthProvider
	if c.Provider == github.Provider {
		if c.pool == nil {
			return nil, fmt.Errorf("pool must be initialized with initProvider")
		}

		authProvider = c.pool
	}

	lib := git.NewLibrary(osfs.New(c.Library))
	sync := git.NewSyncer(lib, authProvider, conf.Timeout.GitFetch)
	loader := git.NewLibraryCommitLoader(lib, sync)

	gitService := git.NewService(loader)
	enryService := enry.NewService(gitService, gitService)
	bblfshService := bblfsh.NewService(enryService, enryService, bblfshConn, conf.Timeout.BblfshParse)
	purgeService := purge.NewService(bblfshService, bblfshService)

	srv := &lookout.DataServerHandler{
		ChangeGetter: purgeService,
		FileGetter:   purgeService,
	}

	return srv, nil
}

func (c *queueConsumerCommand) initDataServer(srv *lookout.DataServerHandler) (startFunc, stopFunc) {
	var grpcSrv *grpc.Server

	start := func() error {
		var err error
		grpcSrv, err = grpchelper.NewBblfshProxyServer(c.Bblfshd)
		if err != nil {
			return err
		}

		lookout.RegisterDataServer(grpcSrv, srv)
		lis, err := pb.Listen(c.DataServer)
		if err != nil {
			return err
		}

		return grpcSrv.Serve(lis)
	}

	stop := func() {
		if grpcSrv == nil {
			return
		}

		grpcSrv.GracefulStop()
	}

	return start, stop
}

func (c *queueConsumerCommand) initDBOperators(db *sql.DB) (*store.DBEventOperator, *store.DBCommentOperator) {
	reviewStore := models.NewReviewEventStore(db)
	reviewTargetStore := models.NewReviewTargetStore(db)
	eventOp := store.NewDBEventOperator(
		reviewStore,
		reviewTargetStore,
		models.NewPushEventStore(db),
	)
	commentsOp := store.NewDBCommentOperator(
		models.NewCommentStore(db),
		reviewStore,
		reviewTargetStore,
	)

	return eventOp, commentsOp
}

func (c *queueConsumerCommand) initAnalyzers(conf Config) (map[string]lookout.Analyzer, error) {
	analyzers := make(map[string]lookout.Analyzer)
	for _, aConf := range conf.Analyzers {
		if aConf.Disabled {
			continue
		}
		client, err := c.startAnalyzer(aConf)
		if err != nil {
			return nil, err
		}
		analyzers[aConf.Name] = lookout.Analyzer{
			Client: client,
			Config: aConf,
		}
	}

	return analyzers, nil
}

func (c *lookoutdCommand) runEventEnqueuer(
	ctx context.Context,
	qOpt cli.QueueOptions,
	watcher lookout.Watcher,
) error {
	return cli.RunWatcher(
		ctx,
		watcher,
		lookout.CachedHandler(queue_util.EventEnqueuer(ctx, qOpt.Q)))
}

func (c *queueConsumerCommand) runEventDequeuer(ctx context.Context, qOpt cli.QueueOptions, server *server.Server) error {
	if c.Workers <= 0 {
		c.Workers = runtime.NumCPU()
		log.Infof("option --workers is 0, it will be set to the number of processors: %d", c.Workers)
	}

	return queue_util.RunEventDequeuer(ctx, qOpt.Q, server.HandleEvent, c.Workers)
}
