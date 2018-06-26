package lookout

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

var (
	ErrRepositoryExists    = errors.NewKind("repository %s already exists")
	ErrRepositoryNotExists = errors.NewKind("repository %s not exists")
)

// Library controls the persistence of multiple git repositories.
type Library struct {
	fs billy.Filesystem
}

// NewLibrary creates a new Library based on the given filesystem.
func NewLibrary(fs billy.Filesystem) *Library {
	return &Library{fs: fs}
}

// GetOrInit get the requested repository based on the given URL, or inits a
// new repository.
func (l *Library) GetOrInit(url *RepositoryInfo) (*git.Repository, error) {
	has, err := l.Has(url)
	if err != nil {
		return nil, err
	}

	if has {
		return l.Get(url)
	}

	return l.Init(url)
}

// Init inits a new repository for the given URL.
func (l *Library) Init(url *RepositoryInfo) (*git.Repository, error) {
	has, err := l.Has(url)
	if err != nil {
		return nil, err
	}

	if has {
		return nil, ErrRepositoryExists.New(url.CloneURL)
	}

	s, err := l.repositoryStorer(url)
	if err != nil {
		return nil, err
	}

	r, err := git.Init(s, nil)
	if err != nil {
		return nil, err
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{url.CloneURL},
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}

// Has returns true if a repository with the given URL exists.
func (l *Library) Has(url *RepositoryInfo) (bool, error) {
	_, err := l.fs.Stat(l.repositoryPath(url))
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// Get get the requested repository based on the given URL.
func (l *Library) Get(url *RepositoryInfo) (*git.Repository, error) {
	has, err := l.Has(url)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrRepositoryNotExists.New(url.CloneURL)
	}

	s, err := l.repositoryStorer(url)
	if err != nil {
		return nil, err
	}

	return git.Open(s, nil)
}

func (l *Library) repositoryStorer(url *RepositoryInfo) (storage.Storer, error) {
	fs, err := l.fs.Chroot(l.repositoryPath(url))
	if err != nil {
		return nil, err
	}

	return filesystem.NewStorage(fs)
}

func (l *Library) repositoryPath(url *RepositoryInfo) string {
	return fmt.Sprintf("%s/%s", url.RepoHost, url.FullName)
}
