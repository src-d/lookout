package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHunks(t *testing.T) {
	require := require.New(t)

	hunks, err := parseHunks("@@ -0,0 +1 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 0,
		OldLines:     0,
		NewStartLine: 1,
		NewLines:     1,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@ module Test @@ -1000,7 +1000,7 @@ module Test")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 1000,
		OldLines:     7,
		NewStartLine: 1000,
		NewLines:     7,
	}}, hunks)

	hunks, err = parseHunks("@@ -132,7 +132,7 @@@@ -1000,7 +1000,7 @@")
	require.NoError(err)
	require.Equal([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 1000,
		OldLines:     7,
		NewStartLine: 1000,
		NewLines:     7,
	}}, hunks)
}

func TestConvertRanges(t *testing.T) {
	require := require.New(t)

	ranges := convertRanges([]*hunk{&hunk{
		OldStartLine: 132,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 132, AbsEnd: 139,
		RelStart: 1, RelEnd: 8,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 100,
		OldLines:     7,
		NewStartLine: 132,
		NewLines:     7,
	}, &hunk{
		OldStartLine: 200,
		OldLines:     10,
		NewStartLine: 500,
		NewLines:     10,
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 132, AbsEnd: 139,
		RelStart: 1, RelEnd: 8,
	}, &posRange{
		AbsStart: 500, AbsEnd: 510,
		RelStart: 9, RelEnd: 19,
	}}, ranges)

	ranges = convertRanges([]*hunk{&hunk{
		OldStartLine: 0,
		OldLines:     0,
		NewStartLine: 1,
		NewLines:     446,
	}})

	require.Equal([]*posRange{&posRange{
		AbsStart: 1, AbsEnd: 447,
		RelStart: 1, RelEnd: 447,
	}}, ranges)
}
