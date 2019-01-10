package lookout

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzerCommentsGroupsFilter(t *testing.T) {
	assert := assert.New(t)

	g := AnalyzerCommentsGroups{
		{
			Comments: []*Comment{
				{Text: "survive"},
				{Text: "skip"},
				{Text: "survive"},
			},
		},
		{
			Comments: []*Comment{
				{Text: "survive"},
			},
		},
		{
			Comments: []*Comment{
				{Text: "skip"},
			},
		},
	}

	result, err := g.Filter(func(c *Comment) (skip bool, err error) {
		return c.Text == "skip", nil
	})

	assert.NoError(err)
	assert.Len(result, 2)
	assert.Len(result[0].Comments, 2)
	assert.Len(result[1].Comments, 1)

	e := errors.New("test-error")
	_, err = g.Filter(func(c *Comment) (skip bool, err error) {
		return false, e
	})

	assert.Equal(e, err)
}
