package github

import (
	"net/http"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cache"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
	log "gopkg.in/src-d/go-log.v1"
)

// UserToken holds github username and token
type UserToken struct {
	User        string
	Token       string
	MinInterval string
}

// NewClientPoolFromTokens creates new ClientPool based on map[repoURL]UserToken
// later we will need another constructor that would request installations and create pool from it
func NewClientPoolFromTokens(urls map[string]UserToken, defaultToken UserToken, cache *cache.ValidableCache) (*ClientPool, error) {
	byToken := make(map[UserToken][]*lookout.RepositoryInfo)
	emptyToken := UserToken{}

	for url, ut := range urls {
		repo, err := vcsurl.Parse(url)
		if err != nil {
			return nil, err
		}

		if ut == emptyToken {
			ut = defaultToken
		}

		byToken[ut] = append(byToken[ut], repo)
	}

	byClients := make(map[*Client][]*lookout.RepositoryInfo, len(byToken))
	byRepo := make(map[string]*Client, len(urls))
	for token, repos := range byToken {
		client := NewClient(&roundTripper{
			Log:      log.DefaultLogger,
			User:     token.User,
			Password: token.Token,
		}, cache, log.DefaultLogger, token.MinInterval)

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
