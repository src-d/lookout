// +build integration

package sdk_test

import (
	"context"
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

	"github.com/stretchr/testify/suite"
)

type IntegrationSuite struct {
	suite.Suite
	ctx  context.Context
	stop func()
}

func (suite *IntegrationSuite) SetupTest() {
	suite.ctx, suite.stop = cmdtest.StoppableCtx()
}

func (suite *IntegrationSuite) TearDownTest() {
	suite.stop()
}

func (suite *IntegrationSuite) TestReview() {
	cmdtest.StartDummy(suite.ctx, "--files")

	r := cmdtest.RunCli(suite.ctx, "review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "posting analysis")
}

func (suite *IntegrationSuite) TestPush() {
	cmdtest.StartDummy(suite.ctx, "--files")

	r := cmdtest.RunCli(suite.ctx, "push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "posting analysis")
}

func (suite *IntegrationSuite) TestPushNoComments() {
	cmdtest.StartDummy(suite.ctx)

	r := cmdtest.RunCli(suite.ctx, "push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "no comments were produced")
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
