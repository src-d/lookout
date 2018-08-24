// +build integration

package sdk_test

import (
	"context"
	"io"
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

	"github.com/stretchr/testify/suite"
)

type BblfshIntegrationSuite struct {
	suite.Suite
	ctx  context.Context
	stop func()
}

func (suite *BblfshIntegrationSuite) SetupSuite() {
	suite.ctx, suite.stop = cmdtest.StoppableCtx()
	cmdtest.StartDummy(suite.ctx, "--uast", "--files")
}

func (suite *BblfshIntegrationSuite) TearDownSuite() {
	suite.stop()
}

func (suite *BblfshIntegrationSuite) RunReview() io.Reader {
	return cmdtest.RunCli(suite.ctx, "review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
}

func (suite *BblfshIntegrationSuite) RunPush() io.Reader {
	return cmdtest.RunCli(suite.ctx, "push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
}

func (suite *BblfshIntegrationSuite) TestReviewNoBblfshError() {
	r := cmdtest.RunCli(suite.ctx, "review", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestReviewNoUASTWarning() {
	r := suite.RunReview()
	cmdtest.GrepTrue(r, "The file doesn't have UAST")
}

func (suite *BblfshIntegrationSuite) TestReviewUAST() {
	r := suite.RunReview()
	cmdtest.GrepTrue(r, "The file has UAST.")
}

func (suite *BblfshIntegrationSuite) TestReviewLanguage() {
	r := suite.RunReview()
	cmdtest.GrepTrue(r, `The file has language detected: "Go"`)
}

func (suite *BblfshIntegrationSuite) TestPushNoBblfshError() {
	r := cmdtest.RunCli(suite.ctx, "push", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestPushNoUASTWarning() {
	r := suite.RunPush()
	cmdtest.GrepTrue(r, "The file doesn't have UAST")
}

func (suite *BblfshIntegrationSuite) TestPushUAST() {
	r := suite.RunPush()
	cmdtest.GrepTrue(r, "The file has UAST.")
}

func (suite *BblfshIntegrationSuite) TestPushLanguage() {
	r := suite.RunPush()
	cmdtest.GrepTrue(r, `The file has language detected: "Go"`)
}

func TestBblfshIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test suite in short mode.")
	}

	suite.Run(t, new(BblfshIntegrationSuite))
}
