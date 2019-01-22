package git

import (
	"context"
	"fmt"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type CommitLoader interface {
	LoadCommits(context.Context, ...lookout.ReferencePointer) (
		[]*object.Commit, error)
}

type LibraryCommitLoader struct {
	Library ReposCollectionHandler
	Syncer  Syncer
}

func NewLibraryCommitLoader(l ReposCollectionHandler, s Syncer) *LibraryCommitLoader {
	return &LibraryCommitLoader{
		Library: l,
		Syncer:  s,
	}
}

func (l *LibraryCommitLoader) LoadCommits(
	ctx context.Context, rps ...lookout.ReferencePointer) (
	[]*object.Commit, error) {

	if len(rps) == 0 {
		return nil, nil
	}

	frp := rps[0]
	for _, orp := range rps[1:] {
		if orp.InternalRepositoryURL != frp.InternalRepositoryURL {
			return nil, fmt.Errorf(
				"loading commits from multiple repositories is not supported")
		}
	}

	if err := l.Syncer.Sync(ctx, rps...); err != nil {
		return nil, err
	}

	r, err := l.Library.GetOrInit(ctx, frp.Repository())
	if err != nil {
		return nil, err
	}

	storerCl := NewStorerCommitLoader(r.Storer)
	return storerCl.LoadCommits(ctx, rps...)
}

type StorerCommitLoader struct {
	Storer storer.Storer
}

func NewStorerCommitLoader(storer storer.Storer) *StorerCommitLoader {
	return &StorerCommitLoader{
		Storer: storer,
	}
}

func (l *StorerCommitLoader) LoadCommits(ctx context.Context,
	rps ...lookout.ReferencePointer) ([]*object.Commit, error) {

	commits := make([]*object.Commit, len(rps))
	for i, rp := range rps {
		commit, err := object.GetCommit(l.Storer, plumbing.NewHash(rp.Hash))
		if err != nil {
			return nil, err
		}

		commits[i] = commit
	}

	return commits, nil
}
