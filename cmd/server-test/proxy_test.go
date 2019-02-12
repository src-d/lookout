// +build integration
// +build bblfsh

package server_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/src-d/lookout/util/grpchelper"
	"github.com/stretchr/testify/suite"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

const emptyConfigFile = "../../fixtures/empty_config.yml"

type ProxyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *ProxyIntegrationSuite) SetupTest() {
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(emptyConfigFile)

	// Proxy can connect before or after the "Starting watcher" message is found
	msg := `msg="connection state changed to 'READY'" addr="localhost:9432" app=lookoutd name=bblfsh-proxy`
	if !strings.Contains(suite.Output(), msg) {
		suite.GrepTrue(suite.r, msg)
	}
}

func (suite *ProxyIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *ProxyIntegrationSuite) TestParseOk() {
	addr, err := pb.ToGoGrpcAddress("ipv4://localhost:10301")
	suite.NoError(err)

	bblfshConn, err := grpchelper.DialContext(context.Background(), addr)
	suite.NoError(err)

	client := protocol.NewProtocolServiceClient(bblfshConn)

	ctx := context.TODO()
	req := &protocol.ParseRequest{
		Filename: "my.go",
		Content:  "package main",
		Encoding: protocol.UTF8,
	}
	resp, err := client.Parse(ctx, req)
	suite.NoError(err)
	suite.NotNil(resp)
}

func TestProxyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ProxyIntegrationSuite))
}
