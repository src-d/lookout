package git

import (
	"context"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/mock"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type MockCommitLoader struct {
	mock.Mock
}

func (m *MockCommitLoader) LoadCommits(ctx context.Context,
	rps ...lookout.ReferencePointer) ([]*object.Commit, error) {

	args := m.Called(ctx, rps)
	r0 := args.Get(0)
	if r0 == nil {
		return nil, args.Error(1)
	}

	return r0.([]*object.Commit), args.Error(1)
}
