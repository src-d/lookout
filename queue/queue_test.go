package queue

import (
	"context"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-queue.v1"
)

var (
	longLineFixture = fixtures.GetAll()[0]

	mockEventA = lookout.ReviewEvent{
		Provider:       "github",
		InternalID:     "1234",
		CommitRevision: *longLineFixture.GetCommitRevision(),
	}

	mockEventB = lookout.PushEvent{
		Provider:       "github",
		InternalID:     "5678",
		CommitRevision: *longLineFixture.GetCommitRevision(),
	}
)

func initQueue(t *testing.T, broker string) queue.Queue {
	b, err := queue.NewBroker(broker)
	require.NoError(t, err)

	q, err := b.Queue("lookout-test")
	require.NoError(t, err)

	return q
}

func nextOK(t *testing.T, iter queue.JobIter) {
	retrievedJob, err := iter.Next()
	assert.NoError(t, err)
	assert.NoError(t, retrievedJob.Ack())
}

func nextEOF(t *testing.T, iter queue.JobIter) {
	retrievedJob, err := iter.Next()
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, retrievedJob)
}

func TestQueueJobCreation(t *testing.T) {
	qJob, err := NewQueueJob(context.TODO(), &mockEventA)
	require.NoError(t, err)
	require.NotNil(t, qJob)

	qEv, err := qJob.Event()
	require.NoError(t, err)
	require.NotNil(t, qEv)
	require.EqualValues(t, &mockEventA, qEv)
	require.EqualValues(t, mockEventA.Type(), qJob.EventType)

	require.Nil(t, qJob.PushEvent)
}

func TestEnqueuerNoCache(t *testing.T) {
	// Enqueue the same event twice, dequeue it twice

	q := initQueue(t, "memoryfinite://")

	handler := EventEnqueuer(context.TODO(), q)
	handler(context.TODO(), &mockEventA)
	handler(context.TODO(), &mockEventB)
	handler(context.TODO(), &mockEventA)

	advertisedWindow := 0 // ignored by memory brokers
	iter, err := q.Consume(advertisedWindow)
	assert.NoError(t, err)

	// A, B, A
	nextOK(t, iter)
	nextOK(t, iter)
	nextOK(t, iter)

	nextEOF(t, iter)
}

func TestEnqueuerCache(t *testing.T) {
	// Enqueue the same event twice, it should be available to dequeue only once

	q := initQueue(t, "memoryfinite://")

	handler := lookout.CachedHandler(EventEnqueuer(context.TODO(), q))
	handler(context.TODO(), &mockEventA)
	handler(context.TODO(), &mockEventB)
	handler(context.TODO(), &mockEventA)

	advertisedWindow := 0 // ignored by memory brokers
	iter, err := q.Consume(advertisedWindow)
	assert.NoError(t, err)

	// A, B
	nextOK(t, iter)
	nextOK(t, iter)

	nextEOF(t, iter)
}

func TestDequeuer(t *testing.T) {
	q := initQueue(t, "memory://")

	var wg sync.WaitGroup
	wg.Add(2)

	calls := 0
	handler := func(context.Context, lookout.Event) error {
		calls++
		wg.Done()
		return nil
	}

	go RunEventDequeuer(context.TODO(), q, handler, 1)

	assert.Equal(t, 0, calls)

	enq := EventEnqueuer(context.TODO(), q)
	enq(context.TODO(), &mockEventA)
	enq(context.TODO(), &mockEventB)

	wg.Wait()
	assert.Equal(t, 2, calls)
}

func TestDequeuerConcurrent(t *testing.T) {
	testCases := []int{1, 2, 13, 150}

	for _, n := range testCases {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			var called sync.WaitGroup

			var calls int32
			atomic.StoreInt32(&calls, 0)
			handler := func(context.Context, lookout.Event) error {
				atomic.AddInt32(&calls, 1)
				called.Done()
				wg.Wait()
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			q := initQueue(t, "memory://")
			go RunEventDequeuer(ctx, q, handler, n)

			assert.EqualValues(t, 0, atomic.LoadInt32(&calls))

			called.Add(n)

			// Enqueue some jobs, 3 * n of goroutines
			enq := EventEnqueuer(ctx, q)
			for i := 0; i < n*3; i++ {
				enq(ctx, &mockEventA)
			}

			// The first batch of handler calls should be exactly N
			called.Wait()
			assert.EqualValues(t, n, atomic.LoadInt32(&calls))

			// Let the dequeuer go though all the jobs, should be 3*N
			called.Add(2 * n)
			wg.Done()
			called.Wait()

			assert.EqualValues(t, 3*n, atomic.LoadInt32(&calls))

			cancel()
		})
	}
}
