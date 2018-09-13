// +build integration

package server_test

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/src-d/lookout/util/cmdtest"
)

type IntegrationSuite struct {
	cmdtest.IntegrationSuite
	r io.Reader
	w io.WriteCloser
}

func (suite *IntegrationSuite) sendEvent(json string) {
	_, err := fmt.Fprintln(suite.w, json)
	suite.Require().NoError(err)
}

type jsonReviewEvent struct {
	Event string `json:"event"`
	*lookout.ReviewEvent
}

func (e *jsonReviewEvent) String() string {
	e.Event = "review"
	b, _ := json.Marshal(e)
	return string(b)
}

type jsonPushEvent struct {
	Event string `json:"event"`
	*lookout.PushEvent
}

func (e *jsonPushEvent) String() string {
	e.Event = "push"
	b, _ := json.Marshal(e)
	return string(b)
}

var longLineFixture = fixtures.GetByName("new-go-file-too-long-line")

var successEvent = &jsonReviewEvent{
	ReviewEvent: &lookout.ReviewEvent{
		InternalID:     "1",
		Number:         1,
		CommitRevision: *longLineFixture.GetCommitRevision(),
	},
}

var successJSON = successEvent.String()
