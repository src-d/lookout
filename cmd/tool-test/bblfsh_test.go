// +build integration

package tool_test

import (
	"context"
	"io"
	"testing"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/src-d/lookout/util/grpchelper"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
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
	return suite.RunCli("review",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
}

func (suite *BblfshIntegrationSuite) RunPush() io.Reader {
	return suite.RunCli("push",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
}

func (suite *BblfshIntegrationSuite) TestReviewNoBblfshError() {
	r := suite.RunCliErr("review",
		"--bblfshd=ipv4://localhost:0000",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestReviewNoUASTWarning() {
	fixture := fixtures.GetByName("bblfsh-unknown-language")
	rv := fixture.GetCommitRevision()

	r := suite.RunCli("push",
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
	r := suite.RunCliErr("push",
		"--bblfshd=ipv4://localhost:0000",
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)
	suite.GrepTrue(r, "WantUAST isn't allowed")
}

func (suite *BblfshIntegrationSuite) TestPushNoUASTWarning() {
	fixture := fixtures.GetByName("bblfsh-unknown-language")
	rv := fixture.GetCommitRevision()

	r := suite.RunCli("push",
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

func (suite *BblfshIntegrationSuite) TestConnectToDataServer() {
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	bblfshAnalyzerAddr := "ipv4://localhost:9931"

	a := &BbblfshClientAnalyzer{}

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := pb.Listen(bblfshAnalyzerAddr)
	suite.NoError(err)

	go func() {
		suite.NoError(server.Serve(lis))
	}()
	defer server.Stop()

	r := suite.RunCli("review", bblfshAnalyzerAddr,
		"--git-dir="+suite.gitPath,
		"--from="+logLineRevision.Base.Hash,
		"--to="+logLineRevision.Head.Hash)

	suite.GrepTrue(r, "parse request finished successfully")
}

func TestBblfshIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test suite in short mode.")
	}

	suite.Run(t, new(BblfshIntegrationSuite))
}

var _ lookout.AnalyzerServer = &BbblfshClientAnalyzer{}

type BbblfshClientAnalyzer struct{}

func (a *BbblfshClientAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	dataServer, _ := pb.ToGoGrpcAddress("ipv4://localhost:10301")
	bblfshConn, err := grpchelper.DialContext(context.Background(), dataServer, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := protocol.NewProtocolServiceClient(bblfshConn)

	req := &protocol.ParseRequest{
		Filename: "my.go",
		Content:  "package main",
		Encoding: protocol.UTF8,
	}
	_, err = client.Parse(ctx, req)
	if err != nil {
		return nil, err
	}

	return &lookout.EventResponse{
		Comments: []*pb.Comment{
			{Text: "parse request finished successfully"},
		},
	}, nil
}

func (a *BbblfshClientAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	return nil, nil
}
