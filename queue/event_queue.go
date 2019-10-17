package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	log "gopkg.in/src-d/go-log.v1"
	queue "gopkg.in/src-d/go-queue.v1"
)

// eventJob is the data sent to the queue
type eventJob struct {
	*Event
	// LogFields contains ctxlog logger fields to keep log continuity
	LogFields log.Fields
}

// newQueueJob creates a new QueueJob from the given Event
func newEventJob(ctx context.Context, e lookout.Event) (*eventJob, error) {
	ev, err := NewEvent(e)
	if err != nil {
		return nil, err
	}

	return &eventJob{
		Event:     ev,
		LogFields: ctxlog.Fields(ctx),
	}, nil
}

// EventEnqueuer returns an event handler that pushes events to the queue.
func EventEnqueuer(ctx context.Context, q queue.Queue) lookout.EventHandler {
	return func(ctx context.Context, e lookout.Event) error {
		qJob, err := newEventJob(ctx, e)
		if err != nil {
			ctxlog.Get(ctx).Errorf(err, "queue job creation failure")
			return err
		}

		j, _ := queue.NewJob()
		if err := j.Encode(qJob); err != nil {
			ctxlog.Get(ctx).Errorf(err, "encode failed")
			return err
		}

		if err := q.Publish(j); err != nil {
			ctxlog.Get(ctx).Errorf(err, "publish failed")
			return err
		}

		return nil
	}
}

// RunEventDequeuer starts an infinite loop that takes jobs from the queue as
// they become available. Concurrent determines the maximum number of goroutines
// used to call the given event handler.
func RunEventDequeuer(
	ctx context.Context,
	q queue.Queue,
	eventHandler lookout.EventHandler,
	concurrent int,
) error {
	if concurrent < 1 {
		return fmt.Errorf("wrong value %v for concurrent argument", concurrent)
	}

	iter, err := q.Consume(concurrent)
	if err != nil {
		return fmt.Errorf("queue consume failed: %s", err.Error())
	}

	defer func() {
		if err := iter.Close(); err != nil {
			ctxlog.Get(ctx).Errorf(err, "queue iterator close failed")
		}
	}()

	for {
		consumedJob, err := iter.Next()
		if err != nil {
			return fmt.Errorf("queue iterator failed: %s", err.Error())
		}

		if consumedJob == nil {
			ctxlog.Get(ctx).Warningf("consumedJob is not expected to be nil")
			time.Sleep(1 * time.Second)
			continue
		}

		go func(consumedJob *queue.Job) {
			var qJob eventJob
			if err = consumedJob.Decode(&qJob); err != nil {
				ctxlog.Get(ctx).Errorf(err, "job decode failed")
				consumedJob.Reject(false)
				return
			}

			event, err := qJob.ToInterface()
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
				consumedJob.Reject(false)
				return
			}

			jobCtx, _ := ctxlog.WithLogFields(ctx, qJob.LogFields)
			err = eventHandler(jobCtx, event)
			if err != nil {
				ctxlog.Get(jobCtx).Errorf(err, "error handling the queue job")
				consumedJob.Reject(true)
				return
			}

			consumedJob.Ack()
		}(consumedJob)
	}
}
