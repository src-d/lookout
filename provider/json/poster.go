package json

import (
	"context"
	"encoding/json"
	"io"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
	"gopkg.in/src-d/go-log.v1"
)

// Poster prints json comments to stdout
type Poster struct {
	writer io.Writer
	enc    *json.Encoder
}

var _ lookout.Poster = &Poster{}

// NewPoster creates a new json poster for stdout
func NewPoster(w io.Writer) *Poster {
	return &Poster{
		writer: w,
		enc:    json.NewEncoder(w),
	}
}

// Post prints json comments to stdout
func (p *Poster) Post(ctx context.Context, e lookout.Event,
	aCommentsList []lookout.AnalyzerComments) error {

	for _, a := range aCommentsList {
		for _, c := range a.Comments {
			if err := p.enc.Encode(commentToPrint{AnalyzerName: a.Config.Name, Comment: c}); err != nil {
				return err
			}
		}
	}

	return nil
}

// Status prints the new status to the log
func (p *Poster) Status(ctx context.Context, e lookout.Event,
	status lookout.AnalysisStatus) error {

	ctxlog.Get(ctx).With(log.Fields{"status": status}).Infof("New status")
	return nil
}

type commentToPrint struct {
	AnalyzerName string `json:"analyzer-name"`
	*lookout.Comment
}
