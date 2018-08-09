package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-log.v1"
)

const Provider = "github"

// ProviderConfig represents the yml config
type ProviderConfig struct {
	CommentFooter string `yaml:"comment_footer"`
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
	o    *lookout.WatchOptions
}

// NewWatcher returns a new
func NewWatcher(pool *ClientPool, o *lookout.WatchOptions) (*Watcher, error) {
	return &Watcher{
		pool: pool,
		o:    o,
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"urls": w.o.URLs}).Infof("Starting watcher")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	for client, repos := range w.pool.Clients() {
		go w.watchPrs(ctx, client, repos, cb, errCh)
		go w.watchEvents(ctx, client, repos, cb, errCh)
	}

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

func (w *Watcher) watchPrs(ctx context.Context, c *Client, repos []*lookout.RepositoryInfo, cb lookout.EventHandler, errCh chan error) {
	for {
		for _, repo := range repos {
			resp, prs, err := w.doPRListRequest(ctx, repo.Username, repo.Name)
			if err != nil && !NoErrNotModified.Is(err) {
				errCh <- err
				return
			}

			if err := w.handlePrs(cb, repo, resp, prs); err != nil {
				errCh <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(w.newInterval(c.Rate(coreCategory))):
				continue
			}
		}
	}
}

func (w *Watcher) watchEvents(ctx context.Context, c *Client, repos []*lookout.RepositoryInfo, cb lookout.EventHandler, errCh chan error) {
	for {
		for _, repo := range repos {
			resp, events, err := w.doEventRequest(ctx, repo.Username, repo.Name)
			if err != nil && !NoErrNotModified.Is(err) {
				errCh <- err
				return
			}

			if err := w.handleEvents(cb, repo, resp, events); err != nil {
				errCh <- err
				return
			}

			interval := w.newInterval(c.Rate(coreCategory))
			pollInterval := c.PollInterval(eventsCategory)
			if pollInterval > interval {
				interval = pollInterval
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(pollInterval):
				continue
			}
		}
	}
}

func (w *Watcher) handlePrs(cb lookout.EventHandler, r *lookout.RepositoryInfo,
	resp *github.Response, prs []*github.PullRequest) error {

	if len(prs) == 0 {
		return nil
	}

	for _, e := range prs {
		event := castPullRequest(r, e)

		if err := cb(event); err != nil {
			return err
		}
	}

	log.Debugf("request to %s cached", resp.Request.URL)

	client, err := w.getClient(r.Username, r.Name)
	if err != nil {
		return err
	}

	return client.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvents(cb lookout.EventHandler, r *lookout.RepositoryInfo,
	resp *github.Response, events []*github.Event) error {

	if len(events) == 0 {
		return nil
	}

	for _, e := range events {
		event, err := w.handleEvent(r, e)
		if err != nil {
			log.Errorf(err, "error handling event")
			continue
		}

		if event == nil {
			continue
		}

		if err := cb(event); err != nil {
			return err
		}
	}

	log.Debugf("request to %s cached", resp.Request.URL)

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
		return resp, nil, err
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
		return resp, nil, err
	}

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	return resp, events, err
}

func (w *Watcher) getClient(username, repository string) (*Client, error) {
	client, ok := w.pool.Client(username, repository)
	if !ok {
		return nil, fmt.Errorf("client for %s/%s doesn't exists", username, repository)
	}
	return client, nil
}

func (w *Watcher) newInterval(rate github.Rate) time.Duration {
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
