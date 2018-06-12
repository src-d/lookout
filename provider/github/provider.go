package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/src-d/lookout/provider"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-errors.v0"
	"gopkg.in/src-d/go-log.v1"
)

type Watcher struct {
	c *github.Client

	// delay is time in seconds to wait between requests
	poolInterval time.Duration
}

func NewWatcher() *Watcher {
	cache := httpcache.NewTransport(diskcache.New("/tmp/github"))
	cache.MarkCachedResponses = true

	return &Watcher{c: github.NewClient(&http.Client{
		Transport: cache,
	})}
}

func (w *Watcher) Watch(ctx context.Context, opts provider.WatchOptions) error {
	r, err := vcsurl.Parse(opts.URL)
	if err != nil {
		return err
	}

	for {
		events, err := w.doEventRequest(ctx, r.Username, r.Name)
		if err != nil && !NoErrNotModified.Is(err) {
			return err
		}

		w.handleEvents(r, events)

		fmt.Println(len(events), err, "sleep", w.poolInterval)
		time.Sleep(w.poolInterval)
	}

	return nil
}

func (w *Watcher) handleEvents(r *vcsurl.RepoInfo, events []*github.Event) {
	for _, e := range events {
		if err := w.handleEvent(r, e); err != nil {
			fmt.Println(err)
		}
	}
}
func (w *Watcher) handleEvent(r *vcsurl.RepoInfo, e *github.Event) error {
	event, err := castEvent(r, e)
	if err != nil {
		return err
	}

	if event == nil {
		return nil
	}

	fmt.Println(event)

	return nil
}

var (
	NoErrNotModified       = errors.NewKind("Not modified")
	ErrParsingEventPayload = errors.NewKind("Parse error in event")
)

func (w *Watcher) doEventRequest(ctx context.Context, username, repository string) ([]*github.Event, error) {
	events, resp, err := w.c.Activity.ListRepositoryEvents(
		ctx, username, repository, &github.ListOptions{},
	)

	log.With(log.Fields{
		"remaining-requests": resp.Rate.Remaining,
		"reset-at":           resp.Rate.Reset,
	}).Debugf("Requested more events")

	//fmt.Println(resp.Response)

	secs, _ := strconv.Atoi(resp.Response.Header.Get("X-Poll-Interval"))
	w.poolInterval = time.Duration(secs) * time.Second

	if isStatusNotModified(resp.Response) {
		return nil, NoErrNotModified.New()
	}

	return events, err
}

func isStatusNotModified(resp *http.Response) bool {
	return resp.Header.Get("X-From-Cache") == "1"
}
