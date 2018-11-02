// +build integration

package server_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const doubleDummyConfigFile = "../../fixtures/double_dummy_config.yml"

type MultiDummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *MultiDummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(doubleDummyConfigFile)

	suite.StartDummy("--files")
	suite.GrepTrue(suite.r, `connection state changed to 'READY'`)

	suite.StartDummy("--files", "--analyzer", "ipv4://localhost:10303")
	suite.GrepTrue(suite.r, `connection state changed to 'READY'`)
}

func (suite *MultiDummyIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *MultiDummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)

	suite.GrepAll(suite.r, []string{
		"processing pull request",
		"posting analysis",
		`{"analyzer-name":"Dummy1","file":"another.go","line":3,"text":"This line exceeded`,
		`{"analyzer-name":"Dummy2","file":"another.go","line":3,"text":"This line exceeded`,
		`status=success`,
	})
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
