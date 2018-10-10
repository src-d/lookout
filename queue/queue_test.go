package queue

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/src-d/lookout"
)

func TestQueueJobCreation(t *testing.T) {
	ev := lookout.ReviewEvent{
		Provider:   "github",
		InternalID: "1234",
		CommitRevision: lookout.CommitRevision{
			Base: lookout.ReferencePointer{
				InternalRepositoryURL: "file:///fixture/basic",
				ReferenceName:         "aName",
				Hash:                  "918c48b83bd081e863dbe1b80f8998f058cd8294",
			},
			Head: lookout.ReferencePointer{
				InternalRepositoryURL: "file:///fixture/basic",
				ReferenceName:         "aName",
				Hash:                  "918c48b83bd081e863dbe1b80f8998f058cd8294",
			},
		},
	}

	qJob, err := NewQueueJob(&ev)
	require.NoError(t, err)
	require.NotNil(t, qJob)

	qEv, err := qJob.Event()
	require.NoError(t, err)
	require.NotNil(t, qEv)
	require.EqualValues(t, &ev, qEv)
	require.EqualValues(t, ev.Type(), qJob.EventType)

	require.Nil(t, qJob.PushEvent)
}
