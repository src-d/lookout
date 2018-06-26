package github

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
)

var (
	ErrLineOutOfDiff = errors.NewKind("line number is not in diff")
)

type diffLines struct {
	cc     *github.CommitsComparison
	parsed map[string][]*posRange
}

type hunk struct {
	OldStartLine, OldLines int
	NewStartLine, NewLines int
}

type posRange struct {
	AbsStart, AbsEnd int
	RelStart, RelEnd int
}

func newDiffLines(cc *github.CommitsComparison) *diffLines {
	return &diffLines{
		cc:     cc,
		parsed: make(map[string][]*posRange, len(cc.Files)),
	}
}

func (d *diffLines) ConvertLine(file string, line int) (int, error) {
	ranges, err := d.ranges(file)
	if err != nil {
		return 0, err
	}

	return d.convertLine(ranges, line)
}

func (d *diffLines) convertLine(ranges []*posRange, line int) (int, error) {
	for _, r := range ranges {
		if line >= r.AbsStart && line < r.AbsEnd {
			return line - r.AbsStart + r.RelStart, nil
		}
	}

	return 0, fmt.Errorf("line position is not in diff range")
}

func (d *diffLines) ranges(file string) ([]*posRange, error) {
	if ranges, ok := d.parsed[file]; ok {
		return ranges, nil
	}

	hunks, err := d.hunks(file)
	if err != nil {
		return nil, err
	}

	ranges := convertRanges(hunks)
	d.parsed[file] = ranges
	return ranges, nil
}

func (d *diffLines) hunks(file string) ([]*hunk, error) {
	var ff *github.CommitFile
	for _, f := range d.cc.Files {
		if file == *f.Filename {
			ff = &f
			break
		}
	}

	if ff == nil {
		return nil, fmt.Errorf("file not found: %s", file)
	}

	return parseHunks(*ff.Patch)
}

func parseHunks(s string) ([]*hunk, error) {
	i := 0
	var hs []*hunk
	for i < len(s) {
		read, h, err := parseHunk(s[i:])
		if err != nil {
			return nil, err
		}

		i += read
		hs = append(hs, h)
	}

	return hs, nil
}

var hunkPattern = regexp.MustCompile(`^(@@ -(\d+),(\d+) \+(\d+)(?:,(\d+))? @@[^@]*)(?:@@.*|$)`)

func parseHunk(s string) (int, *hunk, error) {
	var (
		err error
		h   = &hunk{}
	)
	matches := hunkPattern.FindStringSubmatch(s)
	if len(matches) == 0 {
		return 0, nil, fmt.Errorf("bad hunk format")
	}

	h.OldStartLine, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, nil, fmt.Errorf("bad hunk format")
	}

	h.OldLines, err = strconv.Atoi(matches[3])
	if err != nil {
		return 0, nil, fmt.Errorf("bad hunk format")
	}

	h.NewStartLine, err = strconv.Atoi(matches[4])
	if err != nil {
		return 0, nil, fmt.Errorf("bad hunk format")
	}

	if matches[5] == "" {
		h.NewLines = 1
	} else {
		h.NewLines, err = strconv.Atoi(matches[5])
		if err != nil {
			return 0, nil, fmt.Errorf("bad hunk format")
		}
	}

	return len(matches[1]), h, nil
}

func convertRanges(hunks []*hunk) []*posRange {
	if len(hunks) == 0 {
		return nil
	}

	ranges := make([]*posRange, len(hunks))
	for i, hunk := range hunks {
		if i == 0 {
			ranges[0] = &posRange{
				AbsStart: hunk.NewStartLine,
				AbsEnd:   hunk.NewStartLine + hunk.NewLines,
				RelStart: 1,
				RelEnd:   1 + hunk.NewLines,
			}
			continue
		}

		ranges[i] = &posRange{
			AbsStart: hunk.NewStartLine,
			AbsEnd:   hunk.NewStartLine + hunk.NewLines,
			RelStart: ranges[i-1].RelEnd + 1,
			RelEnd:   ranges[i-1].RelEnd + 1 + hunk.NewLines,
		}
	}

	return ranges
}
