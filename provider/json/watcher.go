package json

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-log.v1"
)

// Provider is the name
const Provider = "json"

// Watcher watches for new json events in the console
type Watcher struct {
	o       *lookout.WatchOptions
	scanner *bufio.Scanner
}

// NewWatcher returns a new json console watcher
func NewWatcher(reader io.Reader, o *lookout.WatchOptions) (*Watcher, error) {
	return &Watcher{
		o:       o,
		scanner: bufio.NewScanner(reader),
	}, nil
}

// Watch reads json from stdin and calls cb for each new event
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

type eventType struct {
	Event string `json:"event"`
}

func (w *Watcher) handleInput(cb lookout.EventHandler, line string) error {
	if line == "" {
		return nil
	}

	var eventType eventType

	if err := json.Unmarshal([]byte(line), &eventType); err != nil {
		log.With(log.Fields{"input": line}).Errorf(err, "could not unmarshal the event")

		return nil
	}

	var event lookout.Event

	switch strings.ToLower(eventType.Event) {
	case "":
		log.With(log.Fields{"input": line}).Errorf(nil, `field "event" is mandatory`)
		return nil
	case "review":
		var reviewEvent *lookout.ReviewEvent
		if err := json.Unmarshal([]byte(line), &reviewEvent); err != nil {
			log.With(log.Fields{"input": line}).Errorf(err, "could not unmarshal the ReviewEvent")
			return nil
		}

		event = reviewEvent
	case "push":
		var pushEvent *lookout.PushEvent
		if err := json.Unmarshal([]byte(line), &pushEvent); err != nil {
			log.With(log.Fields{"input": line}).Errorf(err, "could not unmarshal the PushEvent")
			return nil
		}

		event = pushEvent
	default:
		log.Errorf(nil, "event %q not supported", eventType.Event)
		return nil
	}

	return cb(event)
}
