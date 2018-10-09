package queue

import (
	"io"
	"time"

	"gopkg.in/src-d/go-errors.v1"
)

// Priority represents a priority level.
type Priority uint8

const (
	// PriorityUrgent represents an urgent priority level.
	PriorityUrgent Priority = 8
	// PriorityNormal represents a normal priority level.
	PriorityNormal Priority = 4
	// PriorityLow represents a low priority level.
	PriorityLow Priority = 0
)

var (
	// ErrAlreadyClosed is the error returned when trying to close an already closed
	// connection.
	ErrAlreadyClosed = errors.NewKind("already closed")
	// ErrEmptyJob is the error returned when an empty job is published.
	ErrEmptyJob = errors.NewKind("invalid empty job")
	// ErrTxNotSupported is the error returned when the transaction receives a
	// callback does not know how to handle.
	ErrTxNotSupported = errors.NewKind("transactions not supported")
)

// Broker represents a message broker.
type Broker interface {
	// Queue returns a Queue from the Broker with the given name.
	Queue(string) (Queue, error)
	// Close closes the connection of the Broker.
	Close() error
}

// TxCallback is a function to be called in a transaction.
type TxCallback func(q Queue) error

// RepublishConditionFunc is a function used to filter jobs to republish.
type RepublishConditionFunc func(job *Job) bool

// RepublishConditions alias of a list RepublishConditionFunc
type RepublishConditions []RepublishConditionFunc

// Comply checks if the Job matches any of the defined conditions.
func (c RepublishConditions) Comply(job *Job) bool {
	if len(c) == 0 {
		return true
	}

	for _, condition := range c {
		if condition(job) {
			return true
		}
	}

	return false
}

// Queue represents a message queue.
type Queue interface {
	// Publish publishes the given Job to the queue.
	Publish(*Job) error
	// PublishDelayed publishes the given Job to the queue with a given delay.
	PublishDelayed(*Job, time.Duration) error
	// Transaction executes the passed TxCallback inside a transaction.
	Transaction(TxCallback) error
	// Consume returns a JobIter for the queue.  It receives the maximum
	// number of unacknowledged jobs the iterator will allow at any given
	// time (see the Acknowledger interface).
	Consume(advertisedWindow int) (JobIter, error)
	// RepublishBuried republishes to the main queue those jobs complying
	// one of the conditions, leaving the rest of them in the buried queue.
	RepublishBuried(conditions ...RepublishConditionFunc) error
}

// JobIter represents an iterator over a set of Jobs.
type JobIter interface {
	// Next returns the next Job in the iterator. It should block until
	// a new job becomes available or while too many undelivered jobs have
	// been already returned (see the argument to Queue.Consume). Returns
	// ErrAlreadyClosed if the iterator is closed.
	Next() (*Job, error)
	io.Closer
}
