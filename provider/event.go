package provider

import (
	"fmt"
	"time"

	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Revision defines a range of commits, from a base to a head.
type Revision struct {
	// Base of the revision.
	Base ReferencePointer
	// Head of the revision.
	Head ReferencePointer
}

func (r Revision) String() string {
	if r.Base.Repository.CloneURL == r.Head.Repository.CloneURL {
		return fmt.Sprintf("%s..%s", r.Head.Short(), r.Base.Short())
	}

	return fmt.Sprintf("%s..%s", r.Head.String(), r.Base.String())
}

// PullRequestEvent represents a PullRequest being created or updated.
type PullRequestEvent struct {
	ID string //TODO: improve ID references
	// CreateAt is the timestamp of the creation date of the pull request.
	CreatedAt time.Time
	// UpdatedAt is the timestamp of the last modification of the pull request.
	UpdatedAt time.Time
	// IsMergeable, if the pull request is mergeable.
	IsMergeable bool
	// Source reference to the original branch and repository were the changes
	// came from.
	Source ReferencePointer
	// Merge  test branch where the PullRequest was merged.
	Merge ReferencePointer

	Revision
}

func (e *PullRequestEvent) String() string {
	return fmt.Sprintf("[pull-request] %s", e.Revision)
}

type PushEvent struct {
	ID string //TODO: improve ID references
	// CreateAt is the timestamp of the creation date of the push event.
	CreatedAt time.Time
	// Commits is the number of commits in the push.
	Commits int
	// Commits is the number of distinct commits in the push.
	DistinctCommits int

	Revision
}

func (e *PushEvent) String() string {
	return fmt.Sprintf("[push] %s", e.Revision)

}

type ReferencePointer struct {
	Repository *vcsurl.RepoInfo //TODO(mcuadros): improve repository references
	Reference  *plumbing.Reference
}

// Short is a short string representation of a ReferencePointer.
func (e *ReferencePointer) Short() string {
	return fmt.Sprintf(
		"%s@%s",
		e.Reference.Name().Short(),
		e.Reference.Hash().String()[0:6],
	)
}

func (e *ReferencePointer) String() string {
	return fmt.Sprintf(
		"%s/%s@%s",
		e.Repository.CloneURL, e.Reference.Name().Short(),
		e.Reference.Hash().String()[0:6],
	)
}
