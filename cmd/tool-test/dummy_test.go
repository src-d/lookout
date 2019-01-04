// +build integration

package tool_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ToolDummyTestSuite struct {
	ToolIntegrationSuite
}

func (suite *ToolDummyTestSuite) SetupTest() {
	suite.StoppableCtx()
}

func (suite *ToolDummyTestSuite) TearDownTest() {
	suite.Stop()
}

func (suite *ToolDummyTestSuite) TestReview() {
	suite.StartDummy("--files")

	r := suite.RunCli("review",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "posting analysis")
}

func (suite *ToolDummyTestSuite) TestPush() {
	suite.StartDummy("--files")

	r := suite.RunCli("push",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "posting analysis")
}

func (suite *ToolDummyTestSuite) TestPushNoComments() {
	suite.StartDummy()

	r := suite.RunCli("push",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "no comments were produced")
}

func TestToolDummyTestSuite(t *testing.T) {
	suite.Run(t, new(ToolDummyTestSuite))
}
