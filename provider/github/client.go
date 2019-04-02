package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/service/git"
	"github.com/src-d/lookout/util/cache"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
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

// keep default into our package to be able to override them in tests
var (
	defaultBaseURL       = "https://api.github.com/"
	defaultUploadBaseURL = "https://uploads.github.com/"
)

// ClientPoolEvent defines change in ClientPool
type ClientPoolEvent struct {
	Type   ClientPoolEventType
	Client *Client
}

// repositoryInfo wraps a lookout.RepositoryInfo adding an Organization ID
type repositoryInfo struct {
	lookout.RepositoryInfo
	// OrganizationID is this repository's organization
	OrganizationID string
}

// ClientPool holds mapping of repositories to clients
type ClientPool struct {
	byClients map[*Client][]*repositoryInfo
	byRepo    map[string]*Client
	mutex     sync.Mutex

	subs      map[chan ClientPoolEvent]bool
	subsMutex sync.Mutex
}

// NewClientPool creates new pool of clients with repositories
func NewClientPool() *ClientPool {
	return &ClientPool{
		byClients: make(map[*Client][]*repositoryInfo),
		byRepo:    make(map[string]*Client),
		subs:      make(map[chan ClientPoolEvent]bool),
	}
}

// newClientPoolFromClients creates a new pool of clients based on the given
// clients and repositories
func newClientPoolFromClients(
	byClients map[*Client][]*repositoryInfo,
	byRepo map[string]*Client) *ClientPool {

	return &ClientPool{
		byClients: byClients,
		byRepo:    byRepo,
		subs:      make(map[chan ClientPoolEvent]bool),
	}
}

// Clients returns map[Client]RepositoryInfo
func (p *ClientPool) Clients() map[*Client][]*repositoryInfo {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Create the target map
	copyMap := make(map[*Client][]*repositoryInfo, len(p.byClients))

	// Copy from the original map to the target map
	for key, value := range p.byClients {
		copyMap[key] = value
	}

	return copyMap
}

// Client returns client, ok by username and repository name
func (p *ClientPool) Client(username, repo string) (*Client, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	c, ok := p.byRepo[username+"/"+repo]
	return c, ok
}

// Repos returns list of repositories in the pool
func (p *ClientPool) Repos() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var rps []string
	for r := range p.byRepo {
		rps = append(rps, r)
	}

	return rps
}

// ReposByClient returns list of repositories by client
func (p *ClientPool) ReposByClient(c *Client) []*repositoryInfo {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.byClients[c]
}

