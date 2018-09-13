package github

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cache"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	log "gopkg.in/src-d/go-log.v1"
)

// ClientPoolEventType type of the change in ClientPool
type ClientPoolEventType string

const (
	// ClientPoolEventAdd happens when new client is added in the pool
	ClientPoolEventAdd ClientPoolEventType = "add"
	// ClientPoolEventRemove happens when client is removed from the pool
	ClientPoolEventRemove ClientPoolEventType = "remove"
)

// ClientPoolEvent defines change in ClientPool
type ClientPoolEvent struct {
	Type   ClientPoolEventType
	Client *Client
}

// ClientPool holds mapping of repositories to clients
type ClientPool struct {
	byClients map[*Client][]*lookout.RepositoryInfo
	byRepo    map[string]*Client

	subs map[chan ClientPoolEvent]bool
}

// NewClientPool creates new pool of clients with repositories
func NewClientPool() *ClientPool {
	return &ClientPool{
		byClients: make(map[*Client][]*lookout.RepositoryInfo),
		byRepo:    make(map[string]*Client),
		subs:      make(map[chan ClientPoolEvent]bool),
	}
}

// Clients returns map[Client]RepositoryInfo
func (p *ClientPool) Clients() map[*Client][]*lookout.RepositoryInfo {
	return p.byClients
}

// Client returns client, ok by username and repository name
func (p *ClientPool) Client(username, repo string) (*Client, bool) {
	c, ok := p.byRepo[username+"/"+repo]
	return c, ok
}

// Repos returns list of repositories in the pool
func (p *ClientPool) Repos() []string {
	var rps []string
	for r := range p.byRepo {
		rps = append(rps, r)
	}

	return rps
}

// ReposByClient returns list of repositories by client
func (p *ClientPool) ReposByClient(c *Client) []*lookout.RepositoryInfo {
	return p.byClients[c]
}

// Update updates list of repositories for a client
func (p *ClientPool) Update(c *Client, newRepos []*lookout.RepositoryInfo) {
	if len(newRepos) == 0 {
		p.RemoveClient(c)
		return
	}

	repos, ok := p.byClients[c]
	if !ok {
		for _, r := range newRepos {
			p.byRepo[r.FullName] = c
		}

		p.byClients[c] = newRepos

		p.notify(ClientPoolEvent{
			Type:   ClientPoolEventAdd,
			Client: c,
		})

		return
	}

	// delete old repos
	var reposAfterDelete []*lookout.RepositoryInfo
	for _, repo := range repos {
		found := false
		for _, newRepo := range newRepos {
			if repo == newRepo {
				found = true
				break
			}
		}

		if found {
			reposAfterDelete = append(reposAfterDelete, repo)
		} else {
			delete(p.byRepo, repo.FullName)
		}
	}
	p.byClients[c] = reposAfterDelete

	// add new repos
	for _, newRepo := range newRepos {
		if _, ok := p.byRepo[newRepo.FullName]; ok {
			continue
		}

		p.byRepo[newRepo.FullName] = c
		p.byClients[c] = append(p.byClients[c], newRepo)
	}
}

// RemoveClient removes client from the pool and notifies about it
func (p *ClientPool) RemoveClient(c *Client) {
	p.notify(ClientPoolEvent{
		Type:   ClientPoolEventRemove,
		Client: c,
	})

	for repo, client := range p.byRepo {
		if client == c {
			delete(p.byRepo, repo)
		}
	}

	delete(p.byClients, c)
}

// Subscribe allows to subscribe to changes in the pool
func (p *ClientPool) Subscribe(ch chan ClientPoolEvent) {
	p.subs[ch] = true
}

// Unsubscribe stops sending changes to the channel
func (p *ClientPool) Unsubscribe(ch chan ClientPoolEvent) {
	delete(p.subs, ch)
}

func (p *ClientPool) notify(e ClientPoolEvent) {
	for ch := range p.subs {
		// use non-blocking send
		select {
		case ch <- e:
		default:
		}
	}
}

// Client is a wrapper for github.Client that supports cache and provides rate limit information
type Client struct {
	*github.Client
	cache            *cache.ValidableCache
	limitRT          *limitRoundTripper
	watchMinInterval time.Duration
}

