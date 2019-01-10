package lookout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalysisStatusStringer(t *testing.T) {
	// make sure we didn't messup with the names
	assert.Equal(t, "error", ErrorAnalysisStatus.String())
	assert.Equal(t, "failure", FailureAnalysisStatus.String())
	assert.Equal(t, "pending", PendingAnalysisStatus.String())
	assert.Equal(t, "success", SuccessAnalysisStatus.String())

	// unknown statuses
	assert.Equal(t, "unknown", AnalysisStatus(0).String())
	assert.Equal(t, "unknown", AnalysisStatus(100).String())
}
