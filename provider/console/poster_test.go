package console

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

var (
	hash1 = "f67e5455a86d0f2a366f1b980489fac77a373bd0"
	hash2 = "02801e1a27a0a906d59530aeb81f4cd137f2c717"
	base1 = plumbing.ReferenceName("base")
	head1 = plumbing.ReferenceName("refs/pull/42/head")
)

func TestPoster_Post_OK(t *testing.T) {
	require := require.New(t)

	var b bytes.Buffer

	p := NewPoster(&b)
	ev := &lookout.ReviewEvent{
		Provider: Provider,
		CommitRevision: lookout.CommitRevision{
			Base: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
				ReferenceName:         base1,
				Hash:                  hash1,
			},
			Head: lookout.ReferencePointer{
				InternalRepositoryURL: "https://github.com/foo/bar",
				ReferenceName:         head1,
				Hash:                  hash2,
			}}}
	cs := []*lookout.Comment{&lookout.Comment{
		Text: "This is a global comment",
	}, &lookout.Comment{
		File: "main.go",
		Text: "This is a file comment",
	}, &lookout.Comment{
		File: "main.go",
		Line: 5,
		Text: "This is a line comment",
	}, &lookout.Comment{
		Text: "This is a another global comment",
	}}

	err := p.Post(context.Background(), ev, cs)
	require.NoError(err)

	expected := fmt.Sprintf(globalComment, cs[0].Text) +
		fmt.Sprintf(fileComment, cs[1].File, cs[1].Text) +
		fmt.Sprintf(lineComment, cs[2].File, cs[2].Line, cs[2].Text) +
		fmt.Sprintf(globalComment, cs[3].Text)

	require.Equal(expected, b.String())
}
