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
	r *vcsurl.RepoInfo
	o *lookout.WatchOptions
	c *github.Client

	cache *cache.ValidableCache

	// delay is time in seconds to wait between requests
	poolInterval time.Duration
}

// NewWatcher returns a new
func NewWatcher(o *lookout.WatchOptions) (*Watcher, error) {
	r, err := vcsurl.Parse(o.URL)
	if err != nil {
		return nil, err
	}

	cache := cache.NewValidableCache(diskcache.New("/tmp/github"))

	t := httpcache.NewTransport(cache)
	t.MarkCachedResponses = true

	return &Watcher{
		r: r,
		o: o,
		c: github.NewClient(&http.Client{
			Transport: t,
		}),

		cache: cache,
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"url": w.o.URL}).Infof("Starting watcher")

	for {
		resp, events, err := w.doEventRequest(ctx, w.r.Username, w.r.Name)
		if err != nil && !NoErrNotModified.Is(err) {
			return err
		}

		if err := w.handleEvents(cb, resp, events); err != nil {
			if lookout.NoErrStopWatcher.Is(err) {
				return nil
			}

			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.poolInterval):
			continue
		}
	}
}

func (w *Watcher) handleEvents(cb lookout.EventHandler, resp *github.Response, events []*github.Event) error {
	if len(events) == 0 {
		return nil
	}

	for _, e := range events {
		event, err := w.handleEvent(e)
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

func (w *Watcher) handleEvent(e *github.Event) (lookout.Event, error) {
	return castEvent(w.r, e)
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
	w.poolInterval = time.Duration(secs) * time.Second

	if isStatusNotModified(resp.Response) {
		return nil, nil, NoErrNotModified.New()
	}

	log.With(log.Fields{
		"remaining-requests": resp.Rate.Remaining,
		"reset-at":           resp.Rate.Reset,
		"pool-interval":      w.poolInterval,
		"events":             len(events),
	}).Debugf("Requested to events endpoint done.")

	return resp, events, err
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
