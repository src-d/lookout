// +build integration

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/src-d/lookout"
	fixtures "github.com/src-d/lookout-test-fixtures"
	"github.com/src-d/lookout/util/cmdtest"
	"github.com/src-d/lookout/util/grpchelper"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
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

type jsonReviewEvent struct {
	Event string `json:"event"`
	*lookout.ReviewEvent
}

func (e *jsonReviewEvent) String() string {
	e.Event = "review"
	b, _ := json.Marshal(e)
	return string(b)
}

type jsonPushEvent struct {
	Event string `json:"event"`
	*lookout.PushEvent
}

func (e *jsonPushEvent) String() string {
	e.Event = "push"
	b, _ := json.Marshal(e)
	return string(b)
}

type mockAnalyzer interface {
	NotifyReviewEvent(context.Context, *lookout.ReviewEvent) (*lookout.EventResponse, error)
	NotifyPushEvent(context.Context, *lookout.PushEvent) (*lookout.EventResponse, error)
}

func startMockAnalyzer(ctx context.Context, a mockAnalyzer) error {
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

var longLineFixture = fixtures.GetByName("new-go-file-too-long-line")

var successEvent = &jsonReviewEvent{
	ReviewEvent: &lookout.ReviewEvent{
		InternalID:     "1",
		Number:         1,
		CommitRevision: *longLineFixture.GetCommitRevision(),
	},
}

var successJSON = successEvent.String()
