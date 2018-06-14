package dummy

import (
	"bytes"
	"context"
	"fmt"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/utils/binary"
)

type Analyzer struct {
	DataClient *lookout.DataClient
}

var _ lookout.AnalyzerServer = &Analyzer{}

func (a *Analyzer) Analyze(ctx context.Context, req *lookout.AnalysisRequest) (
	*lookout.AnalysisResponse, error) {

	changes, err := a.DataClient.GetChanges(ctx, &lookout.ChangesRequest{
		Repository:   req.Repository,
		Base:         req.BaseHash,
		Top:          req.NewHash,
		WantContents: true,
	})
	if err != nil {
		return nil, err
	}

	resp := &lookout.AnalysisResponse{}
	for changes.Next() {
		change := changes.Change()
		resp.Comments = append(resp.Comments, a.lineIncrease(change)...)
		resp.Comments = append(resp.Comments, a.maxLineWidth(change)...)
	}

	if err := changes.Err(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Analyzer) lineIncrease(ch *lookout.Change) []*lookout.Comment {
	if a.isBinary(ch.GetOld()) || a.isBinary(ch.GetNew()) {
		return nil
	}

	diff := a.countLines(ch.GetNew()) - a.countLines(ch.GetOld())
	if diff <= 0 {
		return nil
	}

	return []*lookout.Comment{{
		File: ch.GetNew().Path,
		Line: int32(0),
		Text: fmt.Sprintf("The file has increased in %d lines.", diff),
	}}
}

func (a *Analyzer) maxLineWidth(ch *lookout.Change) []*lookout.Comment {
	lines := bytes.Split(ch.GetNew().GetContent(), []byte("\n"))
	var comments []*lookout.Comment
	for i, line := range lines {
		if len(line) > 80 {
			comments = append(comments, &lookout.Comment{
				File: ch.GetNew().GetPath(),
				Line: int32(i + 1),
				Text: "This line exceeded 80 bytes.",
			})
		}
	}

	return comments
}

func (a *Analyzer) isBinary(f *lookout.File) bool {
	contents := f.GetContent()
	ok, err := binary.IsBinary(bytes.NewReader(contents))
	return err != nil || ok
}

func (a *Analyzer) countLines(f *lookout.File) int {
	return bytes.Count(f.GetContent(), []byte("\n"))
}
