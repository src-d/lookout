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
	"gopkg.in/src-d/lookout-sdk.v0/pb"

	"github.com/stretchr/testify/suite"
)

const dummyConfigFileWithTimeouts = "../../fixtures/dummy_config_with_timeouts.yml"

type errAnalyzer struct{}

func (a *errAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("review error")
}

func (a *errAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("push error")
}

type sleepyErrAnalyzer struct{}

func (a *sleepyErrAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	time.Sleep(1 * time.Millisecond)
	return nil, errors.New("review error")
}

func (a *sleepyErrAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	time.Sleep(1 * time.Millisecond)
	return nil, errors.New("push error")
}

type ErrorAnalyzerIntegrationSuite struct {
	IntegrationSuite
	configFile string
	analyzer   lookout.AnalyzerServer
	errMessage string
}

func (suite *ErrorAnalyzerIntegrationSuite) startAnalyzer(ctx context.Context, a lookout.AnalyzerServer) error {
	log.DefaultFactory = &log.LoggerFactory{
		Level: log.ErrorLevel,
	}
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := pb.Listen("ipv4://localhost:9930")
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
	suite.r, suite.w = suite.StartLookoutd(suite.configFile)

	suite.startAnalyzer(suite.Ctx, suite.analyzer)
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy`)
}

func (suite *ErrorAnalyzerIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *ErrorAnalyzerIntegrationSuite) TestAnalyzerErr() {
	suite.sendEvent(successJSON)

	suite.GrepTrue(suite.r, suite.errMessage)
}

func TestErrorAnalyzerIntegrationSuite(t *testing.T) {
	suite.Run(t, &ErrorAnalyzerIntegrationSuite{
		configFile: dummyConfigFile,
		analyzer:   &errAnalyzer{},
		errMessage: `msg="analysis failed: code: Unknown - message: review error" analyzer=Dummy app=lookoutd error="rpc error: code = Unknown desc = review error"`,
	})
	suite.Run(t, &ErrorAnalyzerIntegrationSuite{
		configFile: dummyConfigFileWithTimeouts,
		analyzer:   &sleepyErrAnalyzer{},
		errMessage: `msg="analysis failed: timeout exceeded, try increasing analyzerReviewTimeout" analyzer=Dummy app=lookoutd error="rpc error: code = DeadlineExceeded desc = context deadline exceeded"`,
	})
}
