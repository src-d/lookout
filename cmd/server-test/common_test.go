// +build integration

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cmdtest"
	"github.com/src-d/lookout/util/grpchelper"

	"github.com/src-d/lookout-test-fixtures"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

const dummyConfigFile = "../../fixtures/dummy_config.yml"

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
	*pb.ReviewEvent
}

func (e *jsonReviewEvent) String() string {
	e.Event = "review"
	b, _ := json.Marshal(e)
	return string(b)
}

type jsonPushEvent struct {
	Event string `json:"event"`
	*pb.PushEvent
}

func (e *jsonPushEvent) String() string {
	e.Event = "push"
	b, _ := json.Marshal(e)
	return string(b)
}

type mockAnalyzer interface {
	NotifyReviewEvent(context.Context, *pb.ReviewEvent) (*lookout.EventResponse, error)
	NotifyPushEvent(context.Context, *pb.PushEvent) (*lookout.EventResponse, error)
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

func castPullRequest(fixture *fixtures.Fixture) (*jsonReviewEvent, error) {
	pr := fixture.GetPR()

	baseRepoCloneUrl := pr.GetBase().GetRepo().GetCloneURL()
	baseRepoInfo, err := pb.ParseRepositoryInfo(baseRepoCloneUrl)
	if err != nil {
		return nil, err
	}

	headRepoCloneUrl := pr.GetHead().GetRepo().GetCloneURL()
	headRepoInfo, err := pb.ParseRepositoryInfo(headRepoCloneUrl)
	if err != nil {
		return nil, err
	}

	event := pb.ReviewEvent{}

	event.Provider = "github"
	event.InternalID = strconv.FormatInt(pr.GetID(), 10)

	event.Number = uint32(pr.GetNumber())
	event.RepositoryID = uint32(pr.GetHead().GetRepo().GetID())

	sourceRefName := fmt.Sprintf("refs/heads/%s", pr.GetHead().GetRef())
	event.Source = lookout.ReferencePointer{
		InternalRepositoryURL: headRepoInfo.CloneURL,
		ReferenceName:         plumbing.ReferenceName(sourceRefName),
		Hash:                  pr.GetHead().GetSHA(),
	}

	baseRefName := fmt.Sprintf("refs/heads/%s", pr.GetBase().GetRef())
	event.Base = lookout.ReferencePointer{
		InternalRepositoryURL: baseRepoInfo.CloneURL,
		ReferenceName:         plumbing.ReferenceName(baseRefName),
		Hash:                  pr.GetBase().GetSHA(),
	}

	headRefName := fmt.Sprintf("refs/pull/%d/head", pr.GetNumber())
	event.Head = lookout.ReferencePointer{
		InternalRepositoryURL: baseRepoInfo.CloneURL,
		ReferenceName:         plumbing.ReferenceName(headRefName),
		Hash:                  pr.GetHead().GetSHA(),
	}

	event.IsMergeable = pr.GetMergeable()

	return &jsonReviewEvent{ReviewEvent: &event}, nil
}

var longLineFixture = fixtures.GetByName("new-go-file-too-long-line")

var successEvent = &jsonReviewEvent{
	ReviewEvent: &pb.ReviewEvent{
		InternalID:     "1",
		Number:         1,
		CommitRevision: *longLineFixture.GetCommitRevision(),
	},
}

var successJSON = successEvent.String()
