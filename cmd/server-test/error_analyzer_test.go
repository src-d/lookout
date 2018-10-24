// +build integration

package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/src-d/lookout"
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
	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)

	suite.startAnalyzer(suite.Ctx, &errAnalyzer{})
	suite.GrepTrue(suite.r, `connection state changed to 'READY'`)
}

func (suite *ErrorAnalyzerIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *ErrorAnalyzerIntegrationSuite) TestAnalyzerErr() {
	suite.sendEvent(successJSON)

	suite.GrepTrue(suite.r, `msg="analysis failed" analyzer=Dummy app=lookoutd error="rpc error: code = Unknown desc = review error"`)
}

func TestErrorAnalyzerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ErrorAnalyzerIntegrationSuite))
}
