package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-log.v1"
)

const Provider = "github"

// ProviderConfig represents the yml config
type ProviderConfig struct {
	CommentFooter string `yaml:"comment_footer"`
	PrivateKey    string `yaml:"private_key"`
	AppID         int    `yaml:"app_id"`
}

// don't call github more often than
var minInterval = 2 * time.Second

var (
	NoErrNotModified       = errors.NewKind("Not modified")
	ErrParsingEventPayload = errors.NewKind("Parse error in event")

	// RequestTimeout is the max time to wait until the request context is
	// cancelled.
	RequestTimeout = time.Second * 5
)

type Watcher struct {
	pool *ClientPool
	// maps clients to functions that stop watching the client
	stopFuncs map[*Client]func()
}

// NewWatcher returns a new
func NewWatcher(pool *ClientPool) (*Watcher, error) {
	return &Watcher{
		pool:      pool,
		stopFuncs: make(map[*Client]func()),
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	ctxlog.Get(ctx).With(log.Fields{"repos": w.pool.Repos()}).Infof("Starting watcher")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// channel for error from watch loops
	errCh := make(chan error)

	for client := range w.pool.Clients() {
		w.startClientLoops(ctx, client, cb, errCh)
	}

	go w.listenForChanges(ctx, cb, errCh)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if lookout.NoErrStopWatcher.Is(err) {
			return nil
		}
		return err
	}
}

func (w *Watcher) listenForChanges(ctx context.Context, cb lookout.EventHandler, errCh chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case change := <-w.pool.Changes:
			switch change.Type {
			case ClientPoolEventAdd:
				w.startClientLoops(ctx, change.Client, cb, errCh)
			case ClientPoolEventRemove:
				w.stopFuncs[change.Client]()
			default:
				errCh <- fmt.Errorf("unknown type of event from client pool %s", change.Type)
			}
		}
	}
}

func (w *Watcher) startClientLoops(
	ctx context.Context,
	client *Client,
	cb lookout.EventHandler,
	errCh chan error,
) {
	stopCh := make(chan bool)

	w.stopFuncs[client] = func() {
		// send event 2 times to stop both goroutines
		stopCh <- true
		stopCh <- true
		close(stopCh)
	}

	go w.watchLoop(ctx, client, w.processRepoPRs, cb, errCh, stopCh)
	go w.watchLoop(ctx, client, w.processRepoEvents, cb, errCh, stopCh)
}

type requestFun func(context.Context,
	*Client,
	*lookout.RepositoryInfo,
	lookout.EventHandler) (time.Duration, error)

func (w *Watcher) watchLoop(
	ctx context.Context,
	c *Client,
	requestFun requestFun,
	cb lookout.EventHandler,
	errCh chan error,
	stopCh chan bool,
) {
	for {
		for _, repo := range w.pool.ReposByClient(c) {
			categoryInterval, err := requestFun(ctx, c, repo, cb)

			if err != nil {
				errCh <- err
				return
			}

			interval := w.newInterval(c.Rate(coreCategory), c.watchMinInterval)
			if categoryInterval > interval {
				interval = categoryInterval
			}

			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-time.After(interval):
				continue
			}
		}
	}
}

func (w *Watcher) processRepoPRs(
	ctx context.Context,
	c *Client,
	repo *lookout.RepositoryInfo,
	cb lookout.EventHandler,
) (time.Duration, error) {
	resp, prs, err := w.doPRListRequest(ctx, repo.Username, repo.Name)
	if ErrGitHubAPI.Is(err) {
		ctxlog.Get(ctx).With(log.Fields{
			"repository": repo.FullName, "response": resp,
		}).Errorf(err, "request for PR list failed")
		return c.watchMinInterval, nil
	}

	if err != nil && !NoErrNotModified.Is(err) {
		return c.watchMinInterval, err
	}

	err = w.handlePrs(ctx, cb, repo, resp, prs)
	return c.watchMinInterval, err
}

