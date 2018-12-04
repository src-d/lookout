// +build integration

package sdk_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SDKDummyTestSuite struct {
	SdkIntegrationSuite
}

func (suite *SDKDummyTestSuite) SetupTest() {
	suite.StoppableCtx()
}

func (suite *SDKDummyTestSuite) TearDownTest() {
	suite.Stop()
}

func (suite *SDKDummyTestSuite) TestReview() {
	suite.StartDummy("--files")

	r := suite.RunCli("review",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "posting analysis")
}

func (suite *SDKDummyTestSuite) TestPush() {
	suite.StartDummy("--files")

	r := suite.RunCli("push",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "posting analysis")
}

func (suite *SDKDummyTestSuite) TestPushNoComments() {
	suite.StartDummy()

	r := suite.RunCli("push",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "no comments were produced")
}

func TestSDKDummyTestSuite(t *testing.T) {
	suite.Run(t, new(SDKDummyTestSuite))
}
