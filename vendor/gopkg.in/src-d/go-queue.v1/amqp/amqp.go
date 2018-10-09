package queue

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/src-d/go-queue.v1"

	"github.com/jpillora/backoff"
	"github.com/kelseyhightower/envconfig"
	"github.com/streadway/amqp"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-log.v1"
)

func init() {
	err := envconfig.Process("amqp", &DefaultConfiguration)
	if err != nil {
		panic(err)
	}

	queue.Register("amqp", func(uri string) (queue.Broker, error) {
		return New(uri)
	})
}

// DefaultConfiguration contains the default configuration initialized from
// environment variables.
var DefaultConfiguration Configuration

// Configuration AMQP configuration settings, this settings are set using the
// environment variables.
type Configuration struct {
	BuriedQueueSuffix    string `envconfig:"BURIED_QUEUE_SUFFIX" default:".buriedQueue"`
	BuriedExchangeSuffix string `envconfig:"BURIED_EXCHANGE_SUFFIX" default:".buriedExchange"`
	BuriedTimeout        int    `envconfig:"BURIED_TIMEOUT" default:"500"`

	RetriesHeader string `envconfig:"RETRIES_HEADER" default:"x-retries"`
	ErrorHeader   string `envconfig:"ERROR_HEADER" default:"x-error-type"`

	BackoffMin    time.Duration `envconfig:"BACKOFF_MIN" default:"200ms"`
	BackoffMax    time.Duration `envconfig:"BACKOFF_MAX" default:"30s"`
	BackoffFactor float64       `envconfig:"BACKOFF_FACTOR" default:"2"`
}

var consumerSeq uint64

var (
	ErrConnectionFailed = errors.NewKind("failed to connect to RabbitMQ: %s")
	ErrOpenChannel      = errors.NewKind("failed to open a channel: %s")
	ErrRetrievingHeader = errors.NewKind("error retrieving '%s' header from message %s")
	ErrRepublishingJobs = errors.NewKind("couldn't republish some jobs : %s")
)

// Broker implements the queue.Broker interface for AMQP, such as RabbitMQ.
type Broker struct {
	mut        sync.RWMutex
	conn       *amqp.Connection
	ch         *amqp.Channel
	connErrors chan *amqp.Error
	chErrors   chan *amqp.Error
	stop       chan struct{}
	backoff    *backoff.Backoff
}

type connection interface {
	connection() *amqp.Connection
	channel() *amqp.Channel
}

// New creates a new AMQPBroker.
func New(url string) (queue.Broker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, ErrConnectionFailed.New(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, ErrOpenChannel.New(err)
	}

	b := &Broker{
		conn: conn,
		ch:   ch,
		stop: make(chan struct{}),
		backoff: &backoff.Backoff{
			Min:    DefaultConfiguration.BackoffMin,
			Max:    DefaultConfiguration.BackoffMax,
			Factor: DefaultConfiguration.BackoffFactor,
			Jitter: false,
		},
	}

	b.connErrors = make(chan *amqp.Error, 1)
	b.conn.NotifyClose(b.connErrors)

	b.chErrors = make(chan *amqp.Error, 1)
	b.ch.NotifyClose(b.chErrors)

	go b.manageConnection(url)

	return b, nil
}

func (b *Broker) manageConnection(url string) {
	for {
		select {
		case err := <-b.connErrors:
			log.Errorf(err, "amqp connection error - reconnecting")
			if err == nil {
				break
			}
			b.reconnect(url)

		case err := <-b.chErrors:
			log.Errorf(err, "amqp channel error - reopening channel")
			if err == nil {
				break
			}
			b.reopenChannel()

		case <-b.stop:
			return
		}
	}
}

func (b *Broker) reconnect(url string) {
	b.backoff.Reset()

	b.mut.Lock()
	defer b.mut.Unlock()
	// open a new connection and channel
	b.conn = b.tryConnection(url)
	b.connErrors = make(chan *amqp.Error, 1)
	b.conn.NotifyClose(b.connErrors)

	b.ch = b.tryChannel(b.conn)
	b.chErrors = make(chan *amqp.Error, 1)
	b.ch.NotifyClose(b.chErrors)
}

