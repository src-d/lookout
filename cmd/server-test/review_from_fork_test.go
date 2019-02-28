// +build integration

package server_test

import (
	"testing"
	"time"

	"github.com/src-d/lookout-test-fixtures"
	"github.com/stretchr/testify/suite"
)

func TestReviewPrFromForkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(reviewPrFromForkIntegrationSuite))
}

type reviewPrFromForkIntegrationSuite struct {
	IntegrationSuite
}

func (suite *reviewPrFromForkIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)

	suite.StartDummy("--files")
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy`)
}

func (suite *reviewPrFromForkIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *reviewPrFromForkIntegrationSuite) TestReview() {
	fixture := fixtures.GetByName("pr-from-fork")
	jsonReviewEvent, err := castPullRequest(fixture)
	suite.NoError(err)

	expectedComments := []string{
		`{"analyzer-name":"Dummy","file":"javascript.js",`,
	}

	suite.sendEvent(jsonReviewEvent.String())
	suite.GrepAll(suite.r, expectedComments)
}
