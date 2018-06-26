package lookout

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type EventID [20]byte

// ComputeEventID compute the hash for a given provider and content.
func ComputeEventID(provider, content string) EventID {
	var id EventID
	h := sha1.New()
	h.Write([]byte(provider))
	h.Write([]byte("|"))
	h.Write([]byte(content))
	copy(id[:], h.Sum(nil))
	return id
}

func (h EventID) IsZero() bool {
	var empty EventID
	return h == empty
}

func (h EventID) String() string {
	return hex.EncodeToString(h[:])
}

type EventType int

const (
	_ EventType = iota
	PushEventType
	PullRequestEventType
)

// Event represents a repository event.
type Event interface {
	// ID returns the EventID.
	ID() EventID
	// Type returns the EventType, in order to identify the concreate type of
	// the event.
	Type() EventType
	// Revision returns a commit revision, containing the head and the base of
	// the changes.
	Revision() *CommitRevision
}

// CommitRevision defines a range of commits, from a base to a head.
type CommitRevision struct {
	// Base of the revision.
	Base ReferencePointer
	// Head of the revision.
	Head ReferencePointer
}

func (r CommitRevision) String() string {
	if r.Base.Repository.CloneURL == r.Head.Repository.CloneURL {
		return fmt.Sprintf("%s..%s", r.Head.Short(), r.Base.Short())
	}

	return fmt.Sprintf("%s..%s", r.Head.String(), r.Base.String())
}

// PullRequestEvent represents a PullRequest being created or updated.
type PullRequestEvent struct {
	Provider   string
	InternalID string

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

	CommitRevision
}

// ID honors the Event interface.
func (e *PullRequestEvent) ID() EventID {
	return ComputeEventID(e.Provider, e.InternalID)
}

// Type honors the Event interface.
func (e *PullRequestEvent) Type() EventType {
	return PullRequestEventType
}

// Revision honors the Event interface.
func (e *PullRequestEvent) Revision() *CommitRevision {
	return &e.CommitRevision
}

func (e *PullRequestEvent) String() string {
	return fmt.Sprintf("[pull-request][%s] %s", e.ID(), e.CommitRevision)
}

type PushEvent struct {
	Provider   string
	InternalID string

	// CreateAt is the timestamp of the creation date of the push event.
	CreatedAt time.Time
	// Commits is the number of commits in the push.
	Commits int
	// Commits is the number of distinct commits in the push.
	DistinctCommits int

	CommitRevision
}

func (e *PushEvent) String() string {
	return fmt.Sprintf("[push][%s] %s", e.ID(), e.CommitRevision)

}

// ID honors the Event interface.
func (e *PushEvent) ID() EventID {
	return ComputeEventID(e.Provider, e.InternalID)
}

// Type honors the Event interface.
func (e *PushEvent) Type() EventType {
	return PushEventType
}

// Revision honors the Event interface.
func (e *PushEvent) Revision() *CommitRevision {
	return &e.CommitRevision
}

type RepositoryInfo = vcsurl.RepoInfo //TODO(mcuadros): improve repository references

type ReferencePointer struct {
	Repository *RepositoryInfo
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
