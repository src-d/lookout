package queue

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"golang.org/x/sync/semaphore"
	"gopkg.in/src-d/go-queue.v1"
	_ "gopkg.in/src-d/go-queue.v1/amqp"
	_ "gopkg.in/src-d/go-queue.v1/memory"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

// QueueJob is the data sent to the queue
type QueueJob struct {
	EventType lookout.EventType
	// Exported so go-queue marshals it, but should be accessed through Event()
	ReviewEvent *lookout.ReviewEvent
	// Exported so go-queue marshals it, but should be accessed through Event()
	PushEvent *lookout.PushEvent
}

// NewQueueJob creates a new QueueJob from the given Event
func NewQueueJob(e lookout.Event) (*QueueJob, error) {
	qJob := QueueJob{EventType: e.Type()}

	switch ev := e.(type) {
	case *lookout.ReviewEvent:
		qJob.ReviewEvent = ev
	case *lookout.PushEvent:
		qJob.PushEvent = ev
	default:
		return nil, fmt.Errorf("unsupported event type %s", reflect.TypeOf(e).String())
	}

	return &qJob, nil
}

// Event returns the lookout Event stored in this queue job
func (j QueueJob) Event() (lookout.Event, error) {
	switch j.EventType {
	case pb.PushEventType:
		return j.PushEvent, nil
	case pb.ReviewEventType:
		return j.ReviewEvent, nil
	}

	return nil, fmt.Errorf("queue does not contain a valid lookout event")
}

// EventEnqueuer returns an event handler that pushes events to the queue.
func EventEnqueuer(ctx context.Context, q queue.Queue) lookout.EventHandler {
	return func(ctx context.Context, e lookout.Event) error {
		qJob, err := NewQueueJob(e)
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

	sem := semaphore.NewWeighted(int64(concurrent))

	for {
		if err := sem.Acquire(ctx, 1); err != nil {
			if err == context.Canceled {
				return err
			}

			ctxlog.Get(ctx).Errorf(err, "failed to acquire semaphore")
			continue
		}

		go func() {
			defer sem.Release(1)

			consumedJob, err := iter.Next()
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "queue iterator failed")
				return
			}

			if consumedJob == nil {
				ctxlog.Get(ctx).Warningf("consumedJob is not expected to be nil")
				time.Sleep(1 * time.Second)
				return
			}

			var qJob QueueJob
			if err = consumedJob.Decode(&qJob); err != nil {
				ctxlog.Get(ctx).Errorf(err, "job decode failed")
				consumedJob.Reject(false)
				return
			}

			event, err := qJob.Event()
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
				consumedJob.Reject(false)
				return
			}

			err = eventHandler(ctx, event)
			if err != nil {
				ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
				consumedJob.Reject(true)
				return
			}

			consumedJob.Ack()
		}()
	}
}
