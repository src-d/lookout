// +build integration

package server_test

import (
	"testing"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"
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

	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)
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
	if suite.IsQueueTested() {
		suite.T().Skip("skipping test, with a queue the watcher will not enqueue repeated jobs")
	}

	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, `status=success`)

	suite.sendEvent(successJSON)
	suite.GrepTrue(suite.r, `event successfully processed, skipping...`)
}

func (suite *DummyIntegrationSuite) TestReviewDontPostSameComment() {
	fixture := fixtures.GetByName("incremental-pr")

	rev0Event := &jsonReviewEvent{
		ReviewEvent: &lookout.ReviewEvent{
			InternalID:     "some-id",
			Number:         1,
			CommitRevision: *fixture.GetRevision(0).GetCommitRevision(),
		},
	}

	suite.sendEvent(rev0Event.String())
	suite.GrepTrue(suite.r, `{"analyzer-name":"Dummy","file":"dummy.go","line":5,"text":"This line exceeded`)
	suite.GrepTrue(suite.r, `status=success`)

	rev1Event := &jsonReviewEvent{
		ReviewEvent: &lookout.ReviewEvent{
			InternalID:     "some-id",
			Number:         1,
			CommitRevision: *fixture.GetRevision(1).GetCommitRevision(),
		},
	}

	suite.sendEvent(rev1Event.String())
	suite.GrepTrue(suite.r, "processing pull request")

	found, buf := suite.Grep(suite.r, `status=success`)
	suite.Require().Truef(found, "'%s' not found in:\n%s", `status=success`, buf.String())

	st := buf.String()

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy","file":"dummy.go","text":"The file has increased`,
	)

	suite.Require().NotContains(
		st,
		`{"analyzer-name":"Dummy","file":"dummy.go","line":5,"text":"This line exceeded`,
	)
}

func (suite *DummyIntegrationSuite) TestWrongRevision() {
	e := &jsonReviewEvent{
		ReviewEvent: &lookout.ReviewEvent{
			InternalID:     "3",
			Number:         3,
			CommitRevision: *longLineFixture.GetCommitRevision(),
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
			CommitRevision: *longLineFixture.GetCommitRevision(),
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
