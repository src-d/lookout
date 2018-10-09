package queue

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/pkg/errors"
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

	return nil, errors.New("queue does not contain a valid lookout event")
}

// queueWindow defines the maximum number of unacknowledged jobs that the
// queue iterator will allow to retrieve. A number of 1 will mean that each
// job is processed sequentially
// TODO(carlosms): because the eventHandler is called synchronously, a value
// greater than 1 will not have any effect for now. We will not call iter.Next()
// until the previous job is acknowledged or rejected
const queueWindow = 1

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

// RunEventDequeuer starts a loop that takes jobs from the queue as they become
// available, and calls the given event handler. The handler call is synchronous,
// the next job will not be retrieved until the callback handler is finished with
func RunEventDequeuer(ctx context.Context, q queue.Queue, eventHandler lookout.EventHandler) error {
	iter, err := q.Consume(queueWindow)
	if err != nil {
		return errors.Wrap(err, "queue consume failed")
	}

	defer func() {
		if err := iter.Close(); err != nil {
			ctxlog.Get(ctx).Errorf(err, "queue iterator close failed")
		}
	}()

	for {
		consumedJob, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "queue iterator failed")
		}

		if consumedJob == nil {
			ctxlog.Get(ctx).Warningf("consumedJob should not be nil, bug in memory queue?")
			time.Sleep(1 * time.Second)
			continue
		}

		var qJob QueueJob
		if err = consumedJob.Decode(&qJob); err != nil {
			ctxlog.Get(ctx).Errorf(err, "job decode failed")
			consumedJob.Reject(false)
			continue
		}

		event, err := qJob.Event()
		if err != nil {
			ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
			consumedJob.Reject(false)
			continue
		}

		err = eventHandler(ctx, event)
		if err != nil {
			ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
			consumedJob.Reject(true)
			continue
		}

		consumedJob.Ack()
	}
}
