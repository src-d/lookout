package github

import (
	"net/http"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cache"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
	log "gopkg.in/src-d/go-log.v1"
)

// ClientConfig holds github username, token and watch interval
type ClientConfig struct {
	User        string
	Token       string
	MinInterval string
}

var zeroClientConfig = &ClientConfig{}

// IsZero return true if config is empty and false otherwise
func (c *ClientConfig) IsZero() bool {
	return c == zeroClientConfig
}

// NewClientPoolFromTokens creates new ClientPool based on map[repoURL]ClientConfig
// later we will need another constructor that would request installations and create pool from it
func NewClientPoolFromTokens(urlToConfig map[string]ClientConfig, defaultConfig ClientConfig, cache *cache.ValidableCache) (*ClientPool, error) {
	byConfig := make(map[ClientConfig][]*lookout.RepositoryInfo)

	for url, c := range urlToConfig {
		repo, err := vcsurl.Parse(url)
		if err != nil {
			return nil, err
		}

		if c.IsZero() {
			c = defaultConfig
		}

		byConfig[c] = append(byConfig[c], repo)
	}

	byClients := make(map[*Client][]*lookout.RepositoryInfo, len(byConfig))
	byRepo := make(map[string]*Client, len(urlToConfig))
	for conf, repos := range byConfig {
		client := NewClient(&roundTripper{
			Log:      log.DefaultLogger,
			User:     conf.User,
			Password: conf.Token,
		}, cache, log.DefaultLogger, conf.MinInterval)

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
