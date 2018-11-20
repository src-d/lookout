// +build integration

package server_test

import (
	"context"
	"testing"

	"github.com/src-d/lookout/util/grpchelper"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	log "gopkg.in/src-d/go-log.v1"
)

const emptyConfigFile = "../../fixtures/empty_config.yml"

type ProxyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *ProxyIntegrationSuite) SetupTest() {
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(emptyConfigFile)
}

func (suite *ProxyIntegrationSuite) TearDownTest() {
	suite.Stop()
}

func (suite *ProxyIntegrationSuite) TestParseOk() {
	addr, err := grpchelper.ToGoGrpcAddress("ipv4://localhost:10301")
	suite.NoError(err)

	bblfshConn, err := grpchelper.DialContext(context.Background(), addr, grpc.WithInsecure())
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
