// +build integration

package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

const dubleDummyConfigFile = "../../fixtures/double_dummy_config.yml"

type MultiDummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *MultiDummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.StartDummy("--files")
	suite.StartDummy("--files", "--analyzer", "ipv4://localhost:10303")
	suite.r, suite.w = suite.StartServe("--provider", "json",
		"-c", dubleDummyConfigFile)

	// make sure server started correctly
	suite.GrepTrue(suite.r, "Starting watcher")
}

func (suite *MultiDummyIntegrationSuite) TearDownTest() {
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
		`{"analyzer-name":"Dummy1","file":"cmd/lookout/serve.go","line":33,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the first analyzer from %s in '%s'", dubleDummyConfigFile, buf),
	)

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy2","file":"cmd/lookout/serve.go","line":33,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the second analyzer from %s in '%s'", dubleDummyConfigFile, buf),
	)
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
