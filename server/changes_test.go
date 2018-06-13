package server

import (
	"testing"

	"github.com/src-d/lookout/api"

	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type GitChangeScannerSuite struct {
	suite.Suite
	Basic  *fixtures.Fixture
	Storer storer.Storer
}

func TestGitChangeScannerSuite(t *testing.T) {
	suite.Run(t, new(GitChangeScannerSuite))
}

func (s *GitChangeScannerSuite) SetupSuite() {
	require := s.Require()

	err := fixtures.Init()
	require.NoError(err)

	fixture := fixtures.Basic().One()
	fs := fixture.DotGit()
	sto, err := filesystem.NewStorage(fs)
	require.NoError(err)

	s.Basic = fixture
	s.Storer = sto
}

func (s *GitChangeScannerSuite) TearDownSuite() {
	require := s.Require()

	err := fixtures.Clean()
	require.NoError(err)
}

func (s *GitChangeScannerSuite) getCommit(h plumbing.Hash) *object.Commit {
	s.T().Helper()
	require := s.Require()
	obj, err := s.Storer.EncodedObject(plumbing.CommitObject, h)
	require.NoError(err)
	commit, err := object.DecodeCommit(s.Storer, obj)
	require.NoError(err)
	return commit
}

func (s *GitChangeScannerSuite) TestOneCommit() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewGitChangeScanner(s.Storer, nil, headTree)
	var changes []*api.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 9)
}
