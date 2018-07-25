package console

import (
	"context"
	"fmt"
	"io"

	"github.com/src-d/lookout"
)

// Poster prints comments to stdout
type Poster struct {
	writer io.Writer
}

var _ lookout.Poster = &Poster{}

// NewPoster creates a new poster for stdout
func NewPoster(w io.Writer) *Poster {
	return &Poster{
		writer: w,
	}
}

var (
	globalComment = "%s\n"
	fileComment   = "%s: %s\n"
	lineComment   = "%s:%d: %s\n"
)

// Post prints comments to sdtout
func (p *Poster) Post(ctx context.Context, e lookout.Event,
	comments []*lookout.Comment) error {

	for _, c := range comments {
		if c.File == "" {
			fmt.Fprintf(p.writer, globalComment, c.Text)
			continue
		}
		if c.Line == 0 {
			fmt.Fprintf(p.writer, fileComment, c.File, c.Text)
			continue
		}

		fmt.Fprintf(p.writer, lineComment, c.File, c.Line, c.Text)
	}

	return nil
}