// NewClient creates new Client
func NewClient(t http.RoundTripper, cache *cache.ValidableCache, watchMinInterval string) *Client {
	limitRT := &limitRoundTripper{
		Base: t,
	}

	cachedT := httpcache.NewTransport(cache)
	cachedT.MarkCachedResponses = true
	cachedT.Transport = limitRT

	interval := minInterval
	if watchMinInterval != "" {
		d, err := time.ParseDuration(watchMinInterval)
		if err != nil {
			log.Errorf(err, "can't parse min interval %q", watchMinInterval)
		} else {
			interval = d
		}
	}

	return &Client{
		Client:           github.NewClient(&http.Client{Transport: cachedT}),
		cache:            cache,
		limitRT:          limitRT,
		watchMinInterval: interval,
	}
}

// Rate returns last github.Rate for a client by category
func (c *Client) Rate(cat rateLimitCategory) github.Rate {
	return c.limitRT.Rate(cat)
}

// PollInterval returns last duration from X-Poll-Interval for a client by category
func (c *Client) PollInterval(cat pollLimitCategory) time.Duration {
	return c.limitRT.PollInterval(cat)
}

// Validate validates cache by path
func (c *Client) Validate(path string) error {
	return c.cache.Validate(path)
}

type rateLimitCategory uint8
type pollLimitCategory uint8

const (
	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
	headerPollInterval  = "X-Poll-Interval"
)

const (
	coreCategory rateLimitCategory = iota
	searchCategory

	categories // An array of this length will be able to contain all rate limit categories.
)

const (
	eventsCategory pollLimitCategory = iota
	notificationsCategory
	unknownCategory // in case some new endpoint starts return X-Poll-Interval

	pollCategories
)

// category returns the rate limit category of the endpoint, determined by Request.URL.Path.
func category(path string) rateLimitCategory {
	switch {
	default:
		return coreCategory
	case strings.HasPrefix(path, "/search/"):
		return searchCategory
	}
}

// pollCategory returns the poll limit category of the endpoint, determined by Request.URL.Path.
// TODO(max): cover all cases
func pollCategory(path string) pollLimitCategory {
	switch {
	case strings.HasSuffix(path, "/events"):
		return eventsCategory
	case strings.HasSuffix(path, "/notifications"):
		return notificationsCategory
	default:
		return unknownCategory
	}
}

type limitRoundTripper struct {
	Base http.RoundTripper

	// rateLimits for the client as determined by the most recent API calls.
	rateLimits [categories]github.Rate
	// pollInterval for the client by endpoint as determined by the most recent API calls
	pollIntervals [pollCategories]time.Duration

	rateMu sync.Mutex
}

func (t *limitRoundTripper) Rate(c rateLimitCategory) github.Rate {
	t.rateMu.Lock()
	defer t.rateMu.Unlock()
	return t.rateLimits[c]
}

func (t *limitRoundTripper) PollInterval(c pollLimitCategory) time.Duration {
	t.rateMu.Lock()
	defer t.rateMu.Unlock()
	return t.pollIntervals[c]
}

func (t *limitRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.Base
	if rt == nil {
		rt = http.DefaultTransport
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	logFields := log.Fields{}

	t.rateMu.Lock()
	rate := t.rateLimits[category(req.URL.Path)]
	if limit := resp.Header.Get(headerRateLimit); limit != "" {
		rate.Limit, _ = strconv.Atoi(limit)
		logFields["rate-limit"] = rate.Limit
	}

	if remaining := resp.Header.Get(headerRateRemaining); remaining != "" {
		rate.Remaining, _ = strconv.Atoi(remaining)
		logFields["rate-limit"] = rate.Remaining
	}

	if reset := resp.Header.Get(headerRateReset); reset != "" {
		if v, _ := strconv.ParseInt(reset, 10, 64); v != 0 {
			rate.Reset = github.Timestamp{time.Unix(v, 0)}
		}
		logFields["reset-at"] = rate.Reset
	}

	if pollInterval := resp.Header.Get(headerPollInterval); pollInterval != "" {
		secs, _ := strconv.Atoi(pollInterval)
		duration := time.Duration(secs) * time.Second
		t.pollIntervals[pollCategory(req.URL.Path)] = duration
		logFields["poll-interval"] = duration
	}

	t.rateLimits[category(req.URL.Path)] = rate
	t.rateMu.Unlock()

	ctxlog.Get(req.Context()).With(logFields).Debugf("http request to %s", req.URL.Path)

	return resp, err
}

var _ http.RoundTripper = &limitRoundTripper{}
