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

const dummyConfigFileWithTimeouts = "../../fixtures/dummy_config_with_timeouts.yml"

type timeoutErrAnalyzer struct{}

func (a *timeoutErrAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	time.Sleep(1 * time.Millisecond)
	return nil, errors.New("review error")
}

func (a *timeoutErrAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	time.Sleep(1 * time.Millisecond)
	return nil, errors.New("push error")
}

type TimeoutErrorAnalyzerIntegrationSuite struct {
	IntegrationSuite
	configFile string
	analyzer   lookout.AnalyzerServer
	errMessage string
}

func (suite *TimeoutErrorAnalyzerIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(suite.configFile)

	startMockAnalyzer(suite.Ctx, suite.analyzer)
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy`)
}

func (suite *TimeoutErrorAnalyzerIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *TimeoutErrorAnalyzerIntegrationSuite) TestAnalyzerTimeoutErr() {
	suite.sendEvent(successJSON)

	suite.GrepTrue(suite.r, suite.errMessage)
}

func TestTimeoutErrorAnalyzerIntegrationSuite(t *testing.T) {
	suite.Run(t, &TimeoutErrorAnalyzerIntegrationSuite{
		configFile: dummyConfigFileWithTimeouts,
		analyzer:   &timeoutErrAnalyzer{},
		errMessage: `msg="analysis failed: timeout exceeded, try increasing analyzer_review in config.yml" analyzer=Dummy app=lookoutd error="rpc error: code = DeadlineExceeded desc = context deadline exceeded"`,
	})
}
