package github

import (
	"context"
	"net/http"
	"testing"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

var (
	hash1 = "f67e5455a86d0f2a366f1b980489fac77a373bd0"
	hash2 = "02801e1a27a0a906d59530aeb81f4cd137f2c717"
	base1 = plumbing.ReferenceName("base")
	head1 = plumbing.ReferenceName("refs/pull/42/head")
)

func TestPoster_Post_OK(t *testing.T) {
	require := require.New(t)

	mcc := &commitsComparator{}
	mcc.On(
		"CompareCommits",
		mock.Anything, "foo", "bar", hash1, hash2).Once().Return(
		&github.CommitsComparison{
			Files: []github.CommitFile{github.CommitFile{
				Filename: strptr("main.go"),
				Patch:    strptr("@@ -3,10 +3,10 @@"),
			}}},
		&github.Response{Response: &http.Response{StatusCode: 200}},
		nil)

	mrc := &reviewCreator{}
	mrc.On(
		"CreateReview",
		mock.Anything, "foo", "bar", 42,
		&github.PullRequestReviewRequest{
			Body:  strptr("Global comment\n\nAnother global comment"),
			Event: strptr("APPROVE"),
			Comments: []*github.DraftReviewComment{&github.DraftReviewComment{
				Path:     strptr("main.go"),
				Body:     strptr("File comment"),
				Position: intptr(1),
			}, &github.DraftReviewComment{
				Path:     strptr("main.go"),
				Position: intptr(3),
				Body:     strptr("Line comment"),
			}}},
	).Once().Return(
		nil,
		&github.Response{Response: &http.Response{StatusCode: 200}},
		nil)

	p := &Poster{rc: mrc, cc: mcc}
	ev := &lookout.ReviewEvent{
		Provider: Provider,
		CommitRevision: lookout.CommitRevision{
			Base: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
				ReferenceName:         base1,
				Hash:                  hash1,
			},
			Head: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
				ReferenceName:         head1,
				Hash:                  hash2,
			}}}
	cs := []*lookout.Comment{&lookout.Comment{
		Text: "Global comment",
	}, &lookout.Comment{
		File: "main.go",
		Text: "File comment",
	}, &lookout.Comment{
		File: "main.go",
		Line: 5,
		Text: "Line comment",
	}, &lookout.Comment{
		Text: "Another global comment",
	}}

	err := p.Post(context.Background(), ev, cs)
	require.NoError(err)

	mcc.AssertExpectations(t)
	mrc.AssertExpectations(t)
}

func TestPoster_Post_BadProvider(t *testing.T) {
	require := require.New(t)

	mcc := &commitsComparator{}
	mrc := &reviewCreator{}
	p := &Poster{rc: mrc, cc: mcc}
	ev := &lookout.ReviewEvent{
		Provider: "badprovider",
		CommitRevision: lookout.CommitRevision{
			Base: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
			}}}
	cs := []*lookout.Comment{&lookout.Comment{
		Text: "Global comment",
	}, &lookout.Comment{
		File: "main.go",
		Text: "File comment",
	}, &lookout.Comment{
		File: "main.go",
		Line: 5,
		Text: "Line comment",
	}, &lookout.Comment{
		Text: "Another global comment",
	}}

	err := p.Post(context.Background(), ev, cs)
	require.True(ErrEventNotSupported.Is(err))
	require.Equal(
		"event not supported: unsupported provider: badprovider", err.Error())

	mcc.AssertExpectations(t)
	mrc.AssertExpectations(t)
}

func TestPoster_Post_BadReferenceNoRepository(t *testing.T) {
	require := require.New(t)

	mcc := &commitsComparator{}
	mrc := &reviewCreator{}
	p := &Poster{rc: mrc, cc: mcc}
	ev := &lookout.ReviewEvent{
		Provider: Provider,
	}
	cs := []*lookout.Comment{&lookout.Comment{
		Text: "Global comment",
	}, &lookout.Comment{
		File: "main.go",
		Text: "File comment",
	}, &lookout.Comment{
		File: "main.go",
		Line: 5,
		Text: "Line comment",
	}, &lookout.Comment{
		Text: "Another global comment",
	}}

	err := p.Post(context.Background(), ev, cs)
	require.True(ErrEventNotSupported.Is(err))
	require.Equal(
		"event not supported: nil repository", err.Error())

	mcc.AssertExpectations(t)
	mrc.AssertExpectations(t)
}

func TestPoster_Post_BadReference(t *testing.T) {
	require := require.New(t)

	mcc := &commitsComparator{}
	mrc := &reviewCreator{}
	p := &Poster{rc: mrc, cc: mcc}
	ev := &lookout.ReviewEvent{
		Provider: Provider,
		CommitRevision: lookout.CommitRevision{
			Base: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
			},
			Head: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
				ReferenceName:         plumbing.ReferenceName("BAD"),
			}}}
	cs := []*lookout.Comment{&lookout.Comment{
		Text: "Global comment",
	}, &lookout.Comment{
		File: "main.go",
		Text: "File comment",
	}, &lookout.Comment{
		File: "main.go",
		Line: 5,
		Text: "Line comment",
	}, &lookout.Comment{
		Text: "Another global comment",
	}}

	err := p.Post(context.Background(), ev, cs)
	require.True(ErrEventNotSupported.Is(err))
	require.Equal(
		"event not supported: bad PR: BAD", err.Error())

	mcc.AssertExpectations(t)
	mrc.AssertExpectations(t)
}

func strptr(v string) *string {
	return &v
}

func intptr(v int) *int {
	return &v
}

type reviewCreator struct {
	mock.Mock
}

func (m *reviewCreator) CreateReview(ctx context.Context, owner, repo string,
	number int, review *github.PullRequestReviewRequest) (
	*github.PullRequestReview, *github.Response, error) {

	args := m.Called(ctx, owner, repo, number, review)

	var (
		r0 *github.PullRequestReview
		r1 *github.Response
	)

	if v := args.Get(0); v != nil {
		r0 = v.(*github.PullRequestReview)
	}

	if v := args.Get(1); v != nil {
		r1 = v.(*github.Response)
	}

	return r0, r1, args.Error(2)
}

type commitsComparator struct {
	mock.Mock
}

func (m *commitsComparator) CompareCommits(ctx context.Context,
	owner, repo string, base, head string) (
	*github.CommitsComparison, *github.Response, error) {

	args := m.Called(ctx, owner, repo, base, head)

	var (
		r0 *github.CommitsComparison
		r1 *github.Response
	)

	if v := args.Get(0); v != nil {
		r0 = v.(*github.CommitsComparison)
	}

	if v := args.Get(1); v != nil {
		r1 = v.(*github.Response)
	}

	return r0, r1, args.Error(2)
}
