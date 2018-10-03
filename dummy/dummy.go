package dummy

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/utils/binary"
)

type Analyzer struct {
	Version          string
	DataClient       *lookout.DataClient
	RequestUAST      bool
	RequestFilesPush bool
}

var _ lookout.AnalyzerServer = &Analyzer{}

func (a *Analyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (
	*lookout.EventResponse, error) {

	changes, err := a.DataClient.GetChanges(ctx, &lookout.ChangesRequest{
		Base:           &e.CommitRevision.Base,
		Head:           &e.CommitRevision.Head,
		WantContents:   true,
		WantUAST:       a.RequestUAST,
		IncludePattern: ".*",
		ExcludePattern: "^should-never-match$",
	})
	if err != nil {
		return nil, err
	}

	resp := &lookout.EventResponse{AnalyzerVersion: a.Version}
	for changes.Next() {
		change := changes.Change()
		resp.Comments = append(resp.Comments, a.lineIncrease(change)...)
		resp.Comments = append(resp.Comments, a.maxLineLen(change.Head)...)
		if a.RequestUAST {
			resp.Comments = append(resp.Comments, a.hasUAST(change.Head)...)
			resp.Comments = append(resp.Comments, a.language(change.Head)...)
		}
	}

	if err := changes.Err(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Analyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	resp := &lookout.EventResponse{AnalyzerVersion: a.Version}

	if !a.RequestFilesPush {
		return resp, nil
	}
	files, err := a.DataClient.GetFiles(ctx, &lookout.FilesRequest{
		Revision:        &e.CommitRevision.Head,
		ExcludeVendored: true,
		WantContents:    true,
		WantUAST:        a.RequestUAST,
		IncludePattern:  ".*",
		ExcludePattern:  "^should-never-match$",
	})
	if err != nil {
		return nil, err
	}

	for files.Next() {
		file := files.File()
		resp.Comments = append(resp.Comments, a.maxLineLen(file)...)
		if a.RequestUAST {
			resp.Comments = append(resp.Comments, a.hasUAST(file)...)
			resp.Comments = append(resp.Comments, a.language(file)...)
		}
	}

	if err := files.Err(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Analyzer) lineIncrease(ch *lookout.Change) []*lookout.Comment {
	if a.isBinary(ch.Head) || a.isBinary(ch.Base) {
		return nil
	}

	diff := a.countLines(ch.Head) - a.countLines(ch.Base)
	if diff <= 0 {
		return nil
	}

	return []*lookout.Comment{{
		File: ch.Head.Path,
		Line: 0,
		Text: fmt.Sprintf("The file has increased in %d lines.", diff),
	}}
}

const maxLineLength = 120

func (a *Analyzer) maxLineLen(file *lookout.File) []*lookout.Comment {
	if file == nil || a.isBinary(file) {
		return nil
	}

	lines := strings.Split(string(file.Content), "\n")
	var comments []*lookout.Comment
	for i, line := range lines {
		if len(line) > maxLineLength {
			comments = append(comments, &lookout.Comment{
				File: file.Path,
				Line: int32(i + 1),
				Text: fmt.Sprintf("This line exceeded %d chars.", maxLineLength),
			})
		}
	}

	return comments
}

func (a *Analyzer) hasUAST(file *lookout.File) []*lookout.Comment {
	if file == nil {
		return nil
	}

	var text string
	if file.UAST == nil {
		text = "The file doesn't have UAST."
	} else {
		text = "The file has UAST."
	}

	return []*lookout.Comment{{
		File: file.Path,
		Line: 0,
		Text: text,
	}}
}

func (a *Analyzer) language(file *lookout.File) []*lookout.Comment {
	if file == nil {
		return nil
	}

	return []*lookout.Comment{{
		File: file.Path,
		Line: 0,
		Text: fmt.Sprintf("The file has language detected: %q", file.Language),
	}}
}

func (a *Analyzer) isBinary(f *lookout.File) bool {
	if f == nil {
		return false
	}

	ok, err := binary.IsBinary(bytes.NewReader(f.Content))
	return err != nil || ok
}

func (a *Analyzer) countLines(f *lookout.File) int {
	if f == nil {
		return 0
	}

	return bytes.Count(f.Content, []byte("\n"))
}
