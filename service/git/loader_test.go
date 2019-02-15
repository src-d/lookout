package git

import (
	"context"
	"errors"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type MockCommitLoader struct {
	mock.Mock
}

func (m *MockCommitLoader) LoadCommits(ctx context.Context,
	rps ...lookout.ReferencePointer) ([]*object.Commit, storer.Storer, error) {

	args := m.Called(ctx, rps)
	r0 := args.Get(0)
	if r0 == nil {
		return nil, nil, args.Error(1)
	}

	return r0.([]*object.Commit), nil, args.Error(1)
}

type MockSyncer struct {
	mock.Mock
}

func (m *MockSyncer) Sync(ctx context.Context, rps ...lookout.ReferencePointer) error {
	args := m.Called(ctx, rps)
	return args.Error(0)
}

type MockLibrary struct {
	mock.Mock
}

func (m *MockLibrary) GetOrInit(ctx context.Context, url *lookout.RepositoryInfo) (*git.Repository, error) {
	args := m.Called(ctx, url)
	repo := args.Get(0)
	if repo != nil {
		return repo.(*git.Repository), nil
	}

	return nil, args.Error(1)
}

func (m *MockLibrary) Init(ctx context.Context, url *lookout.RepositoryInfo) (*git.Repository, error) {
	args := m.Called(ctx, url)
	repo := args.Get(0)
	if repo != nil {
		return repo.(*git.Repository), nil
	}

	return nil, args.Error(1)
}

func (m *MockLibrary) Has(url *lookout.RepositoryInfo) (bool, error) {
	args := m.Called(url)
	return args.Bool(0), args.Error(1)
}

func (m *MockLibrary) Get(ctx context.Context, url *lookout.RepositoryInfo) (*git.Repository, error) {
	args := m.Called(ctx, url)
	repo := args.Get(0)
	if repo != nil {
		return repo.(*git.Repository), nil
	}

	return nil, args.Error(1)
}

var rpsDifferentRepos = []lookout.ReferencePointer{
	lookout.ReferencePointer{
		InternalRepositoryURL: "file://repo-1",
	},
	lookout.ReferencePointer{
		InternalRepositoryURL: "file://repo-2",
	},
}
var rpsSameRepos = []lookout.ReferencePointer{
	lookout.ReferencePointer{
		InternalRepositoryURL: "file://repo",
	},
	lookout.ReferencePointer{
		InternalRepositoryURL: "file://repo",
	},
}

type LibraryCommitLoaderTestSuite struct {
	suite.Suite
}

func (s *LibraryCommitLoaderTestSuite) TestErrorOnMultiRepos() {
	require := s.Require()

	cl := NewLibraryCommitLoader(&Library{}, &LibrarySyncer{})

	commits, _, err := cl.LoadCommits(context.TODO(), rpsDifferentRepos...)

	require.Nil(commits)
	require.Errorf(err, "loading commits from multiple repositories is not supported")
}

func (s *LibraryCommitLoaderTestSuite) TestErrorOnSync() {
	require := s.Require()

	ctx := context.TODO()

	ms := new(MockSyncer)
	ms.On("Sync", ctx, mock.Anything).Return(
		errors.New("sync mock error"))

	cl := NewLibraryCommitLoader(&Library{}, ms)

	commits, _, err := cl.LoadCommits(ctx, rpsSameRepos...)

	require.Nil(commits)
	require.EqualError(err, "sync mock error")
	ms.AssertExpectations(s.T())
}

func (s *LibraryCommitLoaderTestSuite) TestErrorOnGetOrInit() {
	require := s.Require()

	ctx := context.TODO()

	ms := new(MockSyncer)
	ml := new(MockLibrary)
	ms.On("Sync", ctx, mock.Anything).Return(nil)
	ml.On("GetOrInit", ctx, mock.Anything).Return(
		nil, errors.New("get or init mock error"))

	cl := NewLibraryCommitLoader(ml, ms)

	commits, _, err := cl.LoadCommits(ctx, rpsSameRepos...)

	require.Nil(commits)
	require.EqualError(err, "get or init mock error")
	ms.AssertExpectations(s.T())
}

func (s *LibraryCommitLoaderTestSuite) TestEmpty() {
	require := s.Require()

	cl := NewLibraryCommitLoader(&Library{}, &LibrarySyncer{})

	rps := []lookout.ReferencePointer{}

	commits, _, err := cl.LoadCommits(context.TODO(), rps...)

	require.Nil(commits)
	require.Nil(err)
}

func TestLibraryCommitLoaderTestSuite(t *testing.T) {
	suite.Run(t, new(LibraryCommitLoaderTestSuite))
}
