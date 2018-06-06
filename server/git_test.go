package server

import (
	"testing"

	"github.com/src-d/lookout/api"
	"github.com/stretchr/testify/suite"
	fixtures "gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type GitDataReaderSuite struct {
	suite.Suite
	Basic  *fixtures.Fixture
	Storer storer.Storer
}

func TestGitDataReaderSuite(t *testing.T) {
	suite.Run(t, new(GitDataReaderSuite))
}

func (s *GitDataReaderSuite) SetupSuite() {
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

func (s *GitDataReaderSuite) TearDownSuite() {
	require := s.Require()

	err := fixtures.Clean()
	require.NoError(err)
}

func (s *GitDataReaderSuite) Test() {
	require := s.Require()

	dr := NewGitDataReader(server.MapLoader{
		"repo:///myrepo": s.Storer,
	})

	resp, err := dr.GetChanges(&api.ChangesRequest{
		Repository: "repo:///myrepo",
		Top:        s.Basic.Head.String(),
	})
	require.NoError(err)
	require.NotNil(resp)
}

func (s *GitDataReaderSuite) TestErrorNoRepository() {
	require := s.Require()

	dr := NewGitDataReader(server.MapLoader{})

	resp, err := dr.GetChanges(&api.ChangesRequest{
		Repository: "repo:///myrepo",
		Top:        s.Basic.Head.String(),
	})
	require.Error(err)
	require.Nil(resp)
}

func (s *GitDataReaderSuite) TestErrorBadTop() {
	require := s.Require()

	dr := NewGitDataReader(server.MapLoader{
		"repo:///myrepo": s.Storer,
	})

	resp, err := dr.GetChanges(&api.ChangesRequest{
		Repository: "repo:///myrepo",
		Top:        "979a482e63de12d39675ff741c5a0cf4f068c109",
	})
	require.Error(err)
	require.Nil(resp)
}
