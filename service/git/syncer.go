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

// Sync syncs the local git repository to the given reference pointers.
func (s *Syncer) Sync(ctx context.Context,
	rps ...lookout.ReferencePointer) error {

	if len(rps) == 0 {
		return fmt.Errorf("at least one reference pointer is required")
	}

	frp := rps[0]
	for _, orp := range rps[1:] {
		if orp.InternalRepositoryURL != frp.InternalRepositoryURL {
			return fmt.Errorf(
				"sync from multiple repositories is not supported")
		}
	}

	r, err := s.l.GetOrInit(frp.Repository())
	if err != nil {
		return err
	}

	var refspecs []config.RefSpec
	for _, rp := range rps {
		rs := config.RefSpec(fmt.Sprintf("%s:%[1]s", rp.ReferenceName))
		refspecs = append(refspecs, rs)
	}

	opts := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   refspecs,
		Force:      true,
	}

	if err := r.FetchContext(ctx, opts); err != nil {
		return err
	}

	return nil
}
