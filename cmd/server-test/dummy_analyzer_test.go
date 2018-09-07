// +build integration

package server_test

import (
	"testing"

	"github.com/src-d/lookout"
	"github.com/stretchr/testify/suite"
)

const dummyConfigFile = "../../fixtures/dummy_config.yml"

type DummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *DummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.StartDummy("--files")
	suite.r, suite.w = suite.StartServe("--provider", "json",
		"-c", dummyConfigFile)

	// make sure server started correctly
	suite.GrepTrue(suite.r, "Starting watcher")
}

func (suite *DummyIntegrationSuite) TearDownTest() {
	suite.Stop()
}

func (suite *DummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, "processing pull request")
	suite.GrepTrue(suite.r, `{"analyzer-name":"Dummy","file":"another.go","line":3,"text":"This line exceeded`)
	suite.GrepTrue(suite.r, `status=success`)
}

func (suite *DummyIntegrationSuite) TestSkipReview() {
	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, `status=success`)

	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, `event successfully processed, skipping...`)
}

func (suite *DummyIntegrationSuite) TestReviewDontPost() {
	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, `status=success`)

	newEventForSuccessPREvent := &jsonReviewEvent{
		ReviewEvent: &lookout.ReviewEvent{
			InternalID:     "2",
			Number:         1,
			CommitRevision: longLineFixture.CommitRevision,
		},
	}

	suite.sendEvent(newEventForSuccessPREvent.String())
	suite.GrepTrue(suite.r, "processing pull request")
	suite.GrepAndNot(suite.r, `status=success`, `posting analysis`)
}

func (suite *DummyIntegrationSuite) TestWrongRevision() {
	e := &jsonReviewEvent{
		ReviewEvent: &lookout.ReviewEvent{
			InternalID:     "3",
			Number:         3,
			CommitRevision: longLineFixture.CommitRevision,
		},
	}
	// change hashes to incorrect ones
	e.CommitRevision.Base.Hash = "0000000000000000000000000000000000000000"
	e.CommitRevision.Head.Hash = "0000000000000000000000000000000000000000"

	suite.sendEvent(e.String())
	suite.GrepTrue(suite.r, `event processing failed`)
}

func (suite *DummyIntegrationSuite) TestSuccessPush() {
	pushEvent := jsonPushEvent{
		PushEvent: &lookout.PushEvent{
			InternalID:     "1",
			CommitRevision: longLineFixture.CommitRevision,
		},
	}
	suite.sendEvent(pushEvent.String())
	suite.GrepTrue(suite.r, "processing push")
	suite.GrepTrue(suite.r, "comments can belong only to review event but 1 is given")
	suite.GrepTrue(suite.r, `status=success`)
}

func TestDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DummyIntegrationSuite))
}
