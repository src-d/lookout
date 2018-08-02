package github

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cache"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-log.v1"
)

const Provider = "github"

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
	r []*lookout.RepositoryInfo
	o *lookout.WatchOptions
	c *github.Client

	cache *cache.ValidableCache

	// delay for pull requests
	prPollInterval time.Duration
	// delay is time in seconds to wait between requests for events
	eventsPollInterval time.Duration
}

// NewWatcher returns a new
func NewWatcher(transport http.RoundTripper, o *lookout.WatchOptions) (*Watcher, error) {
	repos := make([]*lookout.RepositoryInfo, len(o.URLs))

	for i, url := range o.URLs {
		repo, err := vcsurl.Parse(url)
		if err != nil {
			return nil, err
		}

		repos[i] = repo
	}

	cache := cache.NewValidableCache(diskcache.New("/tmp/github"))

	t := httpcache.NewTransport(cache)
	t.MarkCachedResponses = true
	t.Transport = transport

	return &Watcher{
		r: repos,
		o: o,
		c: github.NewClient(&http.Client{
			Transport: t,
		}),

		prPollInterval:     minInterval,
		eventsPollInterval: minInterval,

		cache: cache,
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"urls": w.o.URLs}).Infof("Starting watcher")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	go w.watchPrs(ctx, cb, errCh)
	go w.watchEvents(ctx, cb, errCh)

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

func (w *Watcher) watchPrs(ctx context.Context, cb lookout.EventHandler, errCh chan error) {
	for {
		for _, repo := range w.r {
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
			case <-time.After(w.prPollInterval):
				continue
			}
		}
	}
}

func (w *Watcher) watchEvents(ctx context.Context, cb lookout.EventHandler, errCh chan error) {
	for {
		for _, repo := range w.r {
			resp, events, err := w.doEventRequest(ctx, repo.Username, repo.Name)
			if err != nil && !NoErrNotModified.Is(err) {
				errCh <- err
				return
			}

			if err := w.handleEvents(cb, repo, resp, events); err != nil {
				errCh <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(w.eventsPollInterval):
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
	return w.cache.Validate(resp.Request.URL.String())
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
	return w.cache.Validate(resp.Request.URL.String())
}

func (w *Watcher) handleEvent(r *lookout.RepositoryInfo, e *github.Event) (lookout.Event, error) {
	return castEvent(r, e)
}

func (w *Watcher) doPRListRequest(ctx context.Context, username, repository string) (
	*github.Response, []*github.PullRequest, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	prs, resp, err := w.c.PullRequests.List(ctx, username, repository, &github.PullRequestListOptions{})
	if err != nil {
		return resp, nil, err
	}

	w.newInterval(resp)

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	w.responseLogger(resp).With(log.Fields{"poll-interval": w.prPollInterval}).
		Debugf("Request to pull requests endpoint done with %d prs.", len(prs))

	return resp, prs, err
}

func (w *Watcher) doEventRequest(ctx context.Context, username, repository string) (
	*github.Response, []*github.Event, error,
) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	events, resp, err := w.c.Activity.ListRepositoryEvents(
		ctx, username, repository, &github.ListOptions{},
	)

	if err != nil {
		return resp, nil, err
	}

	interval := w.newInterval(resp)
	// obey poll interval
	secs, _ := strconv.Atoi(resp.Response.Header.Get("X-Poll-Interval"))
	pollLimit := time.Duration(secs) * time.Second
	if pollLimit > interval {
		interval = pollLimit
	}
	w.eventsPollInterval = interval

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	w.responseLogger(resp).With(log.Fields{"poll-interval": w.eventsPollInterval}).
		Debugf("Request to events endpoint done with %d events.", len(events))

	return resp, events, err
}

func (w *Watcher) newInterval(resp *github.Response) time.Duration {
	interval := minInterval
	remaining := resp.Rate.Remaining / 2 // we call 2 endpoints for each repo
	if remaining > 0 {
		secs := int(resp.Rate.Reset.Sub(time.Now()).Seconds() / float64(remaining))
		interval = time.Duration(secs) * time.Second
	} else {
		interval = resp.Rate.Reset.Sub(time.Now())
	}

	if interval < minInterval {
		interval = minInterval
	}

	// update pr interval on any call
	w.prPollInterval = interval
	return interval
}

func (w *Watcher) responseLogger(resp *github.Response) log.Logger {
	return log.With(log.Fields{
		"remaining-requests": resp.Rate.Remaining,
		"reset-at":           resp.Rate.Reset,
	})
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
