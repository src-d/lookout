package github

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-errors.v0"
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

	// delay is time in seconds to wait between requests
	poolInterval time.Duration
}

// NewWatcher returns a new
func NewWatcher(o *lookout.WatchOptions) (*Watcher, error) {
	cache := httpcache.NewTransport(diskcache.New("/tmp/github"))
	cache.MarkCachedResponses = true

	r, err := vcsurl.Parse(o.URL)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		r: r,
		o: o,
		c: github.NewClient(&http.Client{
			Transport: cache,
		}),
	}, nil
}

// Watch start to make request to the GitHub API and return the new events.
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"url": w.o.URL}).Infof("Starting watcher")

	for {
		events, err := w.doEventRequest(ctx, w.r.Username, w.r.Name)
		if err != nil && !NoErrNotModified.Is(err) {
			return err
		}

		if err := w.handleEvents(cb, events); err != nil {
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

func (w *Watcher) handleEvents(cb lookout.EventHandler, events []*github.Event) error {
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

	return nil
}

func (w *Watcher) handleEvent(e *github.Event) (lookout.Event, error) {
	return castEvent(w.r, e)
}

func (w *Watcher) doEventRequest(ctx context.Context, username, repository string) ([]*github.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	events, resp, err := w.c.Activity.ListRepositoryEvents(
		ctx, username, repository, &github.ListOptions{},
	)

	secs, _ := strconv.Atoi(resp.Response.Header.Get("X-Poll-Interval"))
	w.poolInterval = time.Duration(secs) * time.Second

	if isStatusNotModified(resp.Response) {
		return nil, NoErrNotModified.New()
	}

	log.With(log.Fields{
		"remaining-requests": resp.Rate.Remaining,
		"reset-at":           resp.Rate.Reset,
		"pool-interval":      w.poolInterval,
		"events":             len(events),
	}).Debugf("Requested to events endpoint done.")

	return events, err
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
