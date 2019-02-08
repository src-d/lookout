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

func correctReviewEvent() *lookout.ReviewEvent {
	return &lookout.ReviewEvent{
		ReviewEvent: pb.ReviewEvent{
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
				}}}}
}

func correctPushEvent() *lookout.PushEvent {
	return &lookout.PushEvent{
		PushEvent: pb.PushEvent{
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
				}}}}
}

var globalConfig = lookout.AnalyzerConfig{
	Name: "test",
	Settings: map[string]interface{}{
		"global_key": "global",
		"reused_key": "global",
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

	reviewEvent := correctReviewEvent()

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

	pushEvent := correctPushEvent()

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

	reviewEvent := correctReviewEvent()

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	// shouldn't comment anything as goes timeout
	require.Len(comments, 0)

	pushEvent := correctPushEvent()

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

	pushEvent := correctPushEvent()

	err := watcher.Send(pushEvent)
	require.Nil(err)

	comments := poster.PopComments()
	// shouldn't comment anything as goes timeout
	require.Len(comments, 0)

	reviewEvent := correctReviewEvent()

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

	reviewEvent := correctReviewEvent()

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

	reviewEvent := correctReviewEvent()

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

	reviewEvent := correctReviewEvent()

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

	err := watcher.Send(correctReviewEvent())
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 0)

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func (s *ServerTestSuite) TestMergeConfig() {
	fileGetter := &FileGetterMockWithConfig{
		content: `analyzers:
 - name: mock
   settings:
     local_key:  local
     reused_key: local
`,
	}

	testCases := []struct {
		name           string
		AnalyzerConfig *lookout.AnalyzerConfig
		FileGetter     lookout.FileGetter
		OrganizationOp store.OrganizationOperator
		expectedMap    map[string]interface{}
	}{
		{
			name:        "none",
			expectedMap: map[string]interface{}{},
		},
		{
			name:       "local",
			FileGetter: fileGetter,
			expectedMap: map[string]interface{}{
				"local_key":  "local",
				"reused_key": "local",
			},
		},
		{
			name:           "org",
			OrganizationOp: &OrganizationOperatorMock{},
			expectedMap: map[string]interface{}{
				"org_key":    "org",
				"reused_key": "org",
			},
		},
		{
			name:           "org,local",
			FileGetter:     fileGetter,
			OrganizationOp: &OrganizationOperatorMock{},
			expectedMap: map[string]interface{}{
				"local_key":  "local",
				"org_key":    "org",
				"reused_key": "local",
			},
		},
		{
			name:           "global",
			AnalyzerConfig: &globalConfig,
			expectedMap: map[string]interface{}{
				"global_key": "global",
				"reused_key": "global",
			},
		},
		{
			name:           "global,local",
			AnalyzerConfig: &globalConfig,
			FileGetter:     fileGetter,
			expectedMap: map[string]interface{}{
				"global_key": "global",
				"local_key":  "local",
				"reused_key": "local",
			},
		},
		{
			name:           "global,org",
			AnalyzerConfig: &globalConfig,
			OrganizationOp: &OrganizationOperatorMock{},
			expectedMap: map[string]interface{}{
				"global_key": "global",
				"org_key":    "org",
				"reused_key": "org",
			},
		},
		{
			name:           "global,org,local",
			AnalyzerConfig: &globalConfig,
			FileGetter:     fileGetter,
			OrganizationOp: &OrganizationOperatorMock{},
			expectedMap: map[string]interface{}{
				"global_key": "global",
				"local_key":  "local",
				"org_key":    "org",
				"reused_key": "local",
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			client := &AnalyzerClientMock{CommentsBuilder: makeComments}

			watcher, _ := setupMockedServer(mockedServerParams{
				AnalyzerClient: client,
				AnalyzerConfig: tc.AnalyzerConfig,
				FileGetter:     tc.FileGetter,
				OrganizationOp: tc.OrganizationOp,
			})

			t.Run("review", func(t *testing.T) {
				require := require.New(t)

				err := watcher.Send(correctReviewEvent())
				require.Nil(err)
				es := client.PopReviewEvents()
				require.Len(es, 1)

				require.Equal(
					pb.ToStruct(tc.expectedMap).GetFields(),
					es[0].Configuration.GetFields())
			})

			t.Run("push", func(t *testing.T) {
				require := require.New(t)

				err := watcher.Send(correctPushEvent())
				require.Nil(err)
				es := client.PopPushEvents()
				require.Len(es, 1)

				require.Equal(
					pb.ToStruct(tc.expectedMap).GetFields(),
					es[0].Configuration.GetFields())
			})
		})
	}
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
	OrganizationOp store.OrganizationOperator
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

	eventOp := params.EventOp
	commentOp := params.CommentOp
	organizationOp := params.OrganizationOp

	if params.Persist {
		eventOp = store.NewMemEventOperator()
		commentOp = store.NewMemCommentOperator()

		//TODO
		organizationOp = &store.NoopOrganizationOperator{}
	}

	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: analyzerClient,
			Config: analyzerConfig,
		},
	}

	srv := NewServer(Options{
		Poster:         poster,
		FileGetter:     fileGetter,
		Analyzers:      analyzers,
		EventOp:        eventOp,
		CommentOp:      commentOp,
		OrganizationOp: organizationOp,
		ReviewTimeout:  params.ReviewTimeout,
		PushTimeout:    params.PushTimeout,
	})

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

type OrganizationOperatorMock struct{}

func (o *OrganizationOperatorMock) Save(ctx context.Context, provider string, orgID string, config string) error {
	return nil
}

func (o *OrganizationOperatorMock) Config(ctx context.Context, provider string, orgID string) (string, error) {
	val := `
analyzers:
  - name: mock
    settings:
      org_key:    org
      reused_key: org
`
	return val, nil
}

var _ store.OrganizationOperator = &OrganizationOperatorMock{}

type AnalyzerClientMock struct {
	reviewEvents    []*pb.ReviewEvent
	pushEvents      []*pb.PushEvent
	CommentsBuilder func(ev lookout.Event, from, to lookout.ReferencePointer) []*lookout.Comment
	ReviewSleep     time.Duration
	PushSleep       time.Duration
}

func (a *AnalyzerClientMock) NotifyReviewEvent(ctx context.Context, in *pb.ReviewEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	if a.ReviewSleep > 0 {
		select {
		case <-time.After(a.ReviewSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	a.reviewEvents = append(a.reviewEvents, in)
	return &lookout.EventResponse{
		Comments: a.CommentsBuilder(&lookout.ReviewEvent{ReviewEvent: *in},
			in.CommitRevision.Base, in.CommitRevision.Head),
	}, nil
}

func (a *AnalyzerClientMock) NotifyPushEvent(ctx context.Context, in *pb.PushEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	if a.PushSleep > 0 {
		select {
		case <-time.After(a.PushSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	a.pushEvents = append(a.pushEvents, in)
	return &lookout.EventResponse{
		Comments: a.CommentsBuilder(&lookout.PushEvent{PushEvent: *in},
			in.CommitRevision.Base, in.CommitRevision.Head),
	}, nil
}

func (a *AnalyzerClientMock) PopReviewEvents() []*pb.ReviewEvent {
	res := a.reviewEvents[:]
	a.reviewEvents = []*pb.ReviewEvent{}
	return res
}

func (a *AnalyzerClientMock) PopPushEvents() []*pb.PushEvent {
	res := a.pushEvents[:]
	a.pushEvents = []*pb.PushEvent{}
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
