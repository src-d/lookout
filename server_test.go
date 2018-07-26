package lookout

import (
	"context"
	"fmt"
	"testing"

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

func TestServerReview(t *testing.T) {
	require := require.New(t)

	watcher, poster := setupMockedServer()

	reviewEvent := &correctReviewEvent

	err := watcher.Send(reviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 1)
	require.Equal(makeComment(reviewEvent.CommitRevision.Base, reviewEvent.CommitRevision.Head), comments[0])
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
}

func TestAnalyzerConfigDisabled(t *testing.T) {
	require := require.New(t)

	log.DefaultLogger = log.New(log.Fields{"app": "lookout"})

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: &AnalyzerClientMock{},
			Config: AnalyzerConfig{},
		},
	}

	srv := NewServer(watcher, poster, fileGetter, analyzers)
	srv.Run(context.TODO())

	err := watcher.Send(&correctReviewEvent)
	require.Nil(err)

	comments := poster.PopComments()
	require.Len(comments, 0)
}

func setupMockedServer() (*WatcherMock, *PosterMock) {
	log.DefaultLogger = log.New(log.Fields{"app": "lookout"})

	watcher := &WatcherMock{}
	poster := &PosterMock{}
	fileGetter := &FileGetterMock{}
	analyzers := map[string]Analyzer{
		"mock": Analyzer{
			Client: &AnalyzerClientMock{},
			Config: AnalyzerConfig{
				Enabled: true,
			},
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

type PosterMock struct {
	comments []*Comment
}

func (p *PosterMock) Post(_ context.Context, e Event, cs []*Comment) error {
	p.comments = cs
	return nil
}

func (p *PosterMock) PopComments() []*Comment {
	cs := p.comments[:]
	p.comments = []*Comment{}
	return cs
}

type FileGetterMock struct {
}

func (g *FileGetterMock) GetFiles(_ context.Context, req *FilesRequest) (FileScanner, error) {
	return &NoopFileScanner{}, nil
}

type AnalyzerClientMock struct {
}

func (a *AnalyzerClientMock) NotifyReviewEvent(ctx context.Context, in *ReviewEvent, opts ...grpc.CallOption) (*EventResponse, error) {
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
