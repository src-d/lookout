// +build integration

package sdk_test

import (
	"testing"

	"github.com/src-d/lookout/util/cmdtest"

	"github.com/stretchr/testify/suite"
)

type SDKDummyTestSuite struct {
	cmdtest.IntegrationSuite
}

func (suite *SDKDummyTestSuite) SetupTest() {
	suite.StoppableCtx()
}

func (suite *SDKDummyTestSuite) TearDownTest() {
	suite.Stop()
}

func (suite *SDKDummyTestSuite) TestReview() {
	suite.StartDummy("--files")

	r := suite.RunCli("review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "posting analysis")
}

func (suite *SDKDummyTestSuite) TestPush() {
	suite.StartDummy("--files")

	r := suite.RunCli("push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "posting analysis")
}

func (suite *SDKDummyTestSuite) TestPushNoComments() {
	suite.StartDummy()

	r := suite.RunCli("push", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	cmdtest.GrepTrue(r, "no comments were produced")
}

func TestSDKDummyTestSuite(t *testing.T) {
	suite.Run(t, new(SDKDummyTestSuite))
}
