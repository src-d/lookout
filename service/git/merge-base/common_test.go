package merge_base

import (
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type baseTestSuite struct {
	suite.Suite
	fixture *fixtures.Fixture
	store   storer.EncodedObjectStorer
}

func (s *baseTestSuite) SetupSuite() {
	err := fixtures.Init()
	s.NoError(err)
	s.fixture = fixtures.Basic().One()
}

func (s *baseTestSuite) TearDownSuite() {
	err := fixtures.Clean()
	s.NoError(err)
}

func (s *baseTestSuite) assertCommits(commits []*object.Commit, hashes []string) {
	require := s.Require()
	require.Len(commits, len(hashes))
	for i, commit := range commits {
		require.Equal(hashes[i], commit.Hash.String())
	}
}

func (s *baseTestSuite) commit(h plumbing.Hash) *object.Commit {
	commit, err := object.GetCommit(s.store, h)
	s.NoError(err)
	return commit
}

// NewRepository returns a new repository using the .git folder, using memfs filesystem as worktree.
func newRepository(f *fixtures.Fixture) *git.Repository {
	var worktree, dotgit billy.Filesystem
	dotgit = f.DotGit()
	worktree = memfs.New()

	st := filesystem.NewStorage(dotgit, cache.NewObjectLRUDefault())

	r, err := git.Open(st, worktree)

	if err != nil {
		panic(err)
	}

	return r
}

func commitsFromHashes(repo *git.Repository, hashes []string) ([]*object.Commit, error) {
	var commits []*object.Commit
	for _, hash := range hashes {
		commit, err := repo.CommitObject(plumbing.NewHash(hash))
		if err != nil {
			return nil, err
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

func commitsFromIter(iter object.CommitIter) ([]*object.Commit, error) {
	var commits []*object.Commit
	err := iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})

	return commits, err
}
