package git

import (
	"fmt"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type ScannerSuite struct {
	suite.Suite
	Basic  *fixtures.Fixture
	Storer storer.Storer
}

func TestScannerSuiteSuite(t *testing.T) {
	suite.Run(t, new(ScannerSuite))
}

func (s *ScannerSuite) SetupSuite() {
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

func (s *ScannerSuite) TearDownSuite() {
	require := s.Require()

	err := fixtures.Clean()
	require.NoError(err)
}

func (s *ScannerSuite) getCommit(h plumbing.Hash) *object.Commit {
	s.T().Helper()
	require := s.Require()
	obj, err := s.Storer.EncodedObject(plumbing.CommitObject, h)
	require.NoError(err)
	commit, err := object.DecodeCommit(s.Storer, obj)
	require.NoError(err)
	return commit
}

func (s *ScannerSuite) TestTreeScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewTreeScanner(headTree)
	var changes []*lookout.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 9)
}

func (s *ScannerSuite) TestFilterScannerIncludeAll() {
	fixtures := []*lookout.ChangesRequest{
		{},
		{IncludePattern: ".*"},
	}

	for i, fixture := range fixtures {
		s.T().Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			require := require.New(t)

			head := s.getCommit(s.Basic.Head)
			headTree, err := head.Tree()
			require.NoError(err)

			cs := NewFilterScanner(
				NewTreeScanner(headTree),
				fixture.IncludePattern, fixture.ExcludePattern,
			)

			var changes []*lookout.Change
			for cs.Next() {
				changes = append(changes, cs.Change())
			}

			require.False(cs.Next())
			require.NoError(cs.Err())
			require.NoError(cs.Close())

			require.Len(changes, 9)
		})
	}
}

func (s *ScannerSuite) TestFilterIncludeSome() {
	fixtures := []*lookout.ChangesRequest{
		{IncludePattern: `.*\.go`},
		{IncludePattern: `.*\.go`, ExcludePattern: `.*\.php`},
	}

	for i, fixture := range fixtures {
		s.T().Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			require := require.New(t)

			head := s.getCommit(s.Basic.Head)
			headTree, err := head.Tree()
			require.NoError(err)

			cs := NewFilterScanner(
				NewTreeScanner(headTree),
				fixture.IncludePattern, fixture.ExcludePattern,
			)

			var changes []*lookout.Change
			for cs.Next() {
				changes = append(changes, cs.Change())
			}

			require.False(cs.Next())
			require.NoError(cs.Err())
			require.NoError(cs.Close())

			require.Len(changes, 2)
		})
	}
}

func (s *ScannerSuite) TestFilterExcludeOne() {
	fixtures := []*lookout.ChangesRequest{
		{IncludePattern: "", ExcludePattern: `\.gitignore`},
		{IncludePattern: ".*", ExcludePattern: `json/short\.json`},
	}

	for i, fixture := range fixtures {
		s.T().Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			require := require.New(t)

			head := s.getCommit(s.Basic.Head)
			headTree, err := head.Tree()
			require.NoError(err)

			cs := NewFilterScanner(
				NewTreeScanner(headTree),
				fixture.IncludePattern, fixture.ExcludePattern,
			)

			var changes []*lookout.Change
			for cs.Next() {
				changes = append(changes, cs.Change())
			}

			require.False(cs.Next())
			require.NoError(cs.Err())
			require.NoError(cs.Close())

			require.Len(changes, 8)
		})
	}
}

func (s *ScannerSuite) TestBlobScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewBlobScanner(
		NewTreeScanner(headTree),
		nil, headTree,
	)

	changes := make(map[string]*lookout.Change)
	for cs.Next() {
		ch := cs.Change()
		changes[ch.Head.Path] = ch
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 9)
	require.Equal(`package main

import "fmt"

func main() {
	fmt.Println("Hello, playground")
}
`, string(changes["vendor/foo.go"].Head.Content))
}

func (s *ScannerSuite) TestDiffTreeScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	parent, err := head.Parent(0)
	require.NoError(err)
	parentTree, err := parent.Tree()
	require.NoError(err)

	cs := NewDiffTreeScanner(parentTree, headTree)
	var changes []*lookout.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 1)
}
