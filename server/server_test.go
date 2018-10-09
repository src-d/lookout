package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/mock"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/grpchelper"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
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

func init() {
	log.DefaultLogger = log.New(log.Fields{"app": "lookout"})
}

func TestServerReview(t *testing.T) {
	require := require.New(t)

	watcher, poster := setupMockedServer()

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)
	require.Equal(makeComment(reviewEvent.CommitRevision.Base, reviewEvent.CommitRevision.Head), comments[0])

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func TestServerPush(t *testing.T) {
	require := require.New(t)

	watcher, poster := setupMockedServer()

	pushEvent := &lookout.PushEvent{
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

	err := watcher.Send(pushEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)
	require.Equal(makeComment(pushEvent.CommitRevision.Base, pushEvent.CommitRevision.Head), comments[0])

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

func TestServerPersistedReview(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	client := &AnalyzerClientMock{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: client,
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, store.NewMemEventOperator(), &store.NoopCommentOperator{})
	watcher.Watch(context.TODO(), srv.HandleEvent)

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

func TestServerIncrementalReview(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	client := &AnalyzerSameCommentClient{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: client,
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, store.NewMemEventOperator(), store.NewMemCommentOperator())
	watcher.Watch(context.TODO(), srv.HandleEvent)

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)

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

func TestAnalyzerConfigDisabled(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: &AnalyzerClientMock{},
			Config: lookout.AnalyzerConfig{
				Disabled: true,
			},
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, &store.NoopEventOperator{}, &store.NoopCommentOperator{})
	watcher.Watch(context.TODO(), srv.HandleEvent)

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 0)

	status := poster.PopStatus()
	require.Equal(lookout.SuccessAnalysisStatus, status)
}

var globalConfig = lookout.AnalyzerConfig{
	Name: "test",
	Settings: map[string]interface{}{
		"key_from_global": 1,
	},
}

func TestMergeConfigWithoutLocal(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzerClient := &AnalyzerClientMock{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: analyzerClient,
			Config: globalConfig,
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, &store.NoopEventOperator{}, &store.NoopCommentOperator{})
	watcher.Watch(context.TODO(), srv.HandleEvent)

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := analyzerClient.PopReviewEvents()
	require.Len(es, 1)

	require.Equal(grpchelper.ToPBStruct(globalConfig.Settings), &es[0].Configuration)
}

func TestMergeConfigWithLocal(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMockWithConfig{
		content: `analyzers:
 - name: mock
   settings:
     some: value
`,
	}
	analyzerClient := &AnalyzerClientMock{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: analyzerClient,
			Config: globalConfig,
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, &store.NoopEventOperator{}, &store.NoopCommentOperator{})
	watcher.Watch(context.TODO(), srv.HandleEvent)

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := analyzerClient.PopReviewEvents()
	require.Len(es, 1)

	expectedMap := make(map[string]interface{})
	for k, v := range globalConfig.Settings {
		expectedMap[k] = v
	}
	expectedMap["some"] = "value"

	require.Equal(grpchelper.ToPBStruct(expectedMap), &es[0].Configuration)
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

func setupMockedServer() (*WatcherMock, *PosterMock) {
	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzers := map[string]lookout.Analyzer{
		"mock": lookout.Analyzer{
			Client: &AnalyzerClientMock{},
		},
	}

	srv := NewServer(poster, fileGetter, analyzers, &store.NoopEventOperator{}, &store.NoopCommentOperator{})
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

func (p *PosterMock) Post(_ context.Context, e lookout.Event, aCommentsList []lookout.AnalyzerComments) error {
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
	reviewEvents []*lookout.ReviewEvent
}

func (a *AnalyzerClientMock) NotifyReviewEvent(ctx context.Context, in *lookout.ReviewEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	a.reviewEvents = append(a.reviewEvents, in)
	return &lookout.EventResponse{
		Comments: []*lookout.Comment{
			makeComment(in.CommitRevision.Base, in.CommitRevision.Head),
		},
	}, nil
}

func (a *AnalyzerClientMock) NotifyPushEvent(ctx context.Context, in *lookout.PushEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	return &lookout.EventResponse{
		Comments: []*lookout.Comment{
			makeComment(in.CommitRevision.Base, in.CommitRevision.Head),
		},
	}, nil
}

func (a *AnalyzerClientMock) PopReviewEvents() []*lookout.ReviewEvent {
	res := a.reviewEvents[:]
	a.reviewEvents = []*lookout.ReviewEvent{}
	return res
}

func makeComment(from, to lookout.ReferencePointer) *lookout.Comment {
	return &lookout.Comment{
		Text: fmt.Sprintf("%s > %s", from.Hash, to.Hash),
	}
}

type AnalyzerSameCommentClient struct {
	reviewEvents []*lookout.ReviewEvent
}

func (a *AnalyzerSameCommentClient) NotifyReviewEvent(ctx context.Context, in *lookout.ReviewEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	a.reviewEvents = append(a.reviewEvents, in)
	return &lookout.EventResponse{
		Comments: []*lookout.Comment{
			{Text: "some-text"},
		},
	}, nil
}

func (a *AnalyzerSameCommentClient) NotifyPushEvent(ctx context.Context, in *lookout.PushEvent, opts ...grpc.CallOption) (*lookout.EventResponse, error) {
	return &lookout.EventResponse{
		Comments: []*lookout.Comment{
			{Text: "some-text"},
		},
	}, nil
}

func (a *AnalyzerSameCommentClient) PopReviewEvents() []*lookout.ReviewEvent {
	res := a.reviewEvents[:]
	a.reviewEvents = []*lookout.ReviewEvent{}
	return res
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