// Update updates list of repositories for a client
func (p *ClientPool) Update(c *Client, newRepos []*repositoryInfo) {
	if len(newRepos) == 0 {
		p.RemoveClient(c)
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

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
	var reposAfterDelete []*repositoryInfo
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
	p.mutex.Lock()
	defer p.mutex.Unlock()

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
	p.subsMutex.Lock()
	defer p.subsMutex.Unlock()

	p.subs[ch] = true
}

// Unsubscribe stops sending changes to the channel
func (p *ClientPool) Unsubscribe(ch chan ClientPoolEvent) {
	p.subsMutex.Lock()
	defer p.subsMutex.Unlock()

	delete(p.subs, ch)
}

func (p *ClientPool) notify(e ClientPoolEvent) {
	p.subsMutex.Lock()
	defer p.subsMutex.Unlock()

	for ch := range p.subs {
		// use non-blocking send
		select {
		case ch <- e:
		default:
		}
	}
}

var _ git.AuthProvider = &ClientPool{}

// GitAuth returns a go-git auth method for a repo
func (p *ClientPool) GitAuth(ctx context.Context, repoInfo *lookout.RepositoryInfo) transport.AuthMethod {
	c, ok := p.Client(repoInfo.Owner, repoInfo.Name)
	if !ok {
		return nil
	}

	return c.gitAuth(ctx)
}

type gitAuthFn = func(ctx context.Context) transport.AuthMethod

// Client is a wrapper for github.Client that supports cache and provides rate limit information
type Client struct {
	*github.Client
	cache            *cache.ValidableCache
	limitRT          *limitRoundTripper
	watchMinInterval time.Duration
	gitAuth          gitAuthFn

	mutex    sync.Mutex
	username string
}

// NewClient creates new Client.
// A timeout of zero means no timeout.
func NewClient(
	t http.RoundTripper,
	cache *cache.ValidableCache,
	watchMinInterval string,
	gitAuth gitAuthFn,
	timeout time.Duration,
) *Client {
	fixT := &fixReviewTransport{
		Transport: t,
	}

	limitRT := &limitRoundTripper{
		Base: fixT,
	}

	interval := minInterval
	if watchMinInterval != "" {
		d, err := time.ParseDuration(watchMinInterval)
		if err != nil {
			log.Errorf(err, "can't parse min interval %q", watchMinInterval)
		} else {
			interval = d
		}
	}

	ghClient, _ := github.NewEnterpriseClient(
		defaultBaseURL,
		defaultUploadBaseURL,
		&http.Client{
			Transport: limitRT,
			Timeout:   timeout,
		})
	return &Client{
		Client:           ghClient,
		cache:            cache,
		limitRT:          limitRT,
		watchMinInterval: interval,
		gitAuth:          gitAuth,
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

// Username returns name of the user for the current client
func (c *Client) Username() (string, error) {
	if c.username != "" {
		return c.username, nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.username != "" {
		return c.username, nil
	}

	u, _, err := c.Users.Get(context.Background(), "me")
	if err != nil {
		return "", err
	}

	c.username = u.GetName()
	return c.username, nil
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

	logFields := log.Fields{"url": req.URL}

	t.rateMu.Lock()
	rate := t.rateLimits[category(req.URL.Path)]
	if limit := resp.Header.Get(headerRateLimit); limit != "" {
		rate.Limit, _ = strconv.Atoi(limit)
		logFields["rate.limit"] = rate.Limit
	}

	if remaining := resp.Header.Get(headerRateRemaining); remaining != "" {
		rate.Remaining, _ = strconv.Atoi(remaining)
		logFields["rate.remaining"] = rate.Remaining
	}

	if reset := resp.Header.Get(headerRateReset); reset != "" {
		if v, _ := strconv.ParseInt(reset, 10, 64); v != 0 {
			rate.Reset = github.Timestamp{time.Unix(v, 0)}
		}
		logFields["rate.reset-at"] = rate.Reset
	}

	if pollInterval := resp.Header.Get(headerPollInterval); pollInterval != "" {
		secs, _ := strconv.Atoi(pollInterval)
		duration := time.Duration(secs) * time.Second
		t.pollIntervals[pollCategory(req.URL.Path)] = duration
		logFields["poll-interval"] = duration
	}

	t.rateLimits[category(req.URL.Path)] = rate
	t.rateMu.Unlock()

	ctxlog.Get(req.Context()).With(logFields).Debugf("http request with GitHub rate limit")

	return resp, err
}

var _ http.RoundTripper = &limitRoundTripper{}

func handleAPIError(resp *github.Response, err error, msg string) error {
	if err != nil {
		if e, ok := err.(*github.ErrorResponse); ok {
			if e.Response == nil {
				e.Response = resp.Response
			} else if e.Response.Request == nil {
				e.Response.Request = resp.Response.Request
			}
		}

		return ErrGitHubAPI.Wrap(err, msg)
	}

	if resp.StatusCode == 200 {
		return nil
	}

	return ErrGitHubAPI.Wrap(
		fmt.Errorf("bad HTTP status: %d", resp.StatusCode),
		msg,
	)
}

// ValidateTokenPermissions checks that client has necessary permissions required by lookout
// returns error if any is missed
func ValidateTokenPermissions(client *Client) error {
	// actually we don't need ALL repo scope but we ask for it in documentation
	// so let's be consistent
	required := []string{"repo"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// authorizations api can be accessed only with username and password, not token
	// read headers of any endpoint instead
	_, r, err := client.Users.Get(ctx, "me")
	if err != nil {
		return err
	}

	scopes := strings.Split(r.Header.Get("X-Oauth-Scopes"), ",")
	scopesMap := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		scopesMap[strings.TrimSpace(scope)] = true
	}

	for _, scope := range required {
		_, ok := scopesMap[scope]
		if !ok {
			return fmt.Errorf("token doesn't have permission scope '%s'", scope)
		}
	}

	return nil
}

// CanPostStatus check if the client has push access to a repository
// which is required for updating status. It assumes client has correct scope.
// returns error if it permission is missed
func CanPostStatus(client *Client, repo *repositoryInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := client.Username()
	if err != nil {
		return err
	}

	r, _, err := client.Repositories.GetPermissionLevel(ctx, repo.Owner, repo.Name, user)
	if err != nil {
		return err
	}

	if r.GetPermission() != "admin" && r.GetPermission() != "write" {
		return fmt.Errorf("client does not have write access to repository %s", repo.FullName)
	}

	return err
}
