package queue

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"
	lookout_mock "github.com/src-d/lookout/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	queue "gopkg.in/src-d/go-queue.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var (
	longLineFixture = fixtures.GetAll()[0]

	mockEventA = lookout.ReviewEvent{
		ReviewEvent: pb.ReviewEvent{
			Provider:       "github",
			InternalID:     "1234",
			CommitRevision: *longLineFixture.GetCommitRevision(),
		}}

	mockEventB = lookout.PushEvent{
		PushEvent: pb.PushEvent{
			Provider:       "github",
			InternalID:     "5678",
			CommitRevision: *longLineFixture.GetCommitRevision(),
		}}

	fakeEvent = lookout_mock.FakeEvent{}
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

type QueueJobTestSuite struct {
	suite.Suite
}

func (s *QueueJobTestSuite) TestCreationWithReviewEvent() {
	require := s.Require()

	qJob, err := NewQueueJob(context.TODO(), &mockEventA)
	require.NoError(err)
	require.NotNil(qJob)

	qEv, err := qJob.ToInterface()
	require.NoError(err)
	require.NotNil(qEv)
	require.EqualValues(&mockEventA, qEv)
	require.EqualValues(mockEventA.Type(), qJob.EventType)

	require.Nil(qJob.PushEvent)
}

func (s *QueueJobTestSuite) TestCreationWithPushEvent() {
	require := s.Require()

	qJob, err := NewQueueJob(context.TODO(), &mockEventB)
	require.NoError(err)
	require.NotNil(qJob)

	qEv, err := qJob.ToInterface()
	require.NoError(err)
	require.NotNil(qEv)
	require.EqualValues(&mockEventB, qEv)
	require.EqualValues(mockEventB.Type(), qJob.EventType)

	require.Nil(qJob.ReviewEvent)
}

func (s *QueueJobTestSuite) TestCreationWithFakeEvent() {
	require := s.Require()

	qJob, err := NewQueueJob(context.TODO(), &fakeEvent)
	require.EqualError(err, "unsupported event type *mock.FakeEvent")
	require.Nil(qJob)
}

func (s *QueueJobTestSuite) TestEventMethodWithFakeEvent() {
	require := s.Require()

	qj := QueueJob{Event: &Event{}}
	qEv, err := qj.ToInterface()
	require.EqualError(err, "unknown lookout event")
	require.Nil(qEv)
}

func TestQueueJobTestSuite(t *testing.T) {
	suite.Run(t, new(QueueJobTestSuite))
}

type EventEnqueuerTestSuite struct {
	suite.Suite
}

func (s *EventEnqueuerTestSuite) TestEnqueueFakeEvent() {
	q := initQueue(s.T(), "memoryfinite://")
	handler := EventEnqueuer(context.TODO(), q)

	err := handler(context.TODO(), &fakeEvent)
	s.EqualError(err, "unsupported event type *mock.FakeEvent")
}

func (s *EventEnqueuerTestSuite) TestErrorOnQueueJobCreation() {
	mq := new(MockQueue)

	mq.On("Publish", mock.Anything).Return(errors.New("publish mock error"))

	handler := EventEnqueuer(context.TODO(), mq)

	err := handler(context.TODO(), &mockEventA)
	s.EqualError(err, "publish mock error")
}

func (s *EventEnqueuerTestSuite) TestNoCache() {
	// Enqueue the same event twice, dequeue it twice

	t := s.T()
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

func (s *EventEnqueuerTestSuite) TestWithCache() {
	// Enqueue the same event twice, it should be available to dequeue only once

	t := s.T()
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

func TestEventEnqueuerTestSuite(t *testing.T) {
	suite.Run(t, new(EventEnqueuerTestSuite))
}

type MockJobIter struct {
	mock.Mock
	io.Closer
}

func (m *MockJobIter) Next() (*queue.Job, error) {
	args := m.Called()
	return args.Get(0).(*queue.Job), args.Error(1)
}

type MockQueue struct {
	mock.Mock
}

func (m *MockQueue) Publish(j *queue.Job) error {
	args := m.Called(j)
	return args.Error(0)
}

func (m *MockQueue) PublishDelayed(j *queue.Job, t time.Duration) error {
	args := m.Called(j, t)
	return args.Error(0)
}

func (m *MockQueue) Transaction(tc queue.TxCallback) error {
	args := m.Called(tc)
	return args.Error(0)
}
func (m *MockQueue) Consume(advertisedWindow int) (queue.JobIter, error) {
	args := m.Called(advertisedWindow)
	rawJobIter := args.Get(0)
	if rawJobIter != nil {
		return rawJobIter.(*MockJobIter), nil
	}

	return nil, args.Error(1)
}

func (m *MockQueue) RepublishBuried(conditions ...queue.RepublishConditionFunc) error {
	args := m.Called(conditions)
	return args.Error(0)
}

type EventDequeuerTestSuite struct {
	suite.Suite
}

func (s *EventDequeuerTestSuite) TestWrongConcurrencyValue() {
	q := initQueue(s.T(), "memory://")
	handler := func(context.Context, lookout.Event) error {
		return nil
	}
	err := RunEventDequeuer(context.TODO(), q, handler, 0)
	s.EqualError(err, "wrong value 0 for concurrent argument")
}

func (s *EventDequeuerTestSuite) TestErrorOnQueueConsumption() {
	mq := new(MockQueue)

	mq.On("Consume", mock.Anything).Return(nil, errors.New("consume mock error"))

	handler := func(context.Context, lookout.Event) error {
		return nil
	}
	err := RunEventDequeuer(context.TODO(), mq, handler, 1)
	s.EqualError(err, "queue consume failed: consume mock error")

	mq.AssertExpectations(s.T())
}

func (s *EventDequeuerTestSuite) TestSimple() {
	t := s.T()
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

func (s *EventDequeuerTestSuite) TestConcurrent() {
	t := s.T()
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

func TestEventDequeuerTestSuite(t *testing.T) {
	suite.Run(t, new(EventDequeuerTestSuite))
}
