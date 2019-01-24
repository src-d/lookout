package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/mock"
	"github.com/src-d/lookout/store"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var correctReviewEvent = lookout.ReviewEvent{
	Provider:    "Mock",
	InternalID:  "internal-id",
	IsMergeable: true,
	Source: lookout.ReferencePointer{
		InternalRepositoryURL: "file:///test",
		ReferenceName:         "feature",
		Hash:                  "source-hash",
	},
	Merge: lookout.ReferencePointer{
		InternalRepositoryURL: "file:///test",
		ReferenceName:         "merge-branch",
		Hash:                  "merge-hash",
	},
	CommitRevision: lookout.CommitRevision{
		Base: lookout.ReferencePointer{
			InternalRepositoryURL: "file:///test",
			ReferenceName:         "master",
			Hash:                  "base-hash",
		},
		Head: lookout.ReferencePointer{
			InternalRepositoryURL: "file:///test",
			ReferenceName:         "master",
			Hash:                  "head-hash",
		},
	},
}
var correctPushEvent = lookout.PushEvent{
	Provider:   "Mock",
	InternalID: "internal-id",
	CommitRevision: lookout.CommitRevision{
		Base: lookout.ReferencePointer{
			InternalRepositoryURL: "file:///test",
			ReferenceName:         "master",
			Hash:                  "base-hash",
		},
		Head: lookout.ReferencePointer{
			InternalRepositoryURL: "file:///test",
			ReferenceName:         "master",
			Hash:                  "head-hash",
		},
	},
}
var globalConfig = lookout.AnalyzerConfig{
	Name: "test",
	Settings: map[string]interface{}{
		"key_from_global": 1,
	},
}

func init() {
	log.DefaultLogger = log.New(log.Fields{"app": "lookout"})
}

type ServerTestSuite struct {
	suite.Suite
}

func (s *ServerTestSuite) TestReview() {
	require := s.Require()

	watcher, poster := setupMockedServerDefault()

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)
	require.Equal(makeComment(reviewEvent.CommitRevision.Base, reviewEvent.CommitRevision.Head), comments[0])

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func (s *ServerTestSuite) TestPush() {
	require := s.Require()

	watcher, poster := setupMockedServerDefault()

	pushEvent := &correctPushEvent

	err := watcher.Send(pushEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)
	require.Equal(makeComment(pushEvent.CommitRevision.Base, pushEvent.CommitRevision.Head), comments[0])

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func (s *ServerTestSuite) TestReviewTimeout() {
	require := s.Require()

	reviewSleep, _ := time.ParseDuration("5ms")
	reviewTimeout, _ := time.ParseDuration("1ms")
	client := &AnalyzerClientMock{
		CommentsBuilder: makeComments,
		ReviewSleep:     reviewSleep,
	}
	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		ReviewTimeout:  reviewTimeout,
	})

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	// shouldn't comment anything as goes timeout
	require.Len(comments, 0)

	pushEvent := &correctPushEvent

	// send other event without timeout
	err = watcher.Send(pushEvent)
	require.Nil(err)

	comments = poster.PopComments()
	require.Len(comments, 1)
}

func (s *ServerTestSuite) TestPushTimeout() {
	require := s.Require()

	pushSleep, _ := time.ParseDuration("5ms")
	pushTimeout, _ := time.ParseDuration("1ms")
	client := &AnalyzerClientMock{
		CommentsBuilder: makeComments,
		PushSleep:       pushSleep,
	}
	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		PushTimeout:    pushTimeout,
	})

	pushEvent := &correctPushEvent

	err := watcher.Send(pushEvent)
	require.Nil(err)

	comments := poster.PopComments()
	// shouldn't comment anything as goes timeout
	require.Len(comments, 0)

	reviewEvent := &correctReviewEvent

	// send other event without timeout
	err = watcher.Send(reviewEvent)
	require.Nil(err)

	comments = poster.PopComments()
	require.Len(comments, 1)
}

func (s *ServerTestSuite) TestPersistedReview() {
	require := s.Require()

	client := &AnalyzerClientMock{CommentsBuilder: makeComments}
	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		Persist:        true,
	})

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)

	// reset client
	client.PopReviewEvents()

	// send the same event once again
	err = watcher.Send(reviewEvent)
	require.Nil(err)

	// shouldn't call analyzer
	require.Len(client.PopReviewEvents(), 0)

	// shouldn't comment anything
	comments = poster.PopComments()
	require.Len(comments, 0)
}

func (s *ServerTestSuite) TestReviewDuplicatedComments() {
	require := s.Require()

	client := &AnalyzerClientMock{
		CommentsBuilder: func(ev lookout.Event, from, to lookout.ReferencePointer) []*lookout.Comment {
			return []*lookout.Comment{
				{File: "foo", Line: 1, Text: "some-text", Confidence: 1},
				{File: "foo", Line: 1, Text: "some-text", Confidence: 2},
				{File: "foo", Line: 1, Text: "some-other-text", Confidence: 4},
				{File: "bar", Line: 1, Text: "some-text", Confidence: 8},
				{File: "bar", Line: 2, Text: "some-text", Confidence: 16},
				{File: "bar", Line: 2, Text: "some-other-text", Confidence: 32},
			}
		},
	}

	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
	})

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	// should discard duplicated comments
	require.Len(comments, 5)

	// for testing use confidence as id, the confidence used in the fixtures
	// has been chosen so that the sum of every possible combination has a
	// unique value (similarly to unix permissions)
	sum := 0
	for _, c := range comments {
		sum += int(c.Confidence)
	}

	require.Equal(sum, 61)
}

