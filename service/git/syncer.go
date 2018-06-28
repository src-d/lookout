package git

import (
	"context"
	"fmt"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
)

// Syncer syncs the local copy of git repository for a given CommitRevision.
type Syncer struct {
	l *Library
}

// NewSyncer returns a Syncer for the given Library.
func NewSyncer(l *Library) *Syncer {
	return &Syncer{l}
}

// Sync syncs the local git repository to the given commit revision.
func (s *Syncer) Sync(ctx context.Context, rev *lookout.CommitRevision) error {
	r, err := s.l.GetOrInit(rev.Head.Repository())
	if err != nil {
		return err
	}

	opts := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:%[1]s", rev.Head.ReferenceName)),
		},
		Force: true,
	}

	if err := r.FetchContext(ctx, opts); err != nil {
		return err
	}

	return err
}
