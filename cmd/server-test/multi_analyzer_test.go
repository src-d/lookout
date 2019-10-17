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
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy1`)

	suite.StartDummy("--files", "--analyzer", "ipv4://localhost:10303")
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:10303" analyzer=Dummy2`)
}

func (suite *MultiDummyIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *MultiDummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)
	// we don't know when server finished posting comments anymore
	// let's read everything from the output during some time and hope it finished
	time.Sleep(5 * time.Second)
	str := suite.AllOutput()

	suite.Require().Contains(str, "processing pull request")
	suite.Require().Contains(str, `msg="posting analysis" app=lookoutd comments=4`)
	suite.Require().Contains(str, `status=success`)

	dummy1First := `{"analyzer-name":"Dummy1","file":"another\.go","text":"The file has increased in 4 lines\."}.*` +
		`{"analyzer-name":"Dummy1","file":"another\.go","line":3,"text":"This line exceeded 120 chars\."}.*` +
		`{"analyzer-name":"Dummy2","file":"another\.go","text":"The file has increased in 4 lines\."}.*` +
		`{"analyzer-name":"Dummy2","file":"another\.go","line":3,"text":"This line exceeded 120 chars\."}`
	dummy2First := `{"analyzer-name":"Dummy2","file":"another\.go","text":"The file has increased in 4 lines\."}.*` +
		`{"analyzer-name":"Dummy2","file":"another\.go","line":3,"text":"This line exceeded 120 chars\."}.*` +
		`{"analyzer-name":"Dummy1","file":"another\.go","text":"The file has increased in 4 lines\."}.*` +
		`{"analyzer-name":"Dummy1","file":"another\.go","line":3,"text":"This line exceeded 120 chars\."}`
	expr := `(?s)(` + dummy1First + `|` + dummy2First + `)`

	found := suite.EgrepWholeFromString(str, expr)
	suite.Require().True(found)
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
