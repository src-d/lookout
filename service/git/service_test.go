package git

import (
	"context"
	"fmt"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	fixtures "gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type ServiceSuite struct {
	suite.Suite
	Basic  *fixtures.Fixture
	Storer storer.Storer
}

func (s *ServiceSuite) SetupSuite() {
	require := s.Require()

	err := fixtures.Init()
	require.NoError(err)

	fixture := fixtures.Basic().One()
	fs := fixture.DotGit()
	sto := filesystem.NewStorage(fs, cache.NewObjectLRU(cache.DefaultMaxSize))

	s.Basic = fixture
	s.Storer = sto
}

func (s *ServiceSuite) TearDownSuite() {
	require := s.Require()

	err := fixtures.Clean()
	require.NoError(err)
}

func (s *ServiceSuite) buildRefPointer(internalRepoURL string, refName string, hash string) *lookout.ReferencePointer {
	return &lookout.ReferencePointer{
		InternalRepositoryURL: internalRepoURL,
		ReferenceName:         plumbing.ReferenceName(refName),
		Hash:                  hash,
	}
}

func TestServiceGetChangesSuite(t *testing.T) {
	suite.Run(t, new(ServiceGetChangesSuite))
}

func TestServiceGetFilesSuite(t *testing.T) {
	suite.Run(t, new(ServiceGetFilesSuite))
}

type ServiceGetChangesSuite struct {
	ServiceSuite
}

func (s *ServiceGetChangesSuite) getCommit(h plumbing.Hash) *object.Commit {
	s.T().Helper()
	require := s.Require()
	obj, err := s.Storer.EncodedObject(plumbing.CommitObject, h)
	require.NoError(err)
	commit, err := object.DecodeCommit(s.Storer, obj)
	require.NoError(err)
	return commit
}

func (s *ServiceGetChangesSuite) getHeadBaseFixtureHash() (string, string) {
	s.T().Helper()
	head := s.getCommit(s.Basic.Head)
	parent, _ := head.Parent(0)
	grandParent, _ := parent.Parent(0)
	base, _ := grandParent.Parent(0)

	return head.Hash.String(), base.Hash.String()
}

func (s *ServiceGetChangesSuite) requireScannerChanges(req *lookout.ChangesRequest, expectedChanges int) map[string]*lookout.Change {
	s.T().Helper()
	require := s.Require()

	srv := NewService(&StorerCommitLoader{s.Storer})
	scanner, err := srv.GetChanges(context.TODO(), req)

	require.NoError(err)
	require.NotNil(scanner)

	changes := make(map[string]*lookout.Change)
	for scanner.Next() {
		ch := scanner.Change()
		changes[ch.Head.Path] = ch
	}

	require.False(scanner.Next())
	require.NoError(scanner.Err())
	require.NoError(scanner.Close())

	require.Len(changes, expectedChanges)

	return changes
}

func (s *ServiceGetChangesSuite) TestInvalidReference() {
	srv := NewService(&StorerCommitLoader{s.Storer})

	headHash, _ := s.getHeadBaseFixtureHash()

	changesRequests := [4]*lookout.ChangesRequest{
		&lookout.ChangesRequest{Head: &lookout.ReferencePointer{}},
		&lookout.ChangesRequest{Head: s.buildRefPointer("file:///myrepo", "referenceName", "")},
		&lookout.ChangesRequest{Head: s.buildRefPointer("", "referenceName", headHash)},
		&lookout.ChangesRequest{Head: s.buildRefPointer("file:///myrepo", "", headHash)},
	}

	testNames := [4]string{
		"empty head",
		"empty string in hash",
		"empty string in internal repo url",
		"empty string in reference name",
	}

	for i, changesReq := range changesRequests {
		s.T().Run(testNames[i], func(t *testing.T) {
			require := require.New(t)
			_, err := srv.GetChanges(context.TODO(), changesReq)

			require.Error(err)
			require.True(ErrRefValidation.Is(err))
		})
	}
}

func (s *ServiceGetChangesSuite) TestInvalidTree() {
	require := s.Require()

	headHash, _ := s.getHeadBaseFixtureHash()

	headRp := s.buildRefPointer("file:///myrepo", "referenceName", headHash)
	rps := []lookout.ReferencePointer{*headRp}

	m := &MockCommitLoader{}
	m.On("LoadCommits", mock.Anything, rps).Once().Return(nil, fmt.Errorf("Error"))

	srv := NewService(m)

	scanner, err := srv.GetChanges(context.TODO(), &lookout.ChangesRequest{Head: headRp})
	require.Error(err)
	require.Nil(scanner)
}