func (b *Broker) reopenChannel() {
	b.backoff.Reset()

	b.mut.Lock()
	defer b.mut.Unlock()
	// open a new channel
	b.ch = b.tryChannel(b.conn)
	b.chErrors = make(chan *amqp.Error, 1)
	b.ch.NotifyClose(b.chErrors)
}

func (b *Broker) tryConnection(url string) *amqp.Connection {
	for {
		conn, err := amqp.Dial(url)
		if err == nil {
			b.backoff.Reset()
			return conn
		}
		d := b.backoff.Duration()

		log.Errorf(err, "error connecting to amqp, reconnecting in %s", d)
		time.Sleep(d)
	}
}

func (b *Broker) tryChannel(conn *amqp.Connection) *amqp.Channel {
	for {
		ch, err := conn.Channel()
		if err == nil {
			b.backoff.Reset()
			return ch
		}
		d := b.backoff.Duration()

		log.Errorf(err, "error creatting channel, new retry in %s", d)
		time.Sleep(d)
	}
}

func (b *Broker) connection() *amqp.Connection {
	b.mut.RLock()
	conn := b.conn
	b.mut.RUnlock()
	return conn
}

func (b *Broker) channel() *amqp.Channel {
	b.mut.RLock()
	ch := b.ch
	b.mut.RUnlock()
	return ch
}

func (b *Broker) newBuriedQueue(mainQueueName string) (q amqp.Queue, rex string, err error) {
	ch, err := b.conn.Channel()
	if err != nil {
		return
	}

	buriedName := mainQueueName + DefaultConfiguration.BuriedQueueSuffix
	rex = mainQueueName + DefaultConfiguration.BuriedExchangeSuffix

	if err = ch.ExchangeDeclare(rex, "fanout", true, false, false, false, nil); err != nil {
		return
	}

	q, err = b.ch.QueueDeclare(
		buriedName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return
	}

	if err = ch.QueueBind(buriedName, "", rex, true, nil); err != nil {
		return
	}

	return
}

// Queue returns the queue with the given name.
func (b *Broker) Queue(name string) (queue.Queue, error) {
	buriedQueue, rex, err := b.newBuriedQueue(name)
	if err != nil {
		return nil, err
	}

	q, err := b.channel().QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    rex,
			"x-dead-letter-routing-key": name,
			"x-max-priority":            uint8(queue.PriorityUrgent),
		},
	)

	if err != nil {
		return nil, err
	}

	return &Queue{
		conn:        b,
		queue:       q,
		buriedQueue: &Queue{conn: b, queue: buriedQueue},
	}, nil
}

// Close closes all the connections managed by the broker.
func (b *Broker) Close() error {
	close(b.stop)

	if err := b.channel().Close(); err != nil {
		return err
	}

	return b.connection().Close()
}

// Queue implements the Queue interface for the AMQP.
type Queue struct {
	conn        connection
	queue       amqp.Queue
	buriedQueue *Queue
}

// Publish publishes the given Job to the Queue.
func (q *Queue) Publish(j *queue.Job) (err error) {
	if j == nil || j.Size() == 0 {
		return queue.ErrEmptyJob.New()
	}

	headers := amqp.Table{}
	if j.Retries > 0 {
		headers[DefaultConfiguration.RetriesHeader] = j.Retries
	}

	if j.ErrorType != "" {
		headers[DefaultConfiguration.ErrorHeader] = j.ErrorType
	}

	for {
		err = q.conn.channel().Publish(
			"",           // exchange
			q.queue.Name, // routing key
			false,        // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				MessageId:    j.ID,
				Priority:     uint8(j.Priority),
				Timestamp:    j.Timestamp,
				ContentType:  j.ContentType,
				Body:         j.Raw,
				Headers:      headers,
			},
		)
		if err == nil {
			break
		}

		log.Errorf(err, "publishing to %s", q.queue.Name)
		if err != amqp.ErrClosed {
			break
		}
	}

	return err
}

