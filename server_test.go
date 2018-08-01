package lookout

import (
	"context"
	"fmt"
	"testing"

	"github.com/src-d/lookout/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
)

var correctReviewEvent = ReviewEvent{
	Provider:    "Mock",
	InternalID:  "internal-id",
	IsMergeable: true,
	Source: ReferencePointer{
		InternalRepositoryURL: "file:///test",
		ReferenceName:         "feature",
		Hash:                  "source-hash",
	},
	Merge: ReferencePointer{
		InternalRepositoryURL: "file:///test",
		ReferenceName:         "merge-branch",
		Hash:                  "merge-hash",
	},
	CommitRevision: CommitRevision{
		Base: ReferencePointer{
			InternalRepositoryURL: "file:///test",
			ReferenceName:         "master",
			Hash:                  "base-hash",
		},
		Head: ReferencePointer{
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
	require.Equal(SuccessAnalysisStatus, status)
}

func TestServerPush(t *testing.T) {
	require := require.New(t)

	watcher, poster := setupMockedServer()

	pushEvent := &PushEvent{
		Provider:   "Mock",
		InternalID: "internal-id",
		CommitRevision: CommitRevision{
			Base: ReferencePointer{
				InternalRepositoryURL: "file:///test",
				ReferenceName:         "master",
				Hash:                  "base-hash",
			},
			Head: ReferencePointer{
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
	require.Equal(SuccessAnalysisStatus, status)
}

func TestAnalyzerConfigDisabled(t *testing.T) {
	require := require.New(t)

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: &AnalyzerClientMock{},
			Config: AnalyzerConfig{
				Disabled: true,
			},
		},
	}

	srv := NewServer(watcher, poster, fileGetter, analyzers)
	srv.Run(context.TODO())

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 0)

	status := poster.PopStatus()
	require.Equal(SuccessAnalysisStatus, status)
}

var globalConfig = AnalyzerConfig{
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
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: analyzerClient,
			Config: globalConfig,
		},
	}

	srv := NewServer(watcher, poster, fileGetter, analyzers)
	srv.Run(context.TODO())

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := analyzerClient.PopReviewEvents()
	require.Len(es, 1)

	require.Equal(pb.ToStruct(globalConfig.Settings), &es[0].Configuration)
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
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: analyzerClient,
			Config: globalConfig,
		},
	}

	srv := NewServer(watcher, poster, fileGetter, analyzers)
	srv.Run(context.TODO())

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	es := analyzerClient.PopReviewEvents()
	require.Len(es, 1)

	expectedMap := make(map[string]interface{})
	for k, v := range globalConfig.Settings {
		expectedMap[k] = v
	}
	expectedMap["some"] = "value"

	require.Equal(pb.ToStruct(expectedMap), &es[0].Configuration)
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
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: &AnalyzerClientMock{},
		},
	}

	srv := NewServer(watcher, poster, fileGetter, analyzers)
	srv.Run(context.TODO())

	return watcher, poster
}

type WatcherMock struct {
	handler EventHandler
}

func (w *WatcherMock) Watch(ctx context.Context, e EventHandler) error {
	w.handler = e
	return nil
}

func (w *WatcherMock) Send(e Event) error {
	return w.handler(e)
}

var _ Poster = &PosterMock{}

type PosterMock struct {
	comments []*Comment
	status   AnalysisStatus
}

func (p *PosterMock) Post(_ context.Context, e Event, aCommentsList []AnalyzerComments) error {
	cs := make([]*Comment, 0)
	for _, aComments := range aCommentsList {
		cs = append(cs, aComments.Comments...)
	}
	p.comments = cs
	return nil
}

func (p *PosterMock) PopComments() []*Comment {
	cs := p.comments[:]
	p.comments = []*Comment{}
	return cs
}

func (p *PosterMock) Status(_ context.Context, e Event, st AnalysisStatus) error {
	p.status = st
	return nil
}

func (p *PosterMock) PopStatus() AnalysisStatus {
	st := p.status
	p.status = 0
	return st
}

type FileGetterMock struct {
}

func (g *FileGetterMock) GetFiles(_ context.Context, req *FilesRequest) (FileScanner, error) {
	return &NoopFileScanner{}, nil
}

type FileGetterMockWithConfig struct {
	content string
}

func (g *FileGetterMockWithConfig) GetFiles(_ context.Context, req *FilesRequest) (FileScanner, error) {
	if req.IncludePattern == `^\.lookout\.yml$` {
		return &SliceFileScanner{Files: []*File{{
			Path:    ".lookout.yml",
			Content: []byte(g.content),
		}}}, nil
	}
	return &NoopFileScanner{}, nil
}

type AnalyzerClientMock struct {
	reviewEvents []*ReviewEvent
}

func (a *AnalyzerClientMock) NotifyReviewEvent(ctx context.Context, in *ReviewEvent, opts ...grpc.CallOption) (*EventResponse, error) {
	a.reviewEvents = append(a.reviewEvents, in)
	return &EventResponse{
		Comments: []*Comment{
			makeComment(in.CommitRevision.Base, in.CommitRevision.Head),
		},
	}, nil
}

func (a *AnalyzerClientMock) NotifyPushEvent(ctx context.Context, in *PushEvent, opts ...grpc.CallOption) (*EventResponse, error) {
	return &EventResponse{
		Comments: []*Comment{
			makeComment(in.CommitRevision.Base, in.CommitRevision.Head),
		},
	}, nil
}

func (a *AnalyzerClientMock) PopReviewEvents() []*ReviewEvent {
	res := a.reviewEvents[:]
	a.reviewEvents = []*ReviewEvent{}
	return res
}

func makeComment(from, to ReferencePointer) *Comment {
	return &Comment{
		Text: fmt.Sprintf("%s > %s", from.Hash, to.Hash),
	}
}

type NoopFileScanner struct {
}

func (s *NoopFileScanner) Next() bool {
	return false
}

func (s *NoopFileScanner) Err() error {
	return nil
}

func (s *NoopFileScanner) File() *File {
	return nil
}

func (s *NoopFileScanner) Close() error {
	return nil
}
