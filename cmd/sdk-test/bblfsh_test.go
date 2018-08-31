// +build integration

package sdk_test

import (
	"io"
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

	"github.com/stretchr/testify/suite"
)

type BblfshIntegrationSuite struct {
	cmdtest.IntegrationSuite
}

func (suite *BblfshIntegrationSuite) SetupSuite() {
	suite.StoppableCtx()
	suite.StartDummy("--uast", "--files")
}

func (suite *BblfshIntegrationSuite) TearDownSuite() {
	suite.Stop()
}

func (suite *BblfshIntegrationSuite) RunReview() io.Reader {
	return suite.RunCli("review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
}

func (suite *BblfshIntegrationSuite) RunPush() io.Reader {
	return suite.RunCli("push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
}

func (suite *BblfshIntegrationSuite) TestReviewNoBblfshError() {
	r := suite.RunCli("review", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestReviewNoUASTWarning() {
	r := suite.RunReview()
	suite.GrepTrue(r, "The file doesn't have UAST")
}

func (suite *BblfshIntegrationSuite) TestReviewUAST() {
	r := suite.RunReview()
	suite.GrepTrue(r, "The file has UAST.")
}

func (suite *BblfshIntegrationSuite) TestReviewLanguage() {
	r := suite.RunReview()
	suite.GrepTrue(r, `The file has language detected: "Go"`)
}

func (suite *BblfshIntegrationSuite) TestPushNoBblfshError() {
	r := suite.RunCli("push", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestPushNoUASTWarning() {
	r := suite.RunPush()
	suite.GrepTrue(r, "The file doesn't have UAST")
}

func (suite *BblfshIntegrationSuite) TestPushUAST() {
	r := suite.RunPush()
	suite.GrepTrue(r, "The file has UAST.")
}

func (suite *BblfshIntegrationSuite) TestPushLanguage() {
	r := suite.RunPush()
	suite.GrepTrue(r, `The file has language detected: "Go"`)
}

func TestBblfshIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test suite in short mode.")
	}

	suite.Run(t, new(BblfshIntegrationSuite))
}
