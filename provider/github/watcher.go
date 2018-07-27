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

	// delay is time in seconds to wait between requests
	pollInterval time.Duration
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

		cache: cache,
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"urls": w.o.URLs}).Infof("Starting watcher")

	repoIndex := 0

	for {
		repo := w.r[repoIndex]
		repoIndex = (repoIndex + 1) % len(w.r)

		resp, events, err := w.doEventRequest(ctx, repo.Username, repo.Name)
		if err != nil && !NoErrNotModified.Is(err) {
			return err
		}

		if err := w.handleEvents(cb, repo, resp, events); err != nil {
			if lookout.NoErrStopWatcher.Is(err) {
				return nil
			}

			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.pollInterval):
			continue
		}
	}
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

	secs, _ := strconv.Atoi(resp.Response.Header.Get("X-Poll-Interval"))
	w.pollInterval = time.Duration(secs) * time.Second

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	log.With(log.Fields{
		"remaining-requests": resp.Rate.Remaining,
		"reset-at":           resp.Rate.Reset,
		"poll-interval":      w.pollInterval,
		"events":             len(events),
	}).Debugf("Request to events endpoint done.")

	return resp, events, err
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
