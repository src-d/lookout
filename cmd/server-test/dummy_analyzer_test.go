// +build integration

package server_test

import (
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

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
		"-c", dummyConfigFile, "dummy-repo-url")

	// make sure server started correctly
	cmdtest.GrepTrue(suite.r, "Starting watcher")
}

func (suite *DummyIntegrationSuite) TearDownTest() {
	suite.Stop()
}

const successJSON = `{"event":"review", "internal_id": "1", "number": 1, "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"66924f49aa9987273a137857c979ee5f0e709e30"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"2c9f56bcb55be47cf35d40d024ec755399f699c7"}}}`

func (suite *DummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)
	cmdtest.GrepTrue(suite.r, "processing pull request")
	cmdtest.GrepTrue(suite.r, `{"analyzer-name":"Dummy","file":"cmd/lookout/serve.go","line":33,"text":"This line exceeded`)
	cmdtest.GrepTrue(suite.r, `status=success`)
}

func (suite *DummyIntegrationSuite) TestSkipReview() {
	suite.sendEvent(successJSON)
	cmdtest.GrepTrue(suite.r, `status=success`)

	suite.sendEvent(successJSON)
	cmdtest.GrepTrue(suite.r, `event successfully processed, skipping...`)
}

func (suite *DummyIntegrationSuite) TestReviewDontPost() {
	suite.sendEvent(successJSON)
	cmdtest.GrepTrue(suite.r, `status=success`)

	json := `{"event":"review", "internal_id": "2", "number": 1, "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"66924f49aa9987273a137857c979ee5f0e709e30"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"2c9f56bcb55be47cf35d40d024ec755399f699c7"}}}`
	suite.sendEvent(json)
	cmdtest.GrepTrue(suite.r, "processing pull request")
	cmdtest.GrepAndNot(suite.r, `status=success`, `posting analysis`)
}

func (suite *DummyIntegrationSuite) TestWrongRevision() {
	json := `{"event":"review", "internal_id": "3", "number": 3, "commit_revision": {"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"0000000000000000000000000000000000000000"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"0000000000000000000000000000000000000000"}}}`
	suite.sendEvent(json)
	cmdtest.GrepTrue(suite.r, `event processing failed`)
}

func (suite *DummyIntegrationSuite) TestSuccessPush() {
	successPushJSON := `{"event":"push", "internal_id": "1", "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"66924f49aa9987273a137857c979ee5f0e709e30"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"2c9f56bcb55be47cf35d40d024ec755399f699c7"}}}`
	suite.sendEvent(successPushJSON)
	cmdtest.GrepTrue(suite.r, "processing push")
	cmdtest.GrepTrue(suite.r, "comments can belong only to review event but 1 is given")
	cmdtest.GrepTrue(suite.r, `status=success`)
}

func TestDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DummyIntegrationSuite))
}
