package git

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	log "gopkg.in/src-d/go-log.v1"
)

const defaultRemoteName = "origin"

// Syncer syncs the local copy of git repository for a given CommitRevision.
type Syncer struct {
	m sync.Map // holds mutexes per repository

	l *Library

	authProvider AuthProvider
	// fetchTimeout of zero means no timeout.
	fetchTimeout time.Duration
}

// AuthProvider is an object that provides go-git auth methods
type AuthProvider interface {
	// GitAuth returns a go-git auth method for a repo
	GitAuth(ctx context.Context, repoInfo *lookout.RepositoryInfo) transport.AuthMethod
}

// NewSyncer returns a Syncer for the given Library. authProvider can be nil.
// A fetchTimeout of zero means no timeout.
func NewSyncer(l *Library, authProvider AuthProvider, fetchTimeout time.Duration) *Syncer {
	return &Syncer{l: l, authProvider: authProvider, fetchTimeout: fetchTimeout}
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

	repoInfo := frp.Repository()
	gitRepo, err := s.l.GetOrInit(ctx, frp.Repository())
	if err != nil {
		return err
	}

	var refspecs []config.RefSpec
	for _, rp := range rps {
		var rs config.RefSpec
		if "" == rp.ReferenceName {
			rs = config.RefSpec(fmt.Sprintf(config.DefaultFetchRefSpec, defaultRemoteName))
			ctxlog.Get(ctx).Warningf("empty ReferenceName given in %v, using default '%s' instead", rp, rs)
		} else {
			rs = config.RefSpec(fmt.Sprintf("%s:%[1]s", rp.ReferenceName))
		}
		refspecs = append(refspecs, rs)
	}

	return s.fetch(ctx, repoInfo, gitRepo, refspecs)
}

func (s *Syncer) fetch(ctx context.Context, repoInfo *lookout.RepositoryInfo, r *git.Repository, refspecs []config.RefSpec) (err error) {
	ctxlog.Get(ctx).Infof("fetching references for repository %s: %v", repoInfo.CloneURL, refspecs)
	start := time.Now()
	defer func() {
		if err == nil {
			ctxlog.Get(ctx).
				With(log.Fields{"duration": time.Now().Sub(start)}).
				Debugf("references %v fetched for repository %s", refspecs, repoInfo.CloneURL)
		}
		// in case of error it will be logged on upper level
	}()

	var auth transport.AuthMethod
	if s.authProvider != nil {
		auth = s.authProvider.GitAuth(ctx, repoInfo)
	}

	opts := &git.FetchOptions{
		RemoteName: defaultRemoteName,
		RefSpecs:   refspecs,
		Force:      true,
		Auth:       auth,
	}

	mi, _ := s.m.LoadOrStore(repoInfo.CloneURL, &sync.Mutex{})
	mutex := mi.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	if s.fetchTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.fetchTimeout)
		defer cancel()
	}
	err = r.FetchContext(ctx, opts)
	switch err {
	case git.NoErrAlreadyUpToDate:
		err = nil
	case transport.ErrInvalidAuthMethod:
		err = fmt.Errorf("wrong go-git authentication method: %s", err.Error())
	}

	return err
}
