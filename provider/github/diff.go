package github

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
)

var (
	// ErrLineOutOfDiff is returned when the file line number is not
	// in the patch diff
	ErrLineOutOfDiff = errors.NewKind("line number is not in diff")
	// ErrLineNotAddition is returned when the file line number is not
	// a + change in the patch diff
	ErrLineNotAddition = errors.NewKind("line number is not an added change")
	// ErrFileNotFound is returned when the file name is not part of the diff
	ErrFileNotFound = errors.NewKind("file not found")
	// ErrBadPatch is returned when there was a problem parsing the diff
	ErrBadPatch = errors.NewKind("diff patch could not be parsed")
)

type diffLines struct {
	cc     *github.CommitsComparison
	parsed map[string]*parsedFile
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

type parsedFile struct {
	ranges     []*posRange
	linesAdded map[int]bool
}

func newDiffLines(cc *github.CommitsComparison) *diffLines {
	return &diffLines{
		cc:     cc,
		parsed: make(map[string]*parsedFile, len(cc.Files)),
	}
}

// ConvertLine takes a line number on the original file, and returns the
// corresponding line number in the patch diff. It will return ErrLineOutOfDiff
// if the line falls outside of the diff (changed lines plus context).
// With strict set to true, ErrLineNotAddition will be returned for lines
// that are not an addition (+ lines in the diff).
func (d *diffLines) ConvertLine(file string, line int, strict bool) (int, error) {
	parsedFile, err := d.parseFile(file)
	if err != nil {
		return 0, err
	}

	diffLine, err := d.convertLine(parsedFile.ranges, line)
	if err != nil {
		return 0, err
	}

	if strict {
		if !parsedFile.linesAdded[diffLine] {
			return 0, ErrLineNotAddition.New()
		}
	}

	return diffLine, nil
}

func (d *diffLines) convertLine(ranges []*posRange, line int) (int, error) {
	for _, r := range ranges {
		if line >= r.AbsStart && line < r.AbsEnd {
			return line - r.AbsStart + r.RelStart, nil
		}
	}

	return 0, ErrLineOutOfDiff.New()
}

func (d *diffLines) parseFile(file string) (*parsedFile, error) {
	if parsedFile, ok := d.parsed[file]; ok {
		return parsedFile, nil
	}

	hunks, linesAdded, err := d.hunks(file)
	if err != nil {
		return nil, err
	}

	ranges := convertRanges(hunks)
	d.parsed[file] = &parsedFile{ranges: ranges, linesAdded: linesAdded}
	return d.parsed[file], nil
}

func (d *diffLines) filePatch(file string) (string, error) {
	var ff *github.CommitFile
	for _, f := range d.cc.Files {
		if file == *f.Filename {
			ff = &f
			break
		}
	}

	if ff == nil {
		return "", ErrFileNotFound.New()
	}

	if ff.Patch == nil {
		return "", ErrLineOutOfDiff.New()
	}

	return *ff.Patch, nil
}

func (d *diffLines) hunks(file string) ([]*hunk, map[int]bool, error) {
	patch, err := d.filePatch(file)
	if err != nil {
		return nil, nil, err
	}

	hunks, linesAdded, err := parseHunks(patch)
	if err != nil {
		return nil, nil, ErrBadPatch.Wrap(err)
	}

	return hunks, linesAdded, nil
}

var hunkPattern = regexp.MustCompile(`^(@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@[^@]*)(?:@@.*|$)`)

func parseHunks(s string) ([]*hunk, map[int]bool, error) {
	r := strings.NewReader(s)
	scanner := bufio.NewScanner(r)

	var hs []*hunk
	var h *hunk
	var err error
	var lChunk linesChunk
	linesAdded := make(map[int]bool)
	for i := 0; scanner.Scan(); i++ {
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
				return nil, nil, err
			}
			hs = append(hs, h)
			continue
		case strings.HasPrefix(line, "+"):
			lt = lineAdded
			linesAdded[i] = true
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
		return nil, nil, err
	}

	if lChunk.Lines > 0 {
		h.Chunks = append(h.Chunks, lChunk)
	}

	return hs, linesAdded, nil
}

func parseHunkHeader(line string) (*hunk, error) {
	var (
		err error
		h   = &hunk{}
	)

	matches := hunkPattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, fmt.Errorf("bad hunk line format: %s", line)
	}

	h.OldStartLine, err = strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("bad hunk line format: %s", line)
	}

	if matches[3] == "" {
		h.OldLines = 1
	} else {
		h.OldLines, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("bad hunk line format: %s", line)
		}
	}

	h.NewStartLine, err = strconv.Atoi(matches[4])
	if err != nil {
		return nil, fmt.Errorf("bad hunk line format: %s", line)
	}

	if matches[5] == "" {
		h.NewLines = 1
	} else {
		h.NewLines, err = strconv.Atoi(matches[5])
		if err != nil {
			return nil, fmt.Errorf("bad hunk line format: %s", line)
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
