package lookout

import (
	"context"
	"fmt"
	"testing"

	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var (
	longLineFixture = fixtures.GetAll()[0]

	mockEventA = ReviewEvent{ReviewEvent: pb.ReviewEvent{
		Provider:       "github",
		InternalID:     "1234",
		CommitRevision: *longLineFixture.GetCommitRevision(),
	}}

	mockEventB = PushEvent{PushEvent: pb.PushEvent{
		Provider:       "github",
		InternalID:     "5678",
		CommitRevision: *longLineFixture.GetCommitRevision(),
	}}
)

func TestCachedHandler(t *testing.T) {
	calls := 0

	handler := CachedHandler(func(context.Context, Event) error {
		calls++
		return nil
	})

	handler(context.TODO(), &mockEventA)
	handler(context.TODO(), &mockEventB)
	handler(context.TODO(), &mockEventA)

	assert.Equal(t, 2, calls)
}

func TestCachedHandlerErr(t *testing.T) {
	calls := 0

	handler := CachedHandler(func(context.Context, Event) error {
		calls++
		return fmt.Errorf("failure")
	})

	handler(context.TODO(), &mockEventA)
	handler(context.TODO(), &mockEventB)
	handler(context.TODO(), &mockEventA)

	assert.Equal(t, 3, calls)
}