func (s *ServerTestSuite) TestIncrementalReview() {
	require := s.Require()

	client := &AnalyzerClientMock{
		CommentsBuilder: func(ev lookout.Event, from, to lookout.ReferencePointer) []*lookout.Comment {
			return []*lookout.Comment{
				{Text: "some-text-1"},
				{Text: "some-text-2"},
			}
		},
	}
	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		Persist:        true,
	})

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 2)

	// reset client
	client.PopReviewEvents()

	// send event with the same id but different sha1
	reviewEvent.Head.Hash = "new-sha"
	err = watcher.Send(reviewEvent)
	require.Nil(err)

	// should call analyzer
	require.Len(client.PopReviewEvents(), 1)

	// shouldn't comment anything
	comments = poster.PopComments()
	require.Len(comments, 0)
}

func (s *ServerTestSuite) TestAnalyzerConfigDisabled() {
	require := s.Require()

	watcher, poster := setupMockedServer(mockedServerParams{
		AnalyzerConfig: &lookout.AnalyzerConfig{
			Disabled: true,
		},
	})

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 0)

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func (s *ServerTestSuite) TestMergeConfigWithoutLocal() {
	require := s.Require()

	client := &AnalyzerClientMock{CommentsBuilder: makeComments}
	watcher, _ := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		AnalyzerConfig: &globalConfig,
	})

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := client.PopReviewEvents()
	require.Len(es, 1)

	require.Equal(pb.ToStruct(globalConfig.Settings), &es[0].Configuration)
}

func (s *ServerTestSuite) TestMergeConfigWithLocal() {
	require := s.Require()

	fileGetter := &FileGetterMockWithConfig{
		content: `analyzers:
 - name: mock
   settings:
     some: value
`,
	}
	client := &AnalyzerClientMock{CommentsBuilder: makeComments}

	watcher, _ := setupMockedServer(mockedServerParams{
		AnalyzerClient: client,
		AnalyzerConfig: &globalConfig,
		FileGetter:     fileGetter,
	})

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := client.PopReviewEvents()
	require.Len(es, 1)

	expectedMap := make(map[string]interface{})
	for k, v := range globalConfig.Settings {
		expectedMap[k] = v
	}
	expectedMap["some"] = "value"

	require.Equal(pb.ToStruct(expectedMap), &es[0].Configuration)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func TestConfigMerger(t *testing.T) {
	require := require.New(t)

	global := map[string]interface{}{
		"primitive":  1,
		"toOverride": 2,
		"array":      []int{1, 2},
		"object": map[string]interface{}{
			"primitive":  1,
			"toOverride": 2,
			"subobject": map[string]interface{}{
				"primitive": 1,
			},
		},
	}

	local := map[string]interface{}{
		"new":        1,
		"toOverride": 3,
		"array":      []int{3},
		"object": map[string]interface{}{
			"new":        1,
			"toOverride": 3,
			"subobject":  nil,
		},
		"newObject": map[string]interface{}{
			"new": 1,
		},
	}

	merged := mergeSettings(global, local)

	expectedMap := map[string]interface{}{
		"primitive":  1,
		"new":        1,
		"toOverride": 3,
		"array":      []int{3},
		"object": map[string]interface{}{
			"primitive":  1,
			"new":        1,
			"toOverride": 3,
			"subobject":  nil,
		},
		"newObject": map[string]interface{}{
			"new": 1,
		},
	}

	require.Equal(expectedMap, merged)
}

type mockedServerParams struct {
	AnalyzerClient lookout.AnalyzerClient
	AnalyzerConfig *lookout.AnalyzerConfig
	FileGetter     lookout.FileGetter
	EventOp        store.EventOperator
	CommentOp      store.CommentOperator
	ReviewTimeout  time.Duration
	PushTimeout    time.Duration
	Persist        bool
}

func setupMockedServerDefault() (*WatcherMock, *PosterMock) {
	return setupMockedServer(mockedServerParams{})
}

func setupMockedServer(params mockedServerParams) (*WatcherMock, *PosterMock) {
	watcher := &WatcherMock{}
	poster := &PosterMock{}

	var fileGetter lookout.FileGetter
	if params.FileGetter == nil {
		fileGetter = &FileGetterMock{}
	} else {
		fileGetter = params.FileGetter
	}

	var analyzerClient lookout.AnalyzerClient
	if params.AnalyzerClient == nil {
		analyzerClient = &AnalyzerClientMock{CommentsBuilder: makeComments}
	} else {
		analyzerClient = params.AnalyzerClient
	}

	var analyzerConfig lookout.AnalyzerConfig
	if params.AnalyzerConfig == nil {
		analyzerConfig = lookout.AnalyzerConfig{}
	} else {
		analyzerConfig = *params.AnalyzerConfig
	}

	var eventOp store.EventOperator
	var commentOp store.CommentOperator
	var organizationOp store.OrganizationOperator
	if params.Persist {
		eventOp = store.NewMemEventOperator()
		commentOp = store.NewMemCommentOperator()

		//TODO
		organizationOp = &store.NoopOrganizationOperator{}
	} else {
		eventOp = &store.NoopEventOperator{}
		commentOp = &store.NoopCommentOperator{}
		organizationOp = &store.NoopOrganizationOperator{}
	}

	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: analyzerClient,
			Config: analyzerConfig,
		},
	}

	srv := NewServer(
		poster, fileGetter, analyzers, eventOp, commentOp, organizationOp,
		params.ReviewTimeout, params.PushTimeout)
	watcher.Watch(context.TODO(), srv.HandleEvent)

	return watcher, poster
}

