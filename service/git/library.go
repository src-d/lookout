package git

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

var (
	ErrRepositoryExists    = errors.NewKind("repository %s already exists")
	ErrRepositoryNotExists = errors.NewKind("repository %s not exists")
)

// Library controls the persistence of multiple git repositories.
type Library struct {
	m  sync.Mutex
	fs billy.Filesystem
}

// NewLibrary creates a new Library based on the given filesystem.
func NewLibrary(fs billy.Filesystem) *Library {
	return &Library{fs: fs}
}

// GetOrInit get the requested repository based on the given URL, or inits a
// new repository.
func (l *Library) GetOrInit(ctx context.Context, url *lookout.RepositoryInfo) (
	*git.Repository, error) {
	has, err := l.Has(url)
	if err != nil {
		return nil, err
	}

	if has {
		return l.Get(ctx, url)
	}

	return l.Init(ctx, url)
}

// Init inits a new repository for the given URL.
func (l *Library) Init(ctx context.Context, url *lookout.RepositoryInfo) (*git.Repository, error) {
	ctxlog.Get(ctx).Infof("creating local repository for: %s", url.CloneURL)
	l.m.Lock()
	defer l.m.Unlock()

	return l.init(url)
}

func (l *Library) init(url *lookout.RepositoryInfo) (*git.Repository, error) {
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
func (l *Library) Has(url *lookout.RepositoryInfo) (bool, error) {
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
func (l *Library) Get(ctx context.Context, url *lookout.RepositoryInfo) (*git.Repository, error) {
	r, err := l.get(url)

	// it can happen if the repository in a broken state
	if err == git.ErrRepositoryNotExists {
		return l.recreate(url)
	}

	return r, nil
}

func (l *Library) get(url *lookout.RepositoryInfo) (*git.Repository, error) {
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

func (l *Library) repositoryStorer(url *lookout.RepositoryInfo) (
	storage.Storer, error) {
	fs, err := l.fs.Chroot(l.repositoryPath(url))
	if err != nil {
		return nil, err
	}

	// TODO(carlosms) take cache size from config
	cache := cache.NewObjectLRU(cache.DefaultMaxSize)
	return filesystem.NewStorage(fs, cache), nil
}

func (l *Library) repositoryPath(url *lookout.RepositoryInfo) string {
	return fmt.Sprintf("%s/%s", url.RepoHost, url.FullName)
}

func (l *Library) recreate(url *lookout.RepositoryInfo) (*git.Repository, error) {
	l.m.Lock()
	defer l.m.Unlock()

	// in case it was recreated already by another goroutine
	r, err := l.get(url)
	if err != git.ErrRepositoryNotExists {
		return r, err
	}

	if err := util.RemoveAll(l.fs, l.repositoryPath(url)); err != nil {
		return nil, err
	}

	return l.init(url)
}
