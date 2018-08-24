package github

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestParseHunks(t *testing.T) {
	require := require.New(t)

	hunks, err := parseHunks("@@ -0,0 +1 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 0,
		OldLines:     0,
		NewStartLine: 1,
		NewLines:     1,
	}}, hunks)

	hunks, err = parseHunks("@@ -1 +1,3 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 1,
		OldLines:     1,
		NewStartLine: 1,
		NewLines:     3,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@ module Test\n@@ -1000,7 +1000,7 @@ module Test")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 1000,
		OldLines:     7,
		NewStartLine: 1000,
		NewLines:     7,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@\n@@ -1000,7 +1000,7 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 1000,
		OldLines:     7,
		NewStartLine: 1000,
		NewLines:     7,
	}}, hunks)

	hunks, err = parseHunks(`@@ -132,7 +132,7 @@
 context-line
-delete-line
+insert-line`)
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 1},
			{Type: lineDeleted, Lines: 1},
			{Type: lineAdded, Lines: 1},
		},
	}}, hunks)

	hunks, err = parseHunks(`@@ -132,7 +132,7 @@
 context-line
 context-line
-delete-line
-delete-line
 context-line
 context-line`)
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 2},
			{Type: lineDeleted, Lines: 2},
			{Type: lineContext, Lines: 2},
		},
	}}, hunks)
	hunks, err = parseHunks(`@@ -132,7 +132,7 @@
 context-line
 context-line
-delete-line
-delete-line
 context-line
 context-line
@@ -132,7 +132,7 @@
 context-line
 context-line
-delete-line
-delete-line
 context-line
 context-line`)
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 2},
			{Type: lineDeleted, Lines: 2},
			{Type: lineContext, Lines: 2},
		},
	}, &hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 2},
			{Type: lineDeleted, Lines: 2},
			{Type: lineContext, Lines: 2},
		},
	}}, hunks)
}

func TestConvertRanges(t *testing.T) {
	require := require.New(t)

	ranges := convertRanges([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks:       []linesChunk{{Type: lineAdded, Lines: 7}},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 132, AbsEnd: 139,
		RelStart: 1, RelEnd: 8,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 100,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
		Chunks:       []linesChunk{{Type: lineAdded, Lines: 7}},
	}, &hunk{
		OldStartLine: 200,
		OldLines:     10,
		NewStartLine: 500,
		NewLines:     10,
		Chunks:       []linesChunk{{Type: lineAdded, Lines: 10}},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 132, AbsEnd: 139,
		RelStart: 1, RelEnd: 8,
	}, &posRange{
		AbsStart: 500, AbsEnd: 510,
		RelStart: 9, RelEnd: 19,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 0,
		OldLines:     0,
		NewStartLine: 1,
		NewLines:     446,
		Chunks:       []linesChunk{{Type: lineAdded, Lines: 446}},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 1, AbsEnd: 447,
		RelStart: 1, RelEnd: 447,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 5,
		OldLines:     6,
		NewStartLine: 5,
		NewLines:     8,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 3},
			{Type: lineAdded, Lines: 2},
			{Type: lineContext, Lines: 3},
		},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 5, AbsEnd: 13,
		RelStart: 1, RelEnd: 9,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 5,
		OldLines:     6,
		NewStartLine: 5,
		NewLines:     8,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 3},
			{Type: lineAdded, Lines: 2},
			{Type: lineContext, Lines: 3},
		},
	}, &hunk{
		OldStartLine: 20,
		OldLines:     6,
		NewStartLine: 22,
		NewLines:     8,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 3},
			{Type: lineAdded, Lines: 2},
			{Type: lineContext, Lines: 3},
		},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 5, AbsEnd: 13,
		RelStart: 1, RelEnd: 9,
	}, &posRange{
		AbsStart: 22, AbsEnd: 30,
		RelStart: 10, RelEnd: 18,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 5,
		OldLines:     6,
		NewStartLine: 5,
		NewLines:     8,
		Chunks: []linesChunk{
			{Type: lineContext, Lines: 3},
			{Type: lineDeleted, Lines: 2},
			{Type: lineContext, Lines: 3},
		},
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 5, AbsEnd: 8,
		RelStart: 1, RelEnd: 4,
	}, &posRange{
		AbsStart: 8, AbsEnd: 11,
		RelStart: 6, RelEnd: 9,
	}}, ranges)
}

