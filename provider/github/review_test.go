package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v24/github"
	"github.com/src-d/lookout"
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

func TestConvertCommentsOutOfRange(t *testing.T) {
	require := require.New(t)

	dl := newDiffLines(&github.CommitsComparison{
		Files: []github.CommitFile{github.CommitFile{
			Filename: strptr("main.go"),
			Patch:    strptr(mockedPatch),
		}}})

	input := []*lookout.Comment{
		&lookout.Comment{
			File: "main.go",
			Line: 1,
			Text: "out of range comment before",
		},
		&lookout.Comment{
			Text: "Body comment",
		},
		&lookout.Comment{
			File: "main.go",
			Line: 3,
			Text: "Line comment",
		},
		&lookout.Comment{
			File: "main.go",
			Line: 205,
			Text: "out of range comment after",
		}}

	bodyComments, ghComments := convertComments(context.TODO(), input, dl)

	require.Len(bodyComments, 1)
	require.Len(ghComments, 1)

	require.Equal([]*github.DraftReviewComment{&github.DraftReviewComment{
		Path:     strptr("main.go"),
		Position: intptr(1),
		Body:     strptr("Line comment"),
	}}, ghComments)
}

func TestConvertCommentsWrongFile(t *testing.T) {
	require := require.New(t)

	dl := newDiffLines(&github.CommitsComparison{
		Files: []github.CommitFile{github.CommitFile{
			Filename: strptr("main.go"),
			Patch:    strptr(mockedPatch),
		}}})

	input := []*lookout.Comment{
		&lookout.Comment{
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
		}, &lookout.Comment{
			File: "file-does-not-exist.txt",
			Line: 5,
			Text: "Line comment",
		}}

	bodyComments, ghComments := convertComments(context.TODO(), input, dl)

	require.Len(bodyComments, 2)
	require.Len(ghComments, 2)

	require.Equal([]string{
		"Global comment",
		"Another global comment",
	}, bodyComments)

	require.Equal([]*github.DraftReviewComment{&github.DraftReviewComment{
		Path:     strptr("main.go"),
		Body:     strptr("File comment"),
		Position: intptr(1),
	}, &github.DraftReviewComment{
		Path:     strptr("main.go"),
		Position: intptr(3),
		Body:     strptr("Line comment"),
	}}, ghComments)
}

func TestCouldNotExecuteFooterTemplate(t *testing.T) {
	require := require.New(t)

	unkonwnDataTemplate, err := newFooterTemplate("Old template {{.UnknownData}}")
	require.Nil(err)
	commentsWrongTemplate := addFootnote(context.TODO(), "comments", unkonwnDataTemplate, nil)
	require.Equal("comments", commentsWrongTemplate)
}
