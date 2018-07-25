package console

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-log.v1"
)

// Provider is the name
const Provider = "console"

// WatchOptions options to use in the Watcher constructor.
type WatchOptions struct {
	Reader io.Reader
}

// Watcher watches for new events in the console
type Watcher struct {
	o       *WatchOptions
	scanner *bufio.Scanner
}

// NewWatcher returns a new console watcher
func NewWatcher(o *WatchOptions) (*Watcher, error) {
	return &Watcher{
		o:       o,
		scanner: bufio.NewScanner(o.Reader),
	}, nil
}

// Watch reads from stdin and calls cb for each new event
func (w *Watcher) Watch(ctx context.Context, cb lookout.EventHandler) error {
	log.With(log.Fields{"provider": Provider}).Infof("Starting watcher")

	lines := make(chan string, 1)
	go func() {
		for w.scanner.Scan() {
			lines <- w.scanner.Text()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line := <-lines:
			if err := w.handleInput(cb, line); err != nil {
				if lookout.NoErrStopWatcher.Is(err) {
					return nil
				}

				return err
			}
		}
	}
}

func (w *Watcher) handleInput(cb lookout.EventHandler, line string) error {
	if line == "" {
		return nil
	}

	var cmd, gitURL, fromBranch, fromHash, toBranch, toHash string
	_, err := fmt.Sscanf(line,
		"%s %s %s %s %s %s",
		&cmd, &gitURL, &fromBranch, &fromHash, &toBranch, &toHash)

	if err != nil {
		log.Errorf(err, "could not process input line %q", line)
		return nil
	}

	var event lookout.Event

	commitRev := lookout.CommitRevision{
		Base: lookout.ReferencePointer{
			InternalRepositoryURL: gitURL,
			ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", fromBranch)),
			Hash:                  fromHash,
		},
		Head: lookout.ReferencePointer{
			InternalRepositoryURL: gitURL,
			ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", toBranch)),
			Hash:                  toHash,
		},
	}

	switch strings.ToLower(cmd) {
	case "review":
		event = &lookout.ReviewEvent{
			Provider:    Provider,
			InternalID:  "1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			IsMergeable: true,
			//Source
			//Merge
			//Configuration
			CommitRevision: commitRev,
		}

	case "push":
		event = &lookout.PushEvent{
			Provider:   Provider,
			InternalID: "1",
			CreatedAt:  time.Now(),
			//Commits
			//DistinctCommits
			//Configuration
			CommitRevision: commitRev,
		}
	default:
		log.Errorf(nil, "event %q not supported", cmd)
		return nil
	}

	return cb(event)
}
