package lookout

import (
	"gopkg.in/src-d/lookout-sdk.v0/pb"
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
	// Validate returns an error if the event is malformed
	Validate() error
	// GetProvider returns the name of the provider that created this event
	GetProvider() string
	// GetOrganizationID returns the organization to which this event's repository
	// belongs to
	GetOrganizationID() string
}

// EventID is a unique hash id for an event
type EventID = pb.EventID

// EventType defines the supported event types
type EventType = pb.EventType

// CommitRevision defines a range of commits, from a base to a head
type CommitRevision = pb.CommitRevision

// RepositoryInfo contains information about a repository
type RepositoryInfo = pb.RepositoryInfo

// ReferencePointer is a pointer to a git refererence in a repository
type ReferencePointer = pb.ReferencePointer

// PushEvent represents a Push to a git repository. It wraps the pb.PushEvent
// adding information only relevant to lookout, and not for the analyzers.
type PushEvent struct {
	pb.PushEvent
	// OrganizationID is the organization to which this event's repository
	// belongs to
	OrganizationID string
}

// GetProvider returns the name of the provider that created this event
func (e *PushEvent) GetProvider() string {
	if e == nil {
		return ""
	}

	return e.Provider
}

// GetOrganizationID returns the organization to which this event's repository
// belongs to
func (e *PushEvent) GetOrganizationID() string {
	if e == nil {
		return ""
	}

	return e.OrganizationID
}

// ReviewEvent represents a Review (pull request in case of GitHub) being
// created or updated. It wraps the pb.PushEvent adding information only
// relevant to lookout, and not for the analyzers.
type ReviewEvent struct {
	pb.ReviewEvent
	// OrganizationID is the organization to which this event's repository
	// belongs to
	OrganizationID string
}

// GetProvider returns the name of the provider that created this event
func (e *ReviewEvent) GetProvider() string {
	if e == nil {
		return ""
	}

	return e.Provider
}

// GetOrganizationID returns the organization to which this event's repository
// belongs to
func (e *ReviewEvent) GetOrganizationID() string {
	if e == nil {
		return ""
	}

	return e.OrganizationID
}
