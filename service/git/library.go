package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

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

	return filesystem.NewStorage(fs)
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

	if err := removeAll(l.fs, l.repositoryPath(url)); err != nil {
		return nil, err
	}

	return l.init(url)
}

// billy.Filesystem doesn't provide RemoveAll methods as go std lib
// copy-past from std lib but using billy methods
func removeAll(fs billy.Filesystem, path string) error {
	// Simple case: if Remove works, we're done.
	err := fs.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	// Otherwise, is this a directory we need to recurse into?
	dir, serr := fs.Lstat(path)
	if serr != nil {
		if serr, ok := serr.(*os.PathError); ok && (os.IsNotExist(serr.Err) || serr.Err == syscall.ENOTDIR) {
			return nil
		}
		return serr
	}
	if !dir.IsDir() {
		// Not a directory; return the error from Remove.
		return err
	}

	// Remove contents & return first error.
	err = nil
	for {
		fd, err := fs.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Already deleted by someone else.
				return nil
			}
			return err
		}

		const request = 1024
		names, err1 := readdirnames(fs, fd, request)

		// Removing files from the directory may have caused
		// the OS to reshuffle it. Simply calling Readdirnames
		// again may skip some entries. The only reliable way
		// to avoid this is to close and re-open the
		// directory. See issue 20841.
		fd.Close()

		for _, name := range names {
			err1 := removeAll(fs, path+string(os.PathSeparator)+name)
			if err == nil {
				err = err1
			}
		}

		if err1 == io.EOF {
			break
		}
		// If Readdirnames returned an error, use it.
		if err == nil {
			err = err1
		}
		if len(names) == 0 {
			break
		}

		// We don't want to re-open unnecessarily, so if we
		// got fewer than request names from Readdirnames, try
		// simply removing the directory now. If that
		// succeeds, we are done.
		if len(names) < request {
			err1 := fs.Remove(path)
			if err1 == nil || os.IsNotExist(err1) {
				return nil
			}

			if err != nil {
				// We got some error removing the
				// directory contents, and since we
				// read fewer names than we requested
				// there probably aren't more files to
				// remove. Don't loop around to read
				// the directory again. We'll probably
				// just get the same error.
				return err
			}
		}
	}

	// Remove directory.
	err1 := fs.Remove(path)
	if err1 == nil || os.IsNotExist(err1) {
		return nil
	}
	if err == nil {
		err = err1
	}
	return err
}

// billy.File doesn't support Readdirnames method
func readdirnames(fs billy.Filesystem, fd billy.File, n int) ([]string, error) {
	content, err := fs.ReadDir(fd.Name())
	if err != nil {
		return nil, err
	}

	if len(content) > n {
		content = content[:n]
	}

	result := make([]string, len(content))
	for i, fi := range content {
		result[i] = fi.Name()
	}

	return result, nil
}
