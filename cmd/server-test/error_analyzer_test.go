// +build integration

package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/src-d/lookout"

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

func (suite *ErrorAnalyzerIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)

	startMockAnalyzer(suite.Ctx, &errAnalyzer{})
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

	suite.GrepTrue(suite.r, `msg="analysis failed" analyzer=Dummy app=lookoutd error="rpc error: code = Unknown desc = review error"`)
}

func TestErrorAnalyzerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ErrorAnalyzerIntegrationSuite))
}
