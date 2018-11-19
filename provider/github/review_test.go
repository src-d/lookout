package github

import (
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestMergeComments(t *testing.T) {
	require := require.New(t)

	input := []*github.DraftReviewComment{
		{
			Path:     strptr("file1"),
			Position: intptr(1),
			Body:     strptr("comment 1_1"),
		},
		{
			Path:     strptr("file2"),
			Position: intptr(1),
			Body:     strptr("comment 2_1"),
		},
		{
			Path:     strptr("file1"),
			Position: intptr(1),
			Body:     strptr("comment 1_2"),
		},
		{
			Path:     strptr("file1"),
			Position: intptr(2),
			Body:     strptr("comment 1_3"),
		},
	}
	output := mergeComments(input)

	require.Len(output, 3)
	require.Equal("comment 1_1"+commentsSeparator+"comment 1_2", output[0].GetBody())
	require.Equal("comment 1_3", output[1].GetBody())
	require.Equal("comment 2_1", output[2].GetBody())
}

func TestFilterPostedComments(t *testing.T) {
	require := require.New(t)

	input := []*github.DraftReviewComment{
		{
			Path:     strptr("file1"),
			Position: intptr(1),
			Body:     strptr("regular filter out"),
		},
		{
			Path:     strptr("file2"),
			Position: intptr(1),
			Body:     strptr("should stay"),
		},
		{
			Path:     strptr("file1"),
			Position: intptr(2),
			Body:     strptr("merged filter out"),
		},
	}
	posted := []*github.PullRequestComment{
		{
			Path:     strptr("file1"),
			Position: intptr(1),
			Body:     strptr("regular filter out"),
		},
		{
			Path:     strptr("file1"),
			Position: intptr(2),
			Body:     strptr("merged filter out" + commentsSeparator + "another comment"),
		},
	}
	output := filterPostedComments(input, posted)

	require.Len(output, 1)
	require.Equal("should stay", output[0].GetBody())
}

func TestSplitReviewRequest(t *testing.T) {
	require := require.New(t)

	n := 2

	rw := &github.PullRequestReviewRequest{
		Event: strptr(commentEvent),
		Body:  strptr("body"),
	}

	rw.Comments = []*github.DraftReviewComment{
		{Body: strptr("comment1")},
	}

	r := splitReviewRequest(rw, n)
	require.Len(r, 1)
	require.Equal([]*github.PullRequestReviewRequest{rw}, r)

	rw.Comments = []*github.DraftReviewComment{
		{Body: strptr("comment1")},
		{Body: strptr("comment2")},
		{Body: strptr("comment3")},
	}

	r = splitReviewRequest(rw, n)
	require.Len(r, 2)
	require.Equal([]*github.PullRequestReviewRequest{
		{
			Event: strptr(commentEvent),
			Body:  strptr(""),
			Comments: []*github.DraftReviewComment{
				{Body: strptr("comment1")},
				{Body: strptr("comment2")},
			},
		},
		{
			Event: strptr(commentEvent),
			Body:  strptr("body"),
			Comments: []*github.DraftReviewComment{
				{Body: strptr("comment3")},
			},
		},
	}, r)

	rw.Comments = []*github.DraftReviewComment{
		{Body: strptr("comment1")},
		{Body: strptr("comment2")},
		{Body: strptr("comment3")},
		{Body: strptr("comment4")},
		{Body: strptr("comment5")},
		{Body: strptr("comment6")},
	}

	r = splitReviewRequest(rw, n)
	require.Len(r, 3)
}
