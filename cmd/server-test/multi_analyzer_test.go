// +build integration

package server_test

import (
	"fmt"
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
	suite.GrepTrue(suite.r, "processing pull request")
	suite.GrepTrue(suite.r, "posting analysis")
	found, buf := suite.Grep(suite.r, `status=success`)
	suite.Require().Truef(found, "'%s' not found in:\n%s", `status=success`, buf.String())

	st := buf.String()

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy1","file":"another.go","line":3,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the first analyzer from %s in '%s'", doubleDummyConfigFile, buf),
	)

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy2","file":"another.go","line":3,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the second analyzer from %s in '%s'", doubleDummyConfigFile, buf),
	)
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
