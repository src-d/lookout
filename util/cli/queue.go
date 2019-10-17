package cli

import (
	queue "gopkg.in/src-d/go-queue.v1"
)

// QueueOptions contains common flags for commands using a Queue
type QueueOptions struct {
	EventsQueueName string `long:"queue" env:"LOOKOUT_QUEUE" default:"lookout" description:"events queue name"`
	PosterQueueName string `long:"poster-queue" env:"LOOKOUT_POSTER_QUEUE" default:"poster" description:"poster queue name"`
	Broker          string `long:"broker" env:"LOOKOUT_BROKER" default:"amqp://localhost:5672" description:"broker service URI"`
}

// EventsQueue initializes events queue from the given cli options.
func (c *QueueOptions) EventsQueue() (queue.Queue, error) {
	return c.makeQueue(c.EventsQueueName)
}

// PosterQueue initializes poster queue from the given cli options.
func (c *QueueOptions) PosterQueue() (queue.Queue, error) {
	return c.makeQueue(c.PosterQueueName)
}

func (c *QueueOptions) makeQueue(name string) (queue.Queue, error) {
	b, err := queue.NewBroker(c.Broker)
	if err != nil {
		return nil, err
	}

	return b.Queue(name)
}
