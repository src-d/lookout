// +build integration

package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cmdtest"
	"github.com/src-d/lookout/util/grpchelper"
	log "gopkg.in/src-d/go-log.v1"

	"github.com/stretchr/testify/suite"
)

type errAnalyzer struct{}

func (a *errAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("review error")
}

func (a *errAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("push error")
}

type ErrorAnalyzerIntegrationSuite struct {
	IntegrationSuite
}

func (suite *ErrorAnalyzerIntegrationSuite) startAnalyzer(ctx context.Context, a lookout.AnalyzerServer) error {
	log.DefaultFactory = &log.LoggerFactory{
		Level: log.ErrorLevel,
	}
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := grpchelper.Listen("ipv4://localhost:10302")
	if err != nil {
		return err
	}

	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.Stop()
	}()
	return nil
}

func (suite *ErrorAnalyzerIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.startAnalyzer(suite.Ctx, &errAnalyzer{})
	suite.r, suite.w = suite.StartServe("--provider", "json",
		"-c", "../../fixtures/dummy_config.yml", "dummy-repo-url")

	// make sure server started correctly
	cmdtest.GrepTrue(suite.r, "Starting watcher")
}

func (suite *ErrorAnalyzerIntegrationSuite) TearDownTest() {
	suite.Stop()
}

func (suite *ErrorAnalyzerIntegrationSuite) TestAnalyzerErr() {
	suite.sendEvent(successJSON)

	cmdtest.GrepTrue(suite.r, `msg="analysis failed" analyzer=Dummy app=lookout error="rpc error: code = Unknown desc = review error"`)
}

func TestErrorAnalyzerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ErrorAnalyzerIntegrationSuite))
}
