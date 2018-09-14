// +build integration

package sdk_test

import (
	"io"
	"testing"

	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/stretchr/testify/suite"
)

type BblfshIntegrationSuite struct {
	SdkIntegrationSuite
}

func (suite *BblfshIntegrationSuite) SetupSuite() {
	suite.SdkIntegrationSuite.SetupSuite()

	suite.StoppableCtx()
	suite.StartDummy("--uast", "--files")
}

func (suite *BblfshIntegrationSuite) TearDownSuite() {
	suite.Stop()
}

func (suite *BblfshIntegrationSuite) RunReview() io.Reader {
	return suite.RunCli("review", "ipv4://localhost:10302",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
}

func (suite *BblfshIntegrationSuite) RunPush() io.Reader {
	return suite.RunCli("push", "ipv4://localhost:10302",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
}

func (suite *BblfshIntegrationSuite) TestReviewNoBblfshError() {
	r := suite.RunCli("review", "ipv4://localhost:10302",
		"--bblfshd=ipv4://localhost:0000",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestReviewNoUASTWarning() {
	fixture := fixtures.GetByName("bblfsh-unknown-language")
	rv := fixture.GetCommitRevision()

	r := suite.RunCli("push", "ipv4://localhost:10302",
		"--git-dir="+suite.gitPath,
		"--from="+rv.Base.Hash,
		"--to="+rv.Head.Hash)
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
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestPushNoUASTWarning() {
	fixture := fixtures.GetByName("bblfsh-unknown-language")
	rv := fixture.GetCommitRevision()

	r := suite.RunCli("push", "ipv4://localhost:10302",
		"--git-dir="+suite.gitPath,
		"--from="+rv.Base.Hash,
		"--to="+rv.Head.Hash)
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
