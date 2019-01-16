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

func TestAnalyzerCommentsGroupsCount(t *testing.T) {
	assert := assert.New(t)

	g := AnalyzerCommentsGroups{}

	assert.Equal(g.Count(), 0)

	g = AnalyzerCommentsGroups{
		{
			Comments: []*Comment{
				{Text: "some text"},
			},
		},
	}

	assert.Equal(g.Count(), 1)

	g = AnalyzerCommentsGroups{
		{
			Comments: []*Comment{
				{Text: "some text"},
			},
		},
		{
			Comments: []*Comment{
				{Text: "some text"},
				{Text: "some text"},
			},
		},
	}

	assert.Equal(g.Count(), 3)
}

func TestAnalyzerCommentsGroupsDedup(t *testing.T) {
	assert := assert.New(t)

	g := AnalyzerCommentsGroups{
		{
			Comments: []*Comment{
				{File: "f1.go", Line: 1, Text: "some-text", Confidence: 1},
				{File: "f1.go", Line: 1, Text: "some-text", Confidence: 2},
				{File: "f1.go", Line: 2, Text: "some-text", Confidence: 4},
				{File: "f2.go", Line: 1, Text: "some-text", Confidence: 8},
			},
			Config: AnalyzerConfig{Name: "analyzer1"},
		},
		{
			Comments: []*Comment{
				{File: "f1.go", Line: 1, Text: "some-text", Confidence: 1},
				{File: "f2.go", Line: 1, Text: "some-text", Confidence: 2},
				{File: "f2.go", Line: 1, Text: "another-text", Confidence: 4},
			},
			Config: AnalyzerConfig{Name: "analyzer2"},
		},
		{
			Comments: []*Comment{
				{File: "f1.go", Line: 1, Text: "some-text", Confidence: 1},
			},
			Config: AnalyzerConfig{Name: "analyzer3"},
		},
	}

	result := g.Dedup()

	assert.Len(result, 3)
	assert.Len(result[0].Comments, 3)
	assert.Len(result[1].Comments, 3)
	assert.Len(result[2].Comments, 1)

	// for testing use confidence as id, the confidence used in the fixtures
	// has been chosen so that the sum of every possible combination has a
	// unique value (similarly to unix permissions)
	expected := []int{13, 7, 1}
	for i, exp := range expected {
		sum := 0
		for _, c := range result[i].Comments {
			sum += int(c.Confidence)
		}
		assert.Equal(sum, exp)
	}
}
