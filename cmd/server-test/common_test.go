// +build integration

package server_test

import (
	"fmt"
	"io"

	"github.com/src-d/lookout/util/cmdtest"
)

type IntegrationSuite struct {
	cmdtest.IntegrationSuite
	r io.Reader
	w io.WriteCloser
}

func (suite *IntegrationSuite) sendEvent(json string) {
	_, err := fmt.Fprintln(suite.w, json)
	suite.Require().NoError(err)
}