func TestConvertLines(t *testing.T) {
	require := require.New(t)

	filename := "some_file"

	// only insert
	strHunk1 := `@@ -5,6 +5,8 @@ header-line
 context-line1
 context-line2
 context-line3
+new-line1
+new-line2
 context-line4
 context-line5
 context-line6`
	// only delete
	strHunk2 := `@@ -20,8 +22,6 @@ header-line
 context-line1
 context-line2
 context-line3
-old-line1
-old-line2
 context-line4
 context-line5
 context-line6`
	// delete and insert
	strHunk3 := `@@ -35,7 +35,7 @@ header-line
 context-line1
 context-line2
 context-line3
-delete line
+insert line
 context-line4
 context-line5
 context-line6`
	// multiple delete and insert
	strHunk4 := `@@ -50,11 +50,12 @@ header-line
 context-line1
 context-line2
 context-line3
-delete line1
+insert line1
+insert line2
 context-line4
 context-line5
 context-line6
-delete line2
+insert line3
 context-line7
 context-line8
 context-line9`
	patch := strHunk1 + "\n" + strHunk2 + "\n" + strHunk3 + "\n" + strHunk4

	cc := &github.CommitsComparison{
		Files: []github.CommitFile{
			{
				Filename: &filename,
				Patch:    &patch,
			},
		},
	}
	dl := newDiffLines(cc)

	commentLines := map[int]*int{
		// out of range
		1: nil,
		// comment on new line in first hunk
		8: intPointer(4),
		// out of range between hunks
		15: nil,
		// comment on context line before delete in second hunk
		23: intPointer(11),
		// comment on context line after delete in second hunk
		25: intPointer(15),
		// comment on insert in 3rd hunk
		38: intPointer(23),
		// comment on first insert in 4th hunk
		53: intPointer(32),
		// comment on second insert in 4th hunk
		58: intPointer(38),
		// out of range
		100: nil,
	}

	for line, expected := range commentLines {
		newLine, err := dl.ConvertLine(filename, int(line))
		if expected == nil {
			require.EqualError(err, "line number is not in diff", fmt.Sprintf("old line %d, new line %d", line, newLine))
		} else {
			require.Equal(newLine, *expected, fmt.Sprintf("old line: %d", line))
		}
	}
}

func intPointer(v int) *int {
	return &v
}

func TestConvertLines_issue173(t *testing.T) {
	require := require.New(t)

	filename := "cmd/lookout/serve.go"
	patch := "@@ -91,6 +91,21 @@ func (c *ServeCommand) Execute(args []string) error {\n \t\treturn err\n \t}\n \n+\tdb, err := c.initDB()\n+\tif err != nil {\n+\t\treturn fmt.Errorf(\"Can't connect to the DB: %s\", err)\n+\t}\n+\n+\treviewStore := models.NewReviewEventStore(db)\n+\teventOp := store.NewDBEventOperator(\n+\t\treviewStore,\n+\t\tmodels.NewPushEventStore(db),\n+\t)\n+\tcommentsOp := store.NewDBCommentOperator(\n+\t\tmodels.NewCommentStore(db),\n+\t\treviewStore,\n+\t)\n+\n \tanalyzers := make(map[string]lookout.Analyzer)\n \tfor _, aConf := range conf.Analyzers {\n \t\tif aConf.Disabled {\n@@ -121,21 +136,6 @@ func (c *ServeCommand) Execute(args []string) error {\n \t\treturn err\n \t}\n \n-\tdb, err := c.initDB()\n-\tif err != nil {\n-\t\treturn err\n-\t}\n-\n-\treviewStore := models.NewReviewEventStore(db)\n-\teventOp := store.NewDBEventOperator(\n-\t\treviewStore,\n-\t\tmodels.NewPushEventStore(db),\n-\t)\n-\tcommentsOp := store.NewDBCommentOperator(\n-\t\tmodels.NewCommentStore(db),\n-\t\treviewStore,\n-\t)\n-\n \tctx := context.Background()\n \treturn server.NewServer(watcher, poster, dataHandler.FileGetter, analyzers, eventOp, commentsOp).Run(ctx)\n }"

	cc := &github.CommitsComparison{
		Files: []github.CommitFile{
			{
				Filename: &filename,
				Patch:    &patch,
			},
		},
	}
	dl := newDiffLines(cc)

	commentLines := []int{
		44,
		45,
		46,
		47,
		48,
		49,
		50,
		51,
		140,
		167,
		222,
		249,
		306,
		320,
		321,
	}
	for _, line := range commentLines {
		newLine, err := dl.ConvertLine(filename, int(line))
		if line == 140 {
			require.Equal(42, newLine)
		} else {
			require.EqualError(err, "line number is not in diff", fmt.Sprintf("old line %d, new line %d", line, newLine))
		}
	}
}