// PublishDelayed publishes the given Job with a given delay. Delayed messages
// will not go into the buried queue if they fail.
func (q *Queue) PublishDelayed(j *queue.Job, delay time.Duration) error {
	if j == nil || j.Size() == 0 {
		return queue.ErrEmptyJob.New()
	}

	ttl := delay / time.Millisecond
	delayedQueue, err := q.conn.channel().QueueDeclare(
		j.ID,  // name
		true,  // durable
		true,  // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": q.queue.Name,
			"x-message-ttl":             int64(ttl),
			"x-expires":                 int64(ttl) * 2,
			"x-max-priority":            uint8(queue.PriorityUrgent),
		},
	)
	if err != nil {
		return err
	}

	for {
		err = q.conn.channel().Publish(
			"", // exchange
			delayedQueue.Name,
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				MessageId:    j.ID,
				Priority:     uint8(j.Priority),
				Timestamp:    j.Timestamp,
				ContentType:  j.ContentType,
				Body:         j.Raw,
			},
		)
		if err == nil {
			break
		}

		log.Errorf(err, "delay publishing to %s", q.queue.Name)
		if err != amqp.ErrClosed {
			break
		}
	}

	return err
}

type jobErr struct {
	job *queue.Job
	err error
}

// RepublishBuried will republish in the main queue those jobs that timed out without Ack
// or were Rejected with requeue = False and makes comply return true.
func (q *Queue) RepublishBuried(conditions ...queue.RepublishConditionFunc) error {
	if q.buriedQueue == nil {
		return fmt.Errorf("buriedQueue is nil, called RepublishBuried on the internal buried queue?")
	}

	// enforce prefetching only one job
	iter, err := q.buriedQueue.Consume(1)
	if err != nil {
		return err
	}

	defer iter.Close()

	timeout := time.Duration(DefaultConfiguration.BuriedTimeout) * time.Millisecond

	var notComplying []*queue.Job
	var errorsPublishing []*jobErr
	for {
		j, err := iter.(*JobIter).nextWithTimeout(timeout)
		if err != nil {
			return err
		}

		if j == nil {
			log.Debugf("no more jobs in the buried queue")

			break
		}

		if err = j.Ack(); err != nil {
			return err
		}

		if queue.RepublishConditions(conditions).Comply(j) {
			start := time.Now()
			if err = q.Publish(j); err != nil {
				log.With(log.Fields{
					"duration": time.Since(start),
					"id":       j.ID,
				}).Errorf(err, "error publishing job")

				errorsPublishing = append(errorsPublishing, &jobErr{j, err})
			} else {
				log.With(log.Fields{
					"duration": time.Since(start),
					"id":       j.ID,
				}).Debugf("job republished")
			}
		} else {
			log.With(log.Fields{
				"id":           j.ID,
				"error-type":   j.ErrorType,
				"content-type": j.ContentType,
				"retries":      j.Retries,
			}).Debugf("job does not comply with conditions")

			notComplying = append(notComplying, j)
		}
	}

	log.Debugf("rejecting %v non complying jobs", len(notComplying))

	for i, job := range notComplying {
		start := time.Now()

		if err = job.Reject(true); err != nil {
			return err
		}

		log.With(log.Fields{
			"duration": time.Since(start),
			"id":       job.ID,
		}).Debugf("job rejected (%v/%v)", i, len(notComplying))
	}

	return q.handleRepublishErrors(errorsPublishing)
}

func (q *Queue) handleRepublishErrors(list []*jobErr) error {
	if len(list) > 0 {
		stringErrors := []string{}
		for i, je := range list {
			stringErrors = append(stringErrors, je.err.Error())
			start := time.Now()

			if err := q.buriedQueue.Publish(je.job); err != nil {
				return err
			}

			log.With(log.Fields{
				"duration": time.Since(start),
				"id":       je.job.ID,
			}).Debugf("job reburied (%v/%v)", i, len(list))
		}

		return ErrRepublishingJobs.New(strings.Join(stringErrors, ": "))
	}

	return nil
}