func (s *ServiceGetChangesSuite) TestNoParams() {
	headHash, baseHash := s.getHeadBaseFixtureHash()

	changesRequestNoBase := &lookout.ChangesRequest{
		Head: s.buildRefPointer("file:///myrepo", "referenceName", headHash)}
	changesRequestWithBase := &lookout.ChangesRequest{
		Head: s.buildRefPointer("file:///myrepo", "referenceName", headHash),
		Base: s.buildRefPointer("file:///myrepo", "referenceName", baseHash),
	}

	changesRequests := [2]*lookout.ChangesRequest{changesRequestNoBase, changesRequestWithBase}
	testNames := [2]string{"without base", "with base"}
	expectedChanges := [2]int{9, 5}

	for i, changesReq := range changesRequests {
		s.T().Run(testNames[i], func(t *testing.T) {
			s.requireScannerChanges(changesReq, expectedChanges[i])
		})
	}
}

func (s *ServiceGetChangesSuite) TestWithPattern() {
	headHash, baseHash := s.getHeadBaseFixtureHash()

	fixtures := []filterScannerFixture{
		{IncludePattern: `.*\.go`},
		{ExcludePattern: `.*\.php`},
		{IncludePattern: `.*\.jpg`, ExcludePattern: `.*\.json`},
	}
	testNames := [3]string{"with inlcude pattern", "with exclude pattern", "with both include and exclude pattern"}

	expectedChangesWithoutBase := [3]int{2, 8, 1}
	expectedChangesWithBase := [3]int{2, 4, 0}
	for i, fixture := range fixtures {
		s.T().Run(fmt.Sprintf("without base - %s", testNames[i]), func(t *testing.T) {
			changesReq := &lookout.ChangesRequest{
				Head:           s.buildRefPointer("file:///myrepo", "referenceName", headHash),
				IncludePattern: fixture.IncludePattern,
				ExcludePattern: fixture.ExcludePattern,
			}

			s.requireScannerChanges(changesReq, expectedChangesWithoutBase[i])
		})

		s.T().Run(fmt.Sprintf("with base - %s", testNames[i]), func(t *testing.T) {
			changesReq := &lookout.ChangesRequest{
				Head:           s.buildRefPointer("file:///myrepo", "referenceName", headHash),
				Base:           s.buildRefPointer("file:///myrepo", "referenceName", baseHash),
				IncludePattern: fixture.IncludePattern,
				ExcludePattern: fixture.ExcludePattern,
			}

			s.requireScannerChanges(changesReq, expectedChangesWithBase[i])
		})
	}
}

func (s *ServiceGetChangesSuite) TestWithContent() {
	headHash, baseHash := s.getHeadBaseFixtureHash()

	changesRequestNoBase := &lookout.ChangesRequest{
		Head:         s.buildRefPointer("file:///myrepo", "referenceName", headHash),
		WantContents: true,
	}
	changesRequestWithBase := &lookout.ChangesRequest{
		Head:         s.buildRefPointer("file:///myrepo", "referenceName", headHash),
		Base:         s.buildRefPointer("file:///myrepo", "referenceName", baseHash),
		WantContents: true,
	}

	s.T().Run("without base", func(t *testing.T) {
		require := require.New(t)
		changes := s.requireScannerChanges(changesRequestNoBase, 9)
		require.Equal(`*.class

# Mobile Tools for Java (J2ME)
.mtj.tmp/

# Package Files #
*.jar
*.war
*.ear

# virtual machine crash logs, see http://www.java.com/en/download/help/error_hotspot.xml
hs_err_pid*
`, string(changes[".gitignore"].Head.Content))
	})

	s.T().Run("with base", func(t *testing.T) {
		require := require.New(t)
		changes := s.requireScannerChanges(changesRequestWithBase, 5)
		require.Nil(changes[".gitignore"])
	})
}

func (s *ServiceGetChangesSuite) TestWithPatternAndContent() {
	headHash, baseHash := s.getHeadBaseFixtureHash()

	s.T().Run("without base - with both include and exclude pattern", func(t *testing.T) {
		require := require.New(t)
		changesReq := &lookout.ChangesRequest{
			Head:           s.buildRefPointer("file:///myrepo", "referenceName", headHash),
			IncludePattern: `.gitignore`,
			ExcludePattern: `.*\.json`,
			WantContents:   true,
		}

		changes := s.requireScannerChanges(changesReq, 1)
		require.Equal(`*.class

# Mobile Tools for Java (J2ME)
.mtj.tmp/

# Package Files #
*.jar
*.war
*.ear

# virtual machine crash logs, see http://www.java.com/en/download/help/error_hotspot.xml
hs_err_pid*
`, string(changes[".gitignore"].Head.Content))
	})

	s.T().Run("with base - with both include and exclude pattern", func(t *testing.T) {
		require := require.New(t)
		changesReq := &lookout.ChangesRequest{
			Head:           s.buildRefPointer("file:///myrepo", "referenceName", headHash),
			Base:           s.buildRefPointer("file:///myrepo", "referenceName", baseHash),
			IncludePattern: `.gitignore`,
			ExcludePattern: `.*\.json`,
			WantContents:   true,
		}

		changes := s.requireScannerChanges(changesReq, 0)
		require.Nil(changes[".gitignore"])
	})
}

type ServiceGetFilesSuite struct {
	ServiceSuite
}

