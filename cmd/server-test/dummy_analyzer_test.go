// +build integration

package server_test

import (
	"testing"
	"time"

	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

type DummyIntegrationSuite struct {
	IntegrationSuite
}

func (suite *DummyIntegrationSuite) SetupTest() {
	suite.ResetDB()

	suite.StoppableCtx()
	suite.r, suite.w = suite.StartLookoutd(dummyConfigFile)

	suite.StartDummy("--files")
	suite.GrepTrue(suite.r, `msg="connection state changed to 'READY'" addr="ipv4://localhost:9930" analyzer=Dummy`)
}

func (suite *DummyIntegrationSuite) TearDownTest() {
	// TODO: for integration tests with RabbitMQ we wait a bit so the queue
	// is depleted. Ideally this would be done with something similar to ResetDB
	time.Sleep(5 * time.Second)
	suite.Stop()
}

func (suite *DummyIntegrationSuite) TestSuccessReview() {
	suite.sendEvent(successJSON)
	suite.GrepAll(suite.r, []string{
		`processing pull request`,
		`{"analyzer-name":"Dummy","file":"another.go","line":3,"text":"This line exceeded`,
		`status=success`,
	})
}

func (suite *DummyIntegrationSuite) TestGRPCLogs() {
	// Check that 'event-id' log field is sent from lookoutd to the dummy analyzer,
	// and then received back when receiving a grpc call from the analyzer.
	suite.sendEvent(successJSON)
	suite.GrepAll(suite.r, []string{
		`processing pull request`,
		`msg="gRPC streaming server call started" analyzer=Dummy app=lookoutd event-id=16ee0f607886b841c7633ab4cea5334cbc2022a1 event-type="*lookout.ReviewEvent" grpc.method=GetChanges`,
		`status=success`,
	})
}

func (suite *DummyIntegrationSuite) TestReviewDontPostSameComment() {
	fixture := fixtures.GetByName("incremental-pr")

	rev0Event := &jsonReviewEvent{
		ReviewEvent: &pb.ReviewEvent{
			InternalID:     "some-id",
			Number:         1,
			CommitRevision: *fixture.GetRevision(0).GetCommitRevision(),
		},
	}

	suite.sendEvent(rev0Event.String())
	suite.GrepAll(suite.r, []string{
		`{"analyzer-name":"Dummy","file":"dummy.go","line":5,"text":"This line exceeded`,
		`status=success`,
	})

	rev1Event := &jsonReviewEvent{
		ReviewEvent: &pb.ReviewEvent{
			InternalID:     "some-id",
			Number:         1,
			CommitRevision: *fixture.GetRevision(1).GetCommitRevision(),
		},
	}

	suite.sendEvent(rev1Event.String())
	suite.GrepAndNotAll(suite.r,
		[]string{
			`processing pull request`,
			`{"analyzer-name":"Dummy","file":"dummy.go","text":"The file has increased`,
			`status=success`,
		}, []string{
			`{"analyzer-name":"Dummy","file":"dummy.go","line":5,"text":"This line exceeded`,
		})
}

func (suite *DummyIntegrationSuite) TestWrongRevision() {
	e := &jsonReviewEvent{
		ReviewEvent: &pb.ReviewEvent{
			InternalID:     "3",
			Number:         3,
			CommitRevision: *longLineFixture.GetCommitRevision(),
		},
	}
	// change hashes to incorrect ones
	e.CommitRevision.Base.Hash = "0000000000000000000000000000000000000000"
	e.CommitRevision.Head.Hash = "0000000000000000000000000000000000000000"

	suite.sendEvent(e.String())
	suite.GrepTrue(suite.r, `event processing failed`)
}

func (suite *DummyIntegrationSuite) TestSuccessPush() {
	pushEvent := jsonPushEvent{
		PushEvent: &pb.PushEvent{
			InternalID:     "1",
			CommitRevision: *longLineFixture.GetCommitRevision(),
		},
	}
	suite.sendEvent(pushEvent.String())
	suite.GrepAll(suite.r, []string{
		"processing push",
		"comments can belong only to review event but 1 is given",
		`status=error`,
	})
}

func TestDummyIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DummyIntegrationSuite))
}
