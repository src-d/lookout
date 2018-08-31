package github

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
	log "gopkg.in/src-d/go-log.v1"
)

var (
	ErrLineOutOfDiff = errors.NewKind("line number is not in diff")
)

type diffLines struct {
	cc     *github.CommitsComparison
	parsed map[string][]*posRange
}

type lineType int

const (
	lineAdded lineType = iota
	lineDeleted
	lineContext
)

type linesChunk struct {
	Type  lineType
	Lines int
}

type hunk struct {
	OldStartLine, OldLines int
	NewStartLine, NewLines int
	Chunks                 []linesChunk
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

	return 0, ErrLineOutOfDiff.New()
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

	if ff.Patch == nil {
		return nil, ErrLineOutOfDiff.New()
	}

	hunks, err := parseHunks(*ff.Patch)
	if err != nil {
		log.DefaultLogger.With(log.Fields{"hunk": *ff.Patch}).Errorf(err, "bad hunks")
		return nil, err
	}

	return hunks, nil
}

var hunkPattern = regexp.MustCompile(`^(@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@[^@]*)(?:@@.*|$)`)

func parseHunks(s string) ([]*hunk, error) {
	r := strings.NewReader(s)
	scanner := bufio.NewScanner(r)

	var hs []*hunk
	var h *hunk
	var err error
	var lChunk linesChunk
	for scanner.Scan() {
		var lt lineType

		line := scanner.Text()
		switch true {
		case strings.HasPrefix(line, "@@"):
			if lChunk.Lines > 0 {
				h.Chunks = append(h.Chunks, lChunk)
			}
			lChunk = linesChunk{}
			h, err = parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			hs = append(hs, h)
			continue
		case strings.HasPrefix(line, "+"):
			lt = lineAdded
		case strings.HasPrefix(line, "-"):
			lt = lineDeleted
		default:
			lt = lineContext
		}

		if lChunk.Lines != 0 && lChunk.Type != lt {
			h.Chunks = append(h.Chunks, lChunk)
			lChunk = linesChunk{}
		}

		lChunk.Type = lt
		lChunk.Lines++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if lChunk.Lines > 0 {
		h.Chunks = append(h.Chunks, lChunk)
	}

	return hs, nil
}

func parseHunkHeader(line string) (*hunk, error) {
	var (
		err error
		h   = &hunk{}
	)

	matches := hunkPattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, fmt.Errorf("bad hunk format")
	}

	h.OldStartLine, err = strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("bad hunk format")
	}

	if matches[3] == "" {
		h.OldLines = 1
	} else {
		h.OldLines, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("bad hunk format")
		}
	}

	h.NewStartLine, err = strconv.Atoi(matches[4])
	if err != nil {
		return nil, fmt.Errorf("bad hunk format")
	}

	if matches[5] == "" {
		h.NewLines = 1
	} else {
		h.NewLines, err = strconv.Atoi(matches[5])
		if err != nil {
			return nil, fmt.Errorf("bad hunk format")
		}
	}

	return h, nil
}

func convertRanges(hunks []*hunk) []*posRange {
	if len(hunks) == 0 {
		return nil
	}

	ranges := make([]*posRange, 0)
	// relative position of the last range end
	lastRelEnd := 0
	for _, hunk := range hunks {
		absStart := hunk.NewStartLine

		// number of lines in diff to skip
		// each hunk has a header line which should be skipped
		// delete lines should be also skipped
		skipLines := 1
		// number of lines for the range
		lines := 0

		newRange := func() {
			r := &posRange{
				AbsStart: absStart,
				AbsEnd:   absStart + lines,
				RelStart: lastRelEnd + skipLines,
				RelEnd:   lastRelEnd + lines + skipLines,
			}
			ranges = append(ranges, r)

			absStart = r.AbsEnd
			lastRelEnd = r.RelEnd
		}

		for _, chunk := range hunk.Chunks {
			if chunk.Type != lineDeleted {
				lines += chunk.Lines
			} else {
				// create a range for the lines before first deleted line
				if lines > 0 {
					newRange()
					lines = 0

				}

				skipLines = chunk.Lines
				continue
			}
		}
		if lines > 0 {
			newRange()
			skipLines = 0
		}
	}

	return ranges
}
