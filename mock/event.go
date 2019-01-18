package mock

import (
	"github.com/src-d/lookout"
)

// FakeEvent is just an invalid event to be used for testing purposes.
type FakeEvent struct{}

// ID honors the Event interface.
func (e *FakeEvent) ID() lookout.EventID {
	var id [20]byte
	return id
}

// Type honors the Event interface.
func (e *FakeEvent) Type() lookout.EventType {
	return 100
}

// Revision honors the Event interface.
func (e *FakeEvent) Revision() *lookout.CommitRevision {
	return &lookout.CommitRevision{}
}

// Validate honors the Event interface.
func (e *FakeEvent) Validate() error {
	return nil
}
