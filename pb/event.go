package pb

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"

	"gopkg.in/sourcegraph/go-vcsurl.v1"
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
	ReviewEventType
)

// ID honors the Event interface.
func (e *ReviewEvent) ID() EventID {
	return ComputeEventID(e.Provider, e.InternalID)
}

// Type honors the Event interface.
func (e *ReviewEvent) Type() EventType {
	return ReviewEventType
}

// Revision honors the Event interface.
func (e *ReviewEvent) Revision() *CommitRevision {
	return &e.CommitRevision
}

// Validate honors the Event interface.
func (e *ReviewEvent) Validate() error {
	var zeroVal ReviewEvent
	if reflect.DeepEqual(*e, zeroVal) {
		return errors.New("this ReviewEvent event is empty")
	}

	return nil
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

// Validate honors the Event interface.
func (e *PushEvent) Validate() error {
	var zeroVal PushEvent
	if reflect.DeepEqual(*e, zeroVal) {
		return errors.New("this PushEvent event is empty")
	}

	return nil
}

type RepositoryInfo = vcsurl.RepoInfo //TODO(mcuadros): improve repository references

// Repository returns the RepositoryInfo
func (e *ReferencePointer) Repository() *RepositoryInfo {
	info, _ := vcsurl.Parse(e.InternalRepositoryURL)
	return info
}

// Short is a short string representation of a ReferencePointer.
func (e *ReferencePointer) Short() string {
	return fmt.Sprintf(
		"%s@%s",
		e.ReferenceName.Short(),
		e.Hash[0:6],
	)
}
