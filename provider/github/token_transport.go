package github

import (
	"context"
	"net/http"
	"strings"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cache"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
	log "gopkg.in/src-d/go-log.v1"
)

// ClientConfig holds github username, token and watch interval
type ClientConfig struct {
	User        string
	Token       string
	MinInterval string
}

var zeroClientConfig = ClientConfig{}

// IsZero return true if config is empty and false otherwise
func (c ClientConfig) IsZero() bool {
	return c == zeroClientConfig
}

// NewClientPoolFromTokens creates new ClientPool based on map[repoURL]ClientConfig
// later we will need another constructor that would request installations and create pool from it
func NewClientPoolFromTokens(urlToConfig map[string]ClientConfig, cache *cache.ValidableCache) (*ClientPool, error) {
	byConfig := make(map[ClientConfig][]*lookout.RepositoryInfo)

	for url, c := range urlToConfig {
		repo, err := vcsurl.Parse(url)
		if err != nil {
			return nil, err
		}

		byConfig[c] = append(byConfig[c], repo)
	}

	byClients := make(map[*Client][]*lookout.RepositoryInfo, len(byConfig))
	byRepo := make(map[string]*Client, len(urlToConfig))
	for conf, repos := range byConfig {
		client := NewClient(&roundTripper{
			User:     conf.User,
			Password: conf.Token,
		}, cache, conf.MinInterval)

		if _, ok := byClients[client]; !ok {
			byClients[client] = []*lookout.RepositoryInfo{}
		}

		byClients[client] = append(byClients[client], repos...)
		for _, r := range repos {
			byRepo[r.FullName] = client
		}
	}

	pool := &ClientPool{
		byClients: byClients,
		byRepo:    byRepo,
	}
	return pool, nil
}

// NewClientPoolInstallations creates a new ClientPool using the App ID and
// private key set in providerConf
func NewClientPoolInstallations(
	cache *cache.ValidableCache,
	providerConf ProviderConfig,
) (*ClientPool, error) {
	// Use App authorization to list installations
	appTr, err := ghinstallation.NewAppsTransportKeyFromFile(
		http.DefaultTransport, providerConf.AppID, providerConf.PrivateKey)
	if err != nil {
		return nil, err
	}

	appClient := github.NewClient(&http.Client{Transport: appTr})
	app, _, err := appClient.Apps.Get(context.TODO(), "")
	if err != nil {
		return nil, err
	}

	log.Infof("authorized as GitHub application %q, ID %v", app.GetName(), app.GetID())

	installations, _, err := appClient.Apps.ListInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	byClients := make(map[*Client][]*lookout.RepositoryInfo, len(installations))
	byRepo := make(map[string]*Client)
	allRepos := make([]string, 0)

	// Create a client for each installation, assign it to its repos
	for _, installation := range installations {
		installationID := int(installation.GetID())

		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport,
			providerConf.AppID, installationID, providerConf.PrivateKey)
		if err != nil {
			return nil, err
		}

		// TODO (carlosms): hardcoded, take from config
		watchMinInterval := ""
		iClient := NewClient(itr, cache, watchMinInterval)

		ghRepos, _, err := iClient.Apps.ListRepos(context.TODO(), &github.ListOptions{})
		if err != nil {
			return nil, err
		}

		repos := make([]*lookout.RepositoryInfo, len(ghRepos))

		for i, ghRepo := range ghRepos {
			log.Debugf("installation %v has repo %s", installationID, *ghRepo.HTMLURL)

			repo, err := vcsurl.Parse(*ghRepo.HTMLURL)
			if err != nil {
				return nil, err
			}

			repos[i] = repo
			byRepo[repo.FullName] = iClient
			allRepos = append(allRepos, repo.FullName)
		}

		byClients[iClient] = repos
	}

	log.Infof("found %v repositories using the GitHub application: %v",
		len(allRepos), strings.Join(allRepos, ","))

	pool := &ClientPool{
		byClients: byClients,
		byRepo:    byRepo,
	}
	return pool, nil
}

type roundTripper struct {
	Base     http.RoundTripper
	User     string
	Password string
}

func (t *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctxlog.Get(req.Context()).With(log.Fields{
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
