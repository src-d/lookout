// +build integration

package sdk_test

import (
	"context"
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
	cmdtest.StartDummy(suite.ctx, "--uast")
}

func (suite *BblfshIntegrationSuite) TearDownSuite() {
	suite.stop()
}

func (suite *BblfshIntegrationSuite) TestNoBblfshError() {
	r := cmdtest.RunCli(suite.ctx, "review", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestNoUASTWarning() {
	r := cmdtest.RunCli(suite.ctx, "review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "The file doesn't have UAST")
}

func TestBblfshIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test suite in short mode.")
	}

	suite.Run(t, new(BblfshIntegrationSuite))
}
