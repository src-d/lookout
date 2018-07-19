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

func (s *ScannerSuite) TestTreeScannerChanges() {
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

func (s *ScannerSuite) TestTreeScannerFiles() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewTreeScanner(headTree)
	var files []*lookout.File
	for cs.Next() {
		files = append(files, cs.File())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(files, 9)
}

type filterScannerFixture struct {
	IncludePattern string
	ExcludePattern string
}

func (s *ScannerSuite) TestFilterScannerIncludeAll() {
	fixtures := []filterScannerFixture{
		{},
		{IncludePattern: ".*"},
	}

	for i, fixture := range s.changeFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("change case %d", i), s.testChangeFilterScannerFixture(fixture, 9))
	}

	for i, fixture := range s.fileFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("file case %d", i), s.testFileFilterScannerFixture(fixture, 9))
	}
}

func (s *ScannerSuite) TestFilterIncludeSome() {
	fixtures := []filterScannerFixture{
		{IncludePattern: `.*\.go`},
		{IncludePattern: `.*\.go`, ExcludePattern: `.*\.php`},
	}

	for i, fixture := range s.changeFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("change case %d", i), s.testChangeFilterScannerFixture(fixture, 2))
	}

	for i, fixture := range s.fileFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("file case %d", i), s.testFileFilterScannerFixture(fixture, 2))
	}
}

func (s *ScannerSuite) TestFilterExcludeOne() {
	fixtures := []filterScannerFixture{
		{IncludePattern: "", ExcludePattern: `\.gitignore`},
		{IncludePattern: ".*", ExcludePattern: `json/short\.json`},
	}

	for i, fixture := range s.changeFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("change case %d", i), s.testChangeFilterScannerFixture(fixture, 8))
	}

	for i, fixture := range s.fileFixtures(fixtures) {
		s.T().Run(fmt.Sprintf("file case %d", i), s.testFileFilterScannerFixture(fixture, 8))
	}
}

func (s *ScannerSuite) changeFixtures(fs []filterScannerFixture) []*lookout.ChangesRequest {
	res := make([]*lookout.ChangesRequest, len(fs), len(fs))
	for i, f := range fs {
		res[i] = &lookout.ChangesRequest{
			IncludePattern: f.IncludePattern,
			ExcludePattern: f.ExcludePattern,
		}
	}
	return res
}

func (s *ScannerSuite) fileFixtures(fs []filterScannerFixture) []*lookout.FilesRequest {
	res := make([]*lookout.FilesRequest, len(fs), len(fs))
	for i, f := range fs {
		res[i] = &lookout.FilesRequest{
			IncludePattern: f.IncludePattern,
			ExcludePattern: f.ExcludePattern,
		}
	}
	return res
}

func (s *ScannerSuite) testChangeFilterScannerFixture(fixture *lookout.ChangesRequest, len int) func(t *testing.T) {
	return func(t *testing.T) {
		require := require.New(t)

		head := s.getCommit(s.Basic.Head)
		headTree, err := head.Tree()
		require.NoError(err)

		cs := NewChangeFilterScanner(
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

		require.Len(changes, len)
	}
}

func (s *ScannerSuite) testFileFilterScannerFixture(fixture *lookout.FilesRequest, len int) func(t *testing.T) {
	return func(t *testing.T) {
		require := require.New(t)

		head := s.getCommit(s.Basic.Head)
		headTree, err := head.Tree()
		require.NoError(err)

		cs := NewFileFilterScanner(
			NewTreeScanner(headTree),
			fixture.IncludePattern, fixture.ExcludePattern,
		)

		var files []*lookout.File
		for cs.Next() {
			files = append(files, cs.File())
		}

		require.False(cs.Next())
		require.NoError(cs.Err())
		require.NoError(cs.Close())

		require.Len(files, len)
	}
}

func (s *ScannerSuite) TestChangeBlobScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewChangeBlobScanner(
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

func (s *ScannerSuite) TestFileBlobScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewFileBlobScanner(
		NewTreeScanner(headTree),
		headTree,
	)

	files := make(map[string]*lookout.File)
	for cs.Next() {
		f := cs.File()
		files[f.Path] = f
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(files, 9)
	require.Equal(`package main

import "fmt"

func main() {
	fmt.Println("Hello, playground")
}
`, string(files["vendor/foo.go"].Content))
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

func (s *ScannerSuite) TestFileChangeVendorScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewChangeExcludeVendorScanner(NewTreeScanner(headTree))

	var changes []*lookout.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 7)
}

func (s *ScannerSuite) TestFileExcludeVendorScanner() {
	require := s.Require()

	head := s.getCommit(s.Basic.Head)
	headTree, err := head.Tree()
	require.NoError(err)

	cs := NewFileExcludeVendorScanner(NewTreeScanner(headTree))

	var files []*lookout.File
	for cs.Next() {
		files = append(files, cs.File())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(files, 7)
}
