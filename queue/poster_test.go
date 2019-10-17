package queue

import (
	"context"
	"io"
	"testing"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
	"github.com/stretchr/testify/suite"
	log "gopkg.in/src-d/go-log.v1"
)

type PosterSuite struct {
	suite.Suite
}

func TestPosterSuite(t *testing.T) {
	suite.Run(t, new(PosterSuite))
}

func (s *PosterSuite) TestPostSuccess() {
	require := s.Require()
	logFields := log.Fields{"test": "field"}
	ctx, _ := ctxlog.WithLogFields(context.Background(), logFields)

	q := initQueue(s.T(), "memoryfinite://")
	underlying := &mockPoster{}
	p := NewPoster(underlying, q)

	event := &lookout.ReviewEvent{}
	event.InternalID = "test-id"
	comments := []lookout.AnalyzerComments{
		{
			Config: lookout.AnalyzerConfig{Name: "test"},
			Comments: []*lookout.Comment{
				{Text: "test"},
			},
		},
	}
	err := p.Post(ctx, event, comments, true)
	require.NoError(err)

	err = p.Consume(ctx, 1)
	require.Error(err, io.EOF)

	// validate underlying poster got the same comments as we sent to queue poster
	require.Len(underlying.posted, 1)
	require.Equal(event.ID(), underlying.posted[0].Event.ID())
	require.Equal(comments, underlying.posted[0].Comments)
	require.Equal(true, underlying.posted[0].Safe)
	// validate log continuity
	require.Equal(logFields, underlying.posted[0].LogFields)
}

type mockPoster struct {
	posted []*mockPosterItem
}

type mockPosterItem struct {
	Event     lookout.Event
	Comments  []lookout.AnalyzerComments
	Safe      bool
	LogFields log.Fields
}

func (p *mockPoster) Post(ctx context.Context, e lookout.Event, cs []lookout.AnalyzerComments, safe bool) error {
	p.posted = append(p.posted, &mockPosterItem{
		Event:     e,
		Comments:  cs,
		Safe:      safe,
		LogFields: ctxlog.Fields(ctx),
	})
	return nil
}

func (p *mockPoster) Status(ctx context.Context, e lookout.Event, s lookout.AnalysisStatus) error {
	return nil
}
