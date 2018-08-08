package git

import (
	"context"
	"fmt"
	"sync"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	log "gopkg.in/src-d/go-log.v1"
)

// Syncer syncs the local copy of git repository for a given CommitRevision.
type Syncer struct {
	m sync.Map // holds mutexes per repository

	l *Library
}

// NewSyncer returns a Syncer for the given Library.
func NewSyncer(l *Library) *Syncer {
	return &Syncer{l: l}
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

	return s.fetch(ctx, frp.InternalRepositoryURL, r, refspecs)
}

func (s *Syncer) fetch(ctx context.Context, repoURL string, r *git.Repository, refspecs []config.RefSpec) error {
	log.Infof("fetching references for repository %s: %v", repoURL, refspecs)

	opts := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   refspecs,
		Force:      true,
	}

	mi, _ := s.m.LoadOrStore(repoURL, &sync.Mutex{})
	mutex := mi.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	err := r.FetchContext(ctx, opts)
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}
