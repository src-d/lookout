// +build integration

package server_test

import (
	"testing"
	"time"

	fixtures "github.com/src-d/lookout-test-fixtures"
	"gopkg.in/src-d/lookout-sdk.v0/pb"

	"github.com/stretchr/testify/suite"
)

func TestReviewOnlyPrChangesIntegrationSuite(t *testing.T) {
	suite.Run(t, new(reviewOnlyPrChangesIntegrationSuite))
}

type reviewOnlyPrChangesIntegrationSuite struct {
	IntegrationSuite
}

func (suite *reviewOnlyPrChangesIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)

	suite.StartDummy("--files")
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy`)
}

func (suite *reviewOnlyPrChangesIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *reviewOnlyPrChangesIntegrationSuite) TestAnalyzerErr() {
	fixtures := fixtures.GetByName("get-changes-from-outdated-pr")
	jsonReviewEvent := &jsonReviewEvent{
		ReviewEvent: &pb.ReviewEvent{
			InternalID:     "1",
			Number:         1,
			CommitRevision: *fixtures.GetCommitRevision(),
		},
	}

	expectedComments := []string{
		`{"analyzer-name":"Dummy","file":"javascript.js",`,
		`status=success`,
	}
	notExpectedComments := []string{
		`{"analyzer-name":"Dummy","file":"golang.go",`,
	}

	suite.sendEvent(jsonReviewEvent.String())
	suite.GrepAndNotAll(suite.r, expectedComments, notExpectedComments)
}

