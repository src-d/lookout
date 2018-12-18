// +build integration

package sdk_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/src-d/lookout/util/cmdtest"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/suite"
)

const dummyConfigFile = "../../fixtures/dummy_config.yml"

type SDKArgsTestSuite struct {
	SdkIntegrationSuite
}

func (suite *SDKArgsTestSuite) SetupTest() {
	suite.StoppableCtx()
}

func (suite *SDKArgsTestSuite) TearDownTest() {
	suite.Stop()
}

func (suite *SDKArgsTestSuite) TestArgs() {
	suite.StartDummy()

	configArg := fmt.Sprintf("--config=%s", dummyConfigFile)

	mainArgs := []string{
		"review",
		"--git-dir=" + suite.gitPath,
		"--from=" + logLineRevision.Base.Hash,
		"--to=" + logLineRevision.Head.Hash,
	}

	lineTestCases := []struct {
		args []string
		err  bool
	}{
		{[]string{}, false},
		{[]string{"ipv4://localhost:9930"}, false},
		{[]string{"--config-json={}"}, false},
		{[]string{"--config-json={}", "ipv4://localhost:9930"}, false},
		{[]string{configArg}, false},
		{[]string{configArg, "ipv4://localhost:9930"}, true},
		{[]string{configArg, "--config-json={}"}, true},
		{[]string{configArg, "--config-json={}", "ipv4://localhost:9930"}, true},
	}

	for _, tc := range lineTestCases {
		name := fmt.Sprintf("with args %s", strings.Join(tc.args, " "))
		suite.T().Run(name, func(t *testing.T) {
			assert := assert.New(t)

			args := append(mainArgs, tc.args...)

			cliCmd := exec.CommandContext(suite.Ctx, cmdtest.LookoutBin, args...)
			err := cliCmd.Run()

			if tc.err {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestSDKArgsTestSuite(t *testing.T) {
	suite.Run(t, new(SDKArgsTestSuite))
}
