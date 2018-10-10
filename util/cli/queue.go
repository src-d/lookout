package cli

import (
	"gopkg.in/src-d/go-queue.v1"
)

// QueueOptions contains common flags for commands using a Queue
type QueueOptions struct {
	Queue  string `long:"queue" env:"LOOKOUT_QUEUE" default:"lookout" description:"queue name"`
	Broker string `long:"broker" env:"LOOKOUT_BROKER" default:"amqp://localhost:5672" description:"broker service URI"`

	Q queue.Queue
}

// InitQueue initializes the queue from the given cli options.
func (c *QueueOptions) InitQueue() error {
	b, err := queue.NewBroker(c.Broker)
	if err != nil {
		return err
	}

	c.Q, err = b.Queue(c.Queue)
	if err != nil {
		return err
	}

	return nil
}
