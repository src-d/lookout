package json

import (
	"context"
	"encoding/json"
	"io"

	"github.com/src-d/lookout"
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

// Post prints json comments to sdtout
func (p *Poster) Post(ctx context.Context, e lookout.Event,
	comments []*lookout.Comment) error {

	for _, c := range comments {
		if err := p.enc.Encode(c); err != nil {
			return err
		}
	}

	return nil
}