func (s *ServiceGetFilesSuite) requireScannerFiles(req *lookout.FilesRequest, expectedFiles int) map[string]*lookout.File {
	s.T().Helper()
	require := s.Require()

	srv := NewService(&StorerCommitLoader{s.Storer})
	scanner, err := srv.GetFiles(context.TODO(), req)

	require.NoError(err)
	require.NotNil(scanner)

	files := make(map[string]*lookout.File)
	for scanner.Next() {
		f := scanner.File()
		files[f.Path] = f
	}

	require.False(scanner.Next())
	require.NoError(scanner.Err())
	require.NoError(scanner.Close())

	require.Len(files, expectedFiles)

	return files
}

func (s *ServiceGetFilesSuite) TestInvalidReference() {
	srv := NewService(&StorerCommitLoader{s.Storer})

	headHash := s.Basic.Head.String()

	filesRequests := [3]*lookout.FilesRequest{
		&lookout.FilesRequest{Revision: &lookout.ReferencePointer{}},
		&lookout.FilesRequest{Revision: s.buildRefPointer("file:///myrepo", "referenceName", "")},
		&lookout.FilesRequest{Revision: s.buildRefPointer("", "referenceName", headHash)},
	}

	testNames := [3]string{
		"empty revision",
		"empty string in hash",
		"empty string in internal repo url",
	}

	for i, filesReq := range filesRequests {
		s.T().Run(testNames[i], func(t *testing.T) {
			require := require.New(t)
			_, err := srv.GetFiles(context.TODO(), filesReq)

			require.Error(err)
			require.True(ErrRefValidation.Is(err))
		})
	}
}

func (s *ServiceGetFilesSuite) TestInvalidTree() {
	require := s.Require()

	headHash := s.Basic.Head.String()

	revision := s.buildRefPointer("file:///myrepo", "referenceName", headHash)
	rps := []lookout.ReferencePointer{*revision}

	m := &MockCommitLoader{}
	m.On("LoadCommits", mock.Anything, rps).Once().Return(nil, fmt.Errorf("Error"))

	srv := NewService(m)

	scanner, err := srv.GetFiles(context.TODO(), &lookout.FilesRequest{Revision: revision})

	require.Error(err)
	require.Nil(scanner)
}

func (s *ServiceGetFilesSuite) TestNoParams() {
	headHash := s.Basic.Head.String()

	filesRequest := &lookout.FilesRequest{Revision: s.buildRefPointer("file:///myrepo", "referenceName", headHash)}

	s.requireScannerFiles(filesRequest, 9)
}

func (s *ServiceGetFilesSuite) TestWithPattern() {
	headHash := s.Basic.Head.String()

	fixtures := []filterScannerFixture{
		{IncludePattern: `.*\.go`},
		{ExcludePattern: `.*\.php`},
		{IncludePattern: `.*\.jpg`, ExcludePattern: `.*\.json`},
	}
	testNames := [3]string{"with inlcude pattern", "with exclude pattern", "with both include and exclude pattern"}

	expectedFiles := [3]int{2, 8, 1}
	for i, fixture := range fixtures {
		s.T().Run(testNames[i], func(t *testing.T) {
			filesReq := &lookout.FilesRequest{
				Revision:       s.buildRefPointer("file:///myrepo", "referenceName", headHash),
				IncludePattern: fixture.IncludePattern,
				ExcludePattern: fixture.ExcludePattern,
			}

			s.requireScannerFiles(filesReq, expectedFiles[i])
		})
	}
}

func (s *ServiceGetFilesSuite) TestWithContent() {
	require := s.Require()

	headHash := s.Basic.Head.String()

	filesRequest := &lookout.FilesRequest{
		Revision:     s.buildRefPointer("file:///myrepo", "referenceName", headHash),
		WantContents: true,
	}

	files := s.requireScannerFiles(filesRequest, 9)
	require.Equal(`*.class

# Mobile Tools for Java (J2ME)
.mtj.tmp/

# Package Files #
*.jar
*.war
*.ear

# virtual machine crash logs, see http://www.java.com/en/download/help/error_hotspot.xml
hs_err_pid*
`, string(files[".gitignore"].Content))
}

func (s *ServiceGetFilesSuite) TestWithPatternAndContent() {
	require := s.Require()

	headHash := s.Basic.Head.String()

	filesRequest := &lookout.FilesRequest{
		Revision:       s.buildRefPointer("file:///myrepo", "referenceName", headHash),
		IncludePattern: `.gitignore`,
		ExcludePattern: `.*\.json`,
		WantContents:   true,
	}

	files := s.requireScannerFiles(filesRequest, 1)
	require.Equal(`*.class

# Mobile Tools for Java (J2ME)
.mtj.tmp/

# Package Files #
*.jar
*.war
*.ear

# virtual machine crash logs, see http://www.java.com/en/download/help/error_hotspot.xml
hs_err_pid*
`, string(files[".gitignore"].Content))
}
