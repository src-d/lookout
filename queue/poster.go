package queue

import (
	"context"
	"fmt"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
	log "gopkg.in/src-d/go-log.v1"
	queue "gopkg.in/src-d/go-queue.v1"
)

// postCommentsJob is the data sent to the poster queue
type postCommentsJob struct {
	*Event
	Comments []lookout.AnalyzerComments
	Safe     bool
	// LogFields contains ctxlog logger fields to keep log continuity
	LogFields log.Fields
}

// Poster implements lookout.Poster interface by using a queue
type Poster struct {
	underlying lookout.Poster
	q          queue.Queue
}

// NewPoster wraps any lookout.Poster into queue
func NewPoster(p lookout.Poster, q queue.Queue) *Poster {
	return &Poster{underlying: p, q: q}
}

// Post implements Poster interface by sending comments into the queue
func (p *Poster) Post(ctx context.Context, e lookout.Event, cs []lookout.AnalyzerComments, safe bool) error {
	j, err := queue.NewJob()
	if err != nil {
		return err
	}

	ev, err := NewEvent(e)
	if err != nil {
		return err
	}

	if err := j.Encode(&postCommentsJob{
		Event:     ev,
		Comments:  cs,
		Safe:      safe,
		LogFields: ctxlog.Fields(ctx),
	}); err != nil {
		return err
	}

	return p.q.Publish(j)
}

// Status implements Poster interface by calling underlying poster
func (p *Poster) Status(ctx context.Context, e lookout.Event, s lookout.AnalysisStatus) error {
	return p.underlying.Status(ctx, e, s)
}

// Consume starts infinite loop that posts comments from the queue
func (p *Poster) Consume(ctx context.Context, concurrent int) error {
	if concurrent < 1 {
		return fmt.Errorf("wrong value %v for concurrent argument", concurrent)
	}

	iter, err := p.q.Consume(concurrent)
	if err != nil {
		return fmt.Errorf("poster queue consume failed: %s", err.Error())
	}
	defer func() {
		if err := iter.Close(); err != nil {
			ctxlog.Get(ctx).Errorf(err, "poster queue iterator close failed")
		}
	}()

	for {
		j, err := iter.Next()
		if err != nil {
			return fmt.Errorf("poster queue iterator failed: %s", err.Error())
		}
		if err := p.process(ctx, j); err != nil {
			return err
		}
	}
}

func (p *Poster) process(ctx context.Context, j *queue.Job) error {
	var payload postCommentsJob
	if err := j.Decode(&payload); err != nil {
		ctxlog.Get(ctx).Errorf(err, "job decode failed")
		return j.Reject(false)
	}

	e, err := payload.ToInterface()
	if err != nil {
		ctxlog.Get(ctx).Errorf(err, "error handling the queue job")
		return j.Reject(false)
	}

	jobCtx, _ := ctxlog.WithLogFields(ctx, payload.LogFields)
	err = p.underlying.Post(jobCtx, e, payload.Comments, payload.Safe)
	if err != nil {
		ctxlog.Get(jobCtx).Errorf(err, "comment posting has failed")
		return j.Reject(false)
	}

	return j.Ack()
}
