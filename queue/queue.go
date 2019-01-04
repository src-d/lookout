package queue

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	log "gopkg.in/src-d/go-log.v1"
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
	// LogFields contains ctxlog logger fields to keep log continuity
	LogFields log.Fields
}

// NewQueueJob creates a new QueueJob from the given Event
func NewQueueJob(ctx context.Context, e lookout.Event) (*QueueJob, error) {
	qJob := QueueJob{
		EventType: e.Type(),
		LogFields: ctxlog.Fields(ctx),
	}

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
		qJob, err := NewQueueJob(ctx, e)
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