func (w *Watcher) processRepoEvents(
	ctx context.Context,
	c *Client,
	repo *lookout.RepositoryInfo,
	cb lookout.EventHandler,
) (time.Duration, error) {
	resp, events, err := w.doEventRequest(ctx, repo.Username, repo.Name)
	if ErrGitHubAPI.Is(err) {
		ctxlog.Get(ctx).With(log.Fields{
			"repository": repo.FullName, "response": resp,
		}).Errorf(err, "request for events list failed")
		return c.PollInterval(eventsCategory), nil
	}

	if err != nil && !NoErrNotModified.Is(err) {
		return c.PollInterval(eventsCategory), err
	}

	err = w.handleEvents(ctx, cb, repo, resp, events)
	return c.PollInterval(eventsCategory), err
}

func (w *Watcher) handlePrs(ctx context.Context, cb lookout.EventHandler, r *lookout.RepositoryInfo,
	resp *github.Response, prs []*github.PullRequest) error {

	if len(prs) == 0 {
		return nil
	}

	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{"repo": r.Link()})

	for _, e := range prs {
		ctx, _ := ctxlog.WithLogFields(ctx, log.Fields{
			"pr-id":     e.GetID(),
			"pr-number": e.GetNumber(),
		})
		event := castPullRequest(ctx, r, e)

		if err := cb(ctx, event); err != nil {
			return err
		}
	}

	logger.Debugf("request to %s cached", resp.Request.URL)

	client, err := w.getClient(r.Username, r.Name)
	if err != nil {
		return err
	}

	return client.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvents(ctx context.Context, cb lookout.EventHandler, r *lookout.RepositoryInfo,
	resp *github.Response, events []*github.Event) error {

	if len(events) == 0 {
		return nil
	}

	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{"repo": r.Link()})

	for _, e := range events {
		event, err := w.handleEvent(r, e)
		if err != nil {
			logger.Errorf(err, "error handling event")
			continue
		}

		if event == nil {
			continue
		}

		if err := cb(ctx, event); err != nil {
			return err
		}
	}

	logger.Debugf("request to %s cached", resp.Request.URL)

	client, err := w.getClient(r.Username, r.Name)
	if err != nil {
		return err
	}

	return client.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvent(r *lookout.RepositoryInfo, e *github.Event) (lookout.Event, error) {
	return castEvent(r, e)
}

func (w *Watcher) doPRListRequest(ctx context.Context, username, repository string) (
	*github.Response, []*github.PullRequest, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	client, err := w.getClient(username, repository)
	if err != nil {
		return nil, nil, err
	}
	prs, resp, err := client.PullRequests.List(ctx, username, repository, &github.PullRequestListOptions{})
	if err != nil {
		return resp, nil, ErrGitHubAPI.Wrap(err)
	}

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	return resp, prs, err
}

func (w *Watcher) doEventRequest(ctx context.Context, username, repository string) (
	*github.Response, []*github.Event, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	client, err := w.getClient(username, repository)
	if err != nil {
		return nil, nil, err
	}

	events, resp, err := client.Activity.ListRepositoryEvents(
		ctx, username, repository, &github.ListOptions{},
	)

	if err != nil {
		return resp, nil, ErrGitHubAPI.Wrap(err)
	}

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	return resp, events, err
}

func (w *Watcher) getClient(username, repository string) (*Client, error) {
	client, ok := w.pool.Client(username, repository)
	if !ok {
		return nil, fmt.Errorf("client for %s/%s doesn't exist", username, repository)
	}
	return client, nil
}

func (w *Watcher) newInterval(rate github.Rate, minInterval time.Duration) time.Duration {
	interval := minInterval
	remaining := rate.Remaining / 2 // we call 2 endpoints for each repo
	if remaining > 0 {
		secs := int(rate.Reset.Sub(time.Now()).Seconds() / float64(remaining))
		interval = time.Duration(secs) * time.Second
	} else if !rate.Reset.IsZero() {
		interval = rate.Reset.Sub(time.Now())
	}

	if interval < minInterval {
		interval = minInterval
	}

	return interval
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
