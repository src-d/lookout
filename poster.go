package lookout

import "context"

// AnalysisStatus is the status reported to the provider to
// inform that we are performing an analysis, or that it has finished
type AnalysisStatus int

const (
	_ AnalysisStatus = iota
	// ErrorAnalysisStatus represents an error status
	ErrorAnalysisStatus
	// FailureAnalysisStatus represents a failure status
	FailureAnalysisStatus
	// PendingAnalysisStatus represents a pending status
	PendingAnalysisStatus
	// SuccessAnalysisStatus represents a success status
	SuccessAnalysisStatus
)

func (st AnalysisStatus) String() string {
	names := [...]string{"unknown", "error", "failure", "pending", "success"}
	if st < ErrorAnalysisStatus || st > SuccessAnalysisStatus {
		return names[0]
	}

	return names[st]
}

// Poster can post comments about an event.
type Poster interface {
	// Post posts comments about an event.
	Post(context.Context, Event, []AnalyzerComments) error

	// Status sends the current analysis status to the provider
	Status(context.Context, Event, AnalysisStatus) error
}
