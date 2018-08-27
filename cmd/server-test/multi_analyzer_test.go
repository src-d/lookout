// +build integration

package server_test

import (
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

	"github.com/stretchr/testify/suite"
)

type MultiDummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *MultiDummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.StartDummy("--files")
	suite.StartDummy("--files", "--analyzer", "ipv4://localhost:10303")
	suite.r, suite.w = suite.StartServe("--provider", "json",
		"-c", "../../fixtures/double_dummy_config.yml", "dummy-repo-url")

	// make sure server started correctly
	cmdtest.GrepTrue(suite.r, "Starting watcher")
}

func (suite *MultiDummyIntegrationSuite) TearDownTest() {
	suite.Stop()
}

func (suite *MultiDummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)
	cmdtest.GrepTrue(suite.r, "processing pull request")
	cmdtest.GrepTrue(suite.r, "posting analysis")
	found, buf := cmdtest.Grep(suite.r, `status=success`)
	suite.Require().Truef(found, "'%s' not found in:\n%s", `status=success`, buf.String())

	st := buf.String()

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy1","file":"cmd/lookout/push.go","line":13,"text":"This line exceeded 80 bytes."}`,
		"no comments from the first analyzer")

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy2","file":"cmd/lookout/push.go","line":13,"text":"This line exceeded 80 bytes."}`,
		"no comments from the second analyzer")
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
