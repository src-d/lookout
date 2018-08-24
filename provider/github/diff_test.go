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

const largeHunk = "@@ -6,20 +6,21 @@ import (\n \t\"database/sql\"\n \t\"fmt\"\n \t\"io\"\n-\t\"io/ioutil\"\n \t\"os\"\n \t\"os/exec\"\n+\t\"strings\"\n \t\"time\"\n \n \t_ \"github.com/lib/pq\"\n+\t\"github.com/stretchr/testify/require\"\n )\n \n // CmdTimeout defines timeout for a command\n var CmdTimeout = time.Minute\n \n // default path to binaries\n-var dummyBin = \"build/bin/dummy\"\n-var lookoutBin = \"build/bin/lookout\"\n+var dummyBin = \"../../build/bin/dummy\"\n+var lookoutBin = \"../../build/bin/lookout\"\n \n // function to stop running commands\n // redefined in StoppableCtx\n@@ -50,7 +51,7 @@ func StoppableCtx() (context.Context, func()) {\n }\n \n // StartDummy starts dummy analyzer with context and optional arguments\n-func StartDummy(ctx context.Context, args ...string) io.Reader {\n+func StartDummy(ctx context.Context, require *require.Assertions, args ...string) io.Reader {\n \tr, outputWriter := io.Pipe()\n \tbuf := \u0026bytes.Buffer{}\n \ttee := io.TeeReader(r, buf)\n@@ -61,31 +62,25 @@ func StartDummy(ctx context.Context, args ...string) io.Reader {\n \tcmd.Stdout = outputWriter\n \tcmd.Stderr = outputWriter\n \terr := cmd.Start()\n-\tif err != nil {\n-\t\tioutil.ReadAll(tee)\n-\t\tfmt.Println(\"can't start analyzer:\")\n-\t\tfmt.Println(err)\n-\t\tfmt.Printf(\"output:\\n %s\", buf.String())\n-\t\tos.Exit(1)\n-\t} else {\n-\t\tgo func() {\n-\t\t\tif err := cmd.Wait(); err != nil {\n-\t\t\t\t// don't print error if analyzer was killed by cancel\n-\t\t\t\tif ctx.Err() != context.Canceled {\n-\t\t\t\t\tioutil.ReadAll(tee)\n-\t\t\t\t\tfmt.Println(\"analyzer exited with error:\", err)\n-\t\t\t\t\tfmt.Printf(\"output:\\n%s\", buf.String())\n-\t\t\t\t\tfailExit()\n-\t\t\t\t}\n+\trequire.NoError(err, \"can't start analyzer\")\n+\n+\tgo func() {\n+\t\tif err := cmd.Wait(); err != nil {\n+\t\t\t// don't print error if analyzer was killed by cancel\n+\t\t\tif ctx.Err() != context.Canceled {\n+\t\t\t\tfmt.Println(\"analyzer exited with error:\", err)\n+\t\t\t\tfmt.Printf(\"output:\\n%s\", buf.String())\n+\t\t\t\t// T.Fail cannot be called from a goroutine\n+\t\t\t\tfailExit()\n \t\t\t}\n-\t\t}()\n-\t}\n+\t\t}\n+\t}()\n \n \treturn tee\n }\n \n // StartServe starts lookout server with context and optional arguments\n-func StartServe(ctx context.Context, args ...string) (io.Reader, io.WriteCloser) {\n+func StartServe(ctx context.Context, require *require.Assertions, args ...string) (io.Reader, io.WriteCloser) {\n \tr, outputWriter := io.Pipe()\n \tbuf := \u0026bytes.Buffer{}\n \ttee := io.TeeReader(r, buf)\n@@ -97,85 +92,59 @@ func StartServe(ctx context.Context, args ...string) (io.Reader, io.WriteCloser)\n \tcmd.Stderr = outputWriter\n \n \tw, err := cmd.StdinPipe()\n-\tif err != nil {\n-\t\tfmt.Println(\"can't start server:\")\n-\t\tfmt.Println(err)\n-\t\tos.Exit(1)\n-\t}\n+\trequire.NoError(err, \"can't start server\")\n \n \terr = cmd.Start()\n-\tif err != nil {\n-\t\tioutil.ReadAll(tee)\n-\t\tfmt.Println(\"can't start server:\")\n-\t\tfmt.Println(err)\n-\t\tfmt.Printf(\"output:\\n %s\", buf.String())\n-\t\tos.Exit(1)\n-\t} else {\n-\t\tgo func() {\n-\t\t\tif err := cmd.Wait(); err != nil {\n-\t\t\t\t// don't print error if analyzer was killed by cancel\n-\t\t\t\tif ctx.Err() != context.Canceled {\n-\t\t\t\t\tioutil.ReadAll(tee)\n-\t\t\t\t\tfmt.Println(\"server exited with error:\", err)\n-\t\t\t\t\tfmt.Printf(\"output:\\n%s\", buf.String())\n-\t\t\t\t\tfailExit()\n-\t\t\t\t}\n+\trequire.NoError(err, \"can't start server\")\n+\n+\tgo func() {\n+\t\tif err := cmd.Wait(); err != nil {\n+\t\t\t// don't print error if analyzer was killed by cancel\n+\t\t\tif ctx.Err() != context.Canceled {\n+\t\t\t\tfmt.Println(\"server exited with error:\", err)\n+\t\t\t\tfmt.Printf(\"output:\\n%s\", buf.String())\n+\t\t\t\t// T.Fail cannot be called from a goroutine\n+\t\t\t\tfailExit()\n \t\t\t}\n-\t\t}()\n-\t}\n+\t\t}\n+\t}()\n \n \treturn tee, w\n }\n \n // RunCli runs lookout subcommand (not a server)\n-func RunCli(ctx context.Context, cmd string, args ...string) io.Reader {\n+func RunCli(ctx context.Context, require *require.Assertions, cmd string, args ...string) io.Reader {\n \targs = append([]string{cmd}, args...)\n \n \tvar out bytes.Buffer\n-\treviewCmd := exec.CommandContext(ctx, lookoutBin, args...)\n-\treviewCmd.Stdout = \u0026out\n-\treviewCmd.Stderr = \u0026out\n-\n-\terr := reviewCmd.Run()\n-\tif err != nil {\n-\t\tfmt.Println(\"review command returned error\")\n-\t\tfmt.Println(err)\n-\t\tfmt.Printf(\"output:\\n %s\", out.String())\n-\t\tfailExit()\n-\t}\n+\tcliCmd := exec.CommandContext(ctx, lookoutBin, args...)\n+\tcliCmd.Stdout = \u0026out\n+\tcliCmd.Stderr = \u0026out\n+\n+\terr := cliCmd.Run()\n+\trequire.NoErrorf(err,\n+\t\t\"'lookout %s' command returned error. output:\\n%s\",\n+\t\tstrings.Join(args, \" \"), out.String())\n \n \treturn \u0026out\n }\n \n // ResetDB recreates database for the test\n-func ResetDB() {\n+func ResetDB(require *require.Assertions) {\n \tdb, err := sql.Open(\"postgres\", \"postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable\")\n-\tif err != nil {\n-\t\tfmt.Println(\"can't connect to DB:\", err)\n-\t\tos.Exit(1)\n-\t}\n+\trequire.NoError(err, \"can't connect to DB\")\n \n \t_, err = db.Exec(\"DROP SCHEMA public CASCADE;\")\n-\tnoDBErr(err)\n+\trequire.NoError(err, \"can't execute query\")\n \t_, err = db.Exec(\"CREATE SCHEMA public;\")\n-\tnoDBErr(err)\n+\trequire.NoError(err, \"can't execute query\")\n \t_, err = db.Exec(\"GRANT ALL ON SCHEMA public TO postgres;\")\n-\tnoDBErr(err)\n+\trequire.NoError(err, \"can't execute query\")\n \t_, err = db.Exec(\"GRANT ALL ON SCHEMA public TO public;\")\n-\tnoDBErr(err)\n+\trequire.NoError(err, \"can't execute query\")\n \n \terr = exec.Command(lookoutBin, \"migrate\").Run()\n-\tif err != nil {\n-\t\tfmt.Println(\"can't migrate DB:\", err)\n-\t\tos.Exit(1)\n-\t}\n-}\n-\n-func noDBErr(err error) {\n-\tif err != nil {\n-\t\tfmt.Println(\"can't execute query\", err)\n-\t\tos.Exit(1)\n-\t}\n+\trequire.NoError(err, \"can't migrate DB\")\n }\n \n func failExit() {"

func TestParseHunks_issue_165(t *testing.T) {
	require := require.New(t)

	hunks, err := parseHunks(largeHunk)
	require.NoError(err)
	for _, hunk := range hunks {
		hunk.Chunks = nil
	}

	require.Equal([]*hunk{&hunk{
		OldStartLine: 6,
		OldLines:     20,
		NewStartLine: 6,
		NewLines:     21,
	}, &hunk{
		OldStartLine: 50,
		OldLines:     7,
		NewStartLine: 51,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 61,
		OldLines:     31,
		NewStartLine: 62,
		NewLines:     25,
	}, &hunk{
		OldStartLine: 97,
		OldLines:     85,
		NewStartLine: 92,
		NewLines:     59,
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