// Transaction executes the given callback inside a transaction.
func (q *Queue) Transaction(txcb queue.TxCallback) error {
	ch, err := q.conn.connection().Channel()
	if err != nil {
		return ErrOpenChannel.New(err)
	}

	defer ch.Close()

	if err := ch.Tx(); err != nil {
		return err
	}

	txQueue := &Queue{
		conn: &Broker{
			conn: q.conn.connection(),
			ch:   ch,
		},
		queue: q.queue,
	}

	err = txcb(txQueue)
	if err != nil {
		if err := ch.TxRollback(); err != nil {
			return err
		}

		return err
	}

	return ch.TxCommit()
}

// Consume implements the Queue interface. The advertisedWindow value
// is the maximum number of unacknowledged jobs
func (q *Queue) Consume(advertisedWindow int) (queue.JobIter, error) {
	ch, err := q.conn.connection().Channel()
	if err != nil {
		return nil, ErrOpenChannel.New(err)
	}

	if err := ch.Qos(advertisedWindow, 0, false); err != nil {
		return nil, err
	}

	id := q.consumeID()
	c, err := ch.Consume(
		q.queue.Name, // queue
		id,           // consumer
		false,        // autoAck
		false,        // exclusive
		false,        // noLocal
		false,        // noWait
		nil,          // args
	)
	if err != nil {
		return nil, err
	}

	return &JobIter{id: id, ch: ch, c: c}, nil
}

func (q *Queue) consumeID() string {
	return fmt.Sprintf("%s-%s-%d",
		os.Args[0],
		q.queue.Name,
		atomic.AddUint64(&consumerSeq, 1),
	)
}

// JobIter implements the JobIter interface for AMQP.
type JobIter struct {
	id string
	ch *amqp.Channel
	c  <-chan amqp.Delivery
}

// Next returns the next job in the iter.
func (i *JobIter) Next() (*queue.Job, error) {
	d, ok := <-i.c
	if !ok {
		return nil, queue.ErrAlreadyClosed.New()
	}

	return fromDelivery(&d)
}

func (i *JobIter) nextWithTimeout(timeout time.Duration) (*queue.Job, error) {
	select {
	case d, ok := <-i.c:
		if !ok {
			return nil, queue.ErrAlreadyClosed.New()
		}

		return fromDelivery(&d)
	case <-time.After(timeout):
		return nil, nil
	}
}

// Close closes the channel of the JobIter.
func (i *JobIter) Close() error {
	if err := i.ch.Cancel(i.id, false); err != nil {
		return err
	}

	return i.ch.Close()
}

// Acknowledger implements the Acknowledger for AMQP.
type Acknowledger struct {
	ack amqp.Acknowledger
	id  uint64
}

// Ack signals acknowledgement.
func (a *Acknowledger) Ack() error {
	return a.ack.Ack(a.id, false)
}

// Reject signals rejection. If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *Acknowledger) Reject(requeue bool) error {
	return a.ack.Reject(a.id, requeue)
}

func fromDelivery(d *amqp.Delivery) (*queue.Job, error) {
	j, err := queue.NewJob()
	if err != nil {
		return nil, err
	}

	j.ID = d.MessageId
	j.Priority = queue.Priority(d.Priority)
	j.Timestamp = d.Timestamp
	j.ContentType = d.ContentType
	j.Acknowledger = &Acknowledger{d.Acknowledger, d.DeliveryTag}
	j.Raw = d.Body

	if retries, ok := d.Headers[DefaultConfiguration.RetriesHeader]; ok {
		retries, ok := retries.(int32)
		if !ok {
			return nil, ErrRetrievingHeader.New(DefaultConfiguration.RetriesHeader, d.MessageId)
		}

		j.Retries = retries
	}

	if errorType, ok := d.Headers[DefaultConfiguration.ErrorHeader]; ok {
		errorType, ok := errorType.(string)
		if !ok {
			return nil, ErrRetrievingHeader.New(DefaultConfiguration.ErrorHeader, d.MessageId)
		}

		j.ErrorType = errorType
	}

	return j, nil
}
