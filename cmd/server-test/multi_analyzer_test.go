// +build integration

package server_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
)

const doubleDummyConfigFile = "../../fixtures/double_dummy_config.yml"

type MultiDummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *MultiDummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.StartDummy("--files")
	suite.StartDummy("--files", "--analyzer", "ipv4://localhost:10303")

	suite.r, suite.w = suite.StartLookoutd(doubleDummyConfigFile)
}

func (suite *MultiDummyIntegrationSuite) TearDownTest() {
	suite.Stop()
}

func (suite *MultiDummyIntegrationSuite) TestSuccessReview() {
	var teebuf bytes.Buffer
	tee := io.TeeReader(suite.r, &teebuf)

	suite.sendEvent(successJSON)
	suite.GrepTrue(tee, "processing pull request")
	suite.GrepTrue(tee, "posting analysis")
	found, buf := suite.Grep(tee, `status=success`)
	suite.Require().Truef(found, "'%s' not found in:\n%s", `status=success`, teebuf.String())

	st := buf.String()

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy1","file":"another.go","line":3,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the first analyzer from %s in logs:\n%s", doubleDummyConfigFile, teebuf.String()),
	)

	suite.Require().Contains(
		st,
		`{"analyzer-name":"Dummy2","file":"another.go","line":3,"text":"This line exceeded`,
		fmt.Sprintf("no comments from the second analyzer from %s in logs:\n%s", doubleDummyConfigFile, teebuf.String()),
	)
}

func TestMultiDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MultiDummyIntegrationSuite))
}
