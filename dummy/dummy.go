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

func (a *Analyzer) NotifyPullRequestEvent(ctx context.Context, e *lookout.PullRequestEvent) (
	*lookout.EventResponse, error) {

	changes, err := a.DataClient.GetChanges(ctx, &lookout.ChangesRequest{
		Base:         &e.Base,
		Head:         &e.Head,
		WantContents: true,
	})
	if err != nil {
		return nil, err
	}

	resp := &lookout.EventResponse{}
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

func (a *Analyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	return nil, fmt.Errorf("not implemented")
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

func (a *Analyzer) maxLineWidth(ch *lookout.Change) []*lookout.Comment {
	lines := bytes.Split(ch.Head.Content, []byte("\n"))
	var comments []*lookout.Comment
	for i, line := range lines {
		if len(line) > 80 {
			comments = append(comments, &lookout.Comment{
				File: ch.Head.Path,
				Line: int32(i + 1),
				Text: "This line exceeded 80 bytes.",
			})
		}
	}

	return comments
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