type WatcherMock struct {
	handler lookout.EventHandler
}

func (w *WatcherMock) Watch(ctx context.Context, e lookout.EventHandler) error {
	w.handler = e
	return nil
}

func (w *WatcherMock) Send(e lookout.Event) error {
	return w.handler(context.Background(), e)
}

var _ lookout.Poster = &PosterMock{}

type PosterMock struct {
	comments []*lookout.Comment
	status   lookout.AnalysisStatus
}

func (p *PosterMock) Post(_ context.Context, e lookout.Event, aCommentsList []lookout.AnalyzerComments, safe bool) error {
	cs := make([]*lookout.Comment, 0)
	for _, aComments := range aCommentsList {
		cs = append(cs, aComments.Comments...)
	}
	p.comments = cs
	return nil
}

func (p *PosterMock) PopComments() []*lookout.Comment {
	cs := p.comments[:]
	p.comments = []*lookout.Comment{}
	return cs
}

func (p *PosterMock) Status(_ context.Context, e lookout.Event, st lookout.AnalysisStatus) error {
	p.status = st
	return nil
}

func (p *PosterMock) PopStatus() lookout.AnalysisStatus {
	st := p.status
	p.status = 0
	return st
}

type FileGetterMock struct {
}

func (g *FileGetterMock) GetFiles(_ context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	return &NoopFileScanner{}, nil
}

type FileGetterMockWithConfig struct {
	content string
}

func (g *FileGetterMockWithConfig) GetFiles(_ context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	if req.IncludePattern == `^\.lookout\.yml$` {
		return &mock.SliceFileScanner{Files: []*lookout.File{{
			Path:    ".lookout.yml",
			Content: []byte(g.content),
		}}}, nil
	}
	return &NoopFileScanner{}, nil
}

type AnalyzerClientMock struct {
	reviewEvents    []*lookout.ReviewEvent
	pushEvents      []*lookout.PushEvent
	CommentsBuilder func(ev lookout.Event, from, to lookout.ReferencePointer) []*lookout.Comment
	ReviewSleep     time.Duration
	PushSleep       time.Duration
}

func (a *AnalyzerClientMock) NotifyReviewEvent(ctx context.Context, in *lookout.ReviewEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	if a.ReviewSleep > 0 {
		select {
		case <-time.After(a.ReviewSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	a.reviewEvents = append(a.reviewEvents, in)
	return &lookout.EventResponse{
		Comments: a.CommentsBuilder(in, in.CommitRevision.Base, in.CommitRevision.Head),
	}, nil
}

func (a *AnalyzerClientMock) NotifyPushEvent(ctx context.Context, in *lookout.PushEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	if a.PushSleep > 0 {
		select {
		case <-time.After(a.PushSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	a.pushEvents = append(a.pushEvents, in)
	return &lookout.EventResponse{
		Comments: a.CommentsBuilder(in, in.CommitRevision.Base, in.CommitRevision.Head),
	}, nil
}

func (a *AnalyzerClientMock) PopReviewEvents() []*lookout.ReviewEvent {
	res := a.reviewEvents[:]
	a.reviewEvents = []*lookout.ReviewEvent{}
	return res
}

func (a *AnalyzerClientMock) PopPushEvents() []*lookout.PushEvent {
	res := a.pushEvents[:]
	a.pushEvents = []*lookout.PushEvent{}
	return res
}

func makeComment(from, to lookout.ReferencePointer) *lookout.Comment {
	return makeCommentFromString(fmt.Sprintf("%s > %s", from.Hash, to.Hash))
}

func makeComments(ev lookout.Event, from, to lookout.ReferencePointer) []*lookout.Comment {
	return []*lookout.Comment{makeComment(from, to)}
}

func makeCommentFromString(text string) *lookout.Comment {
	return &lookout.Comment{Text: fmt.Sprintf(text)}
}

type NoopFileScanner struct {
}

func (s *NoopFileScanner) Next() bool {
	return false
}

func (s *NoopFileScanner) Err() error {
	return nil
}

func (s *NoopFileScanner) File() *lookout.File {
	return nil
}

func (s *NoopFileScanner) Close() error {
	return nil
}
