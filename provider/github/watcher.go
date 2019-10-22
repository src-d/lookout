package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/google/go-github/v24/github"
	errors "gopkg.in/src-d/go-errors.v1"
	log "gopkg.in/src-d/go-log.v1"
)

const Provider = "github"

// ProviderConfig represents the yml config
type ProviderConfig struct {
	CommentFooter            string `yaml:"comment_footer"`
	PrivateKey               string `yaml:"private_key"`
	AppID                    int    `yaml:"app_id"`
	InstallationSyncInterval string `yaml:"installation_sync_interval"`
	WatchMinInterval         string `yaml:"watch_min_interval"`
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
	stopFuncs               map[*Client]func()
	lastErrPR, lastErrEvent map[*repositoryInfo]*errThrottlerState
}

// NewWatcher returns a new
func NewWatcher(pool *ClientPool) (*Watcher, error) {
	return &Watcher{
		pool:         pool,
		stopFuncs:    make(map[*Client]func()),
		lastErrPR:    make(map[*repositoryInfo]*errThrottlerState),
		lastErrEvent: make(map[*repositoryInfo]*errThrottlerState),
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
	ch := make(chan ClientPoolEvent)
	w.pool.Subscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case change := <-ch:
			ctxlog.Get(ctx).
				With(log.Fields{"type": change.Type}).
				Debugf("New event from the client pool")

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
	repoNames := make([]string, 0)
	for _, repo := range w.pool.ReposByClient(client) {
		repoNames = append(repoNames, repo.FullName)
	}
	ctxlog.Get(ctx).With(log.Fields{
		"repositories": repoNames,
	}).Infof("start github client loop")

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
	*repositoryInfo,
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
	repo *repositoryInfo,
	cb lookout.EventHandler,
) (time.Duration, error) {
	resp, prs, err := w.doPRListRequest(ctx, c, repo.Owner, repo.Name)
	if ErrGitHubAPI.Is(err) {
		if w.lastErrPR[repo] == nil {
			w.lastErrPR[repo] = &errThrottlerState{}
		}

		// go-errors %+v prints the stack trace. Doing this we create a plain
		// error with no stack trace for the log
		err := fmt.Errorf("%s", err)

		newErrThrottlerLogger(ctxlog.Get(ctx), w.lastErrPR[repo]).With(log.Fields{
			"repository": repo.FullName,
		}).Errorf(err, "request for PR list failed")

		return c.watchMinInterval, nil
	}

	if NoErrNotModified.Is(err) {
		return c.watchMinInterval, nil
	}

	if err != nil {
		return c.watchMinInterval, err
	}

	err = w.handlePrs(ctx, c, cb, repo, resp, prs)
	return c.watchMinInterval, err
}

func (w *Watcher) processRepoEvents(
	ctx context.Context,
	c *Client,
	repo *repositoryInfo,
	cb lookout.EventHandler,
) (time.Duration, error) {
	resp, events, err := w.doEventRequest(ctx, c, repo.Owner, repo.Name)
	if ErrGitHubAPI.Is(err) {
		if w.lastErrEvent[repo] == nil {
			w.lastErrEvent[repo] = &errThrottlerState{}
		}

		// go-errors %+v prints the stack trace. Doing this we create a plain
		// error with no stack trace for the log
		err := fmt.Errorf("%s", err)

		newErrThrottlerLogger(ctxlog.Get(ctx), w.lastErrEvent[repo]).With(log.Fields{
			"repository": repo.FullName,
		}).Errorf(err, "request for events list failed")

		return c.PollInterval(eventsCategory), nil
	}

	if NoErrNotModified.Is(err) {
		return c.PollInterval(eventsCategory), nil
	}

	if err != nil {
		return c.PollInterval(eventsCategory), err
	}

	err = w.handleEvents(ctx, c, cb, repo, resp, events)
	return c.PollInterval(eventsCategory), err
}

func (w *Watcher) handlePrs(ctx context.Context,
	client *Client,
	cb lookout.EventHandler,
	r *repositoryInfo,
	resp *github.Response,
	prs []*github.PullRequest,
) error {

	if len(prs) == 0 {
		return nil
	}

	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{"repository": r.CloneURL})

	for _, e := range prs {
		// github doesn't run any checks on draft PRs
		// emulate this behaviour by skipping prs as long as they are draft
		if e.GetDraft() {
			continue
		}

		ctx, _ := ctxlog.WithLogFields(ctx, log.Fields{
			"github.pr": e.GetNumber(),
		})
		event := castPullRequest(ctx, r, e)

		if err := cb(ctx, event); err != nil {
			return err
		}
	}

	logger.Debugf("request to %s cached", resp.Request.URL)

	return client.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvents(
	ctx context.Context,
	client *Client,
	cb lookout.EventHandler,
	r *repositoryInfo,
	resp *github.Response,
	events []*github.Event,
) error {

	if len(events) == 0 {
		return nil
	}

	ctx, logger := ctxlog.WithLogFields(ctx, log.Fields{"repository": r.CloneURL})

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

	return client.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvent(r *repositoryInfo, e *github.Event) (lookout.Event, error) {
	return castEvent(r, e)
}

func (w *Watcher) doPRListRequest(ctx context.Context, client *Client, username, repository string) (
	*github.Response, []*github.PullRequest, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	prs, resp, err := client.PullRequests.List(ctx, username, repository, &github.PullRequestListOptions{})
	if err != nil {
		return resp, nil, ErrGitHubAPI.Wrap(err, "pull requests could not be listed")
	}

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	return resp, prs, err
}

func (w *Watcher) doEventRequest(ctx context.Context, client *Client, username, repository string) (
	*github.Response, []*github.Event, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	events, resp, err := client.Activity.ListRepositoryEvents(
		ctx, username, repository, &github.ListOptions{},
	)

	if err != nil {
		return resp, nil, ErrGitHubAPI.Wrap(err, "repository events could not be listed")
	}

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	return resp, events, err
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
