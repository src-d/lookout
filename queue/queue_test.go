package queue

import (
	"context"
	"io"
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

func initQueue(t *testing.T) queue.Queue {
	b, err := queue.NewBroker("memoryfinite://")
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
	qJob, err := NewQueueJob(&mockEventA)
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

	q := initQueue(t)

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

	q := initQueue(t)

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
