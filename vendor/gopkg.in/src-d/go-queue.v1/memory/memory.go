package memory

import (
	"io"
	"sync"
	"time"

	"gopkg.in/src-d/go-queue.v1"
)

func init() {
	queue.Register("memory", func(uri string) (queue.Broker, error) {
		return New(), nil
	})
}

// Broker is a in-memory implementation of Broker.
type Broker struct {
	queues map[string]queue.Queue
}

// New creates a new Broker for an in-memory queue.
func New() queue.Broker {
	return &Broker{make(map[string]queue.Queue)}
}

// Queue returns the queue with the given name.
func (b *Broker) Queue(name string) (queue.Queue, error) {
	if _, ok := b.queues[name]; !ok {
		b.queues[name] = &Queue{jobs: make([]*queue.Job, 0, 10)}
	}

	return b.queues[name], nil
}

// Close closes the connection in the Broker.
func (b *Broker) Close() error {
	return nil
}

// Queue implements a queue.Queue interface.
type Queue struct {
	jobs       []*queue.Job
	buriedJobs []*queue.Job
	sync.RWMutex
	idx                int
	publishImmediately bool
}

// Publish publishes a Job to the queue.
func (q *Queue) Publish(j *queue.Job) error {
	if j == nil || j.Size() == 0 {
		return queue.ErrEmptyJob.New()
	}

	q.Lock()
	defer q.Unlock()
	q.jobs = append(q.jobs, j)
	return nil
}

// PublishDelayed publishes a Job to the queue with a given delay.
func (q *Queue) PublishDelayed(j *queue.Job, delay time.Duration) error {
	if j == nil || j.Size() == 0 {
		return queue.ErrEmptyJob.New()
	}

	if q.publishImmediately {
		return q.Publish(j)
	}
	go func() {
		time.Sleep(delay)
		q.Publish(j)
	}()
	return nil
}

// RepublishBuried implements the Queue interface.
func (q *Queue) RepublishBuried(conditions ...queue.RepublishConditionFunc) error {
	for _, job := range q.buriedJobs {
		if queue.RepublishConditions(conditions).Comply(job) {
			job.ErrorType = ""
			if err := q.Publish(job); err != nil {
				return err
			}
		}
	}
	return nil
}

// Transaction calls the given callback inside a transaction.
func (q *Queue) Transaction(txcb queue.TxCallback) error {
	txQ := &Queue{jobs: make([]*queue.Job, 0, 10), publishImmediately: true}
	if err := txcb(txQ); err != nil {
		return err
	}

	q.jobs = append(q.jobs, txQ.jobs...)
	return nil
}

// Consume implements Queue.  MemoryQueues have infinite advertised window.
func (q *Queue) Consume(_ int) (queue.JobIter, error) {
	return &JobIter{q: q, RWMutex: &q.RWMutex}, nil
}

// JobIter implements a queue.JobIter interface.
type JobIter struct {
	q      *Queue
	closed bool
	*sync.RWMutex
}

// Acknowledger implements a queue.Acknowledger interface.
type Acknowledger struct {
	q *Queue
	j *queue.Job
}

// Ack is called when the Job has finished.
func (*Acknowledger) Ack() error {
	return nil
}

// Reject is called when the Job has errored. The argument indicates whether the Job
// should be put back in queue or not.  If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *Acknowledger) Reject(requeue bool) error {
	if !requeue {
		// Send to the buried queue for later republishing
		a.q.buriedJobs = append(a.q.buriedJobs, a.j)
		return nil
	}

	return a.q.Publish(a.j)
}

func (i *JobIter) isClosed() bool {
	i.RLock()
	defer i.RUnlock()
	return i.closed
}

// Next returns the next job in the iter.
func (i *JobIter) Next() (*queue.Job, error) {
	for {
		if i.isClosed() {
			return nil, queue.ErrAlreadyClosed.New()
		}

		j, err := i.next()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		return j, nil
	}
}

func (i *JobIter) next() (*queue.Job, error) {
	i.Lock()
	defer i.Unlock()
	if len(i.q.jobs) <= i.q.idx {
		return nil, io.EOF
	}

	j := i.q.jobs[i.q.idx]
	j.Acknowledger = &Acknowledger{j: j, q: i.q}
	i.q.idx++

	return j, nil
}

// Close closes the iter.
func (i *JobIter) Close() error {
	i.Lock()
	defer i.Unlock()
	i.closed = true
	return nil
}
