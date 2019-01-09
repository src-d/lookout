package cmdtest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// GrepTimeout defines timeout grep is waiting for substring
var GrepTimeout = 30 * time.Second

const extraDebug = false

// GrepTrue reads from reader until finds substring with timeout or fails,
// printing read lines
func (s *IntegrationSuite) GrepTrue(r io.Reader, substr string) string {
	return s.GrepAll(r, []string{substr})
}

// GrepAll is like GrepTrue but for an array of strings. It waits util the last
// line is found, then looks for all the lines in the read text
func (s *IntegrationSuite) GrepAll(r io.Reader, strs []string) string {
	return s.GrepAndNotAll(r, strs, nil)
}

// GrepAndNot reads from reader until finds substring with timeout and checks noSubstr was read
// or fails printing read lines
func (s *IntegrationSuite) GrepAndNot(r io.Reader, substr, noSubstr string) string {
	return s.GrepAndNotAll(r, []string{substr}, []string{noSubstr})
}

// GrepAndNotAll is like GrepAndNot but for arrays of strings. It waits util
// the last line is found, then looks for all the lines in the read text
func (s *IntegrationSuite) GrepAndNotAll(r io.Reader, strs []string, noStrs []string) string {
	// If the stream from stdin is read sequentially with Grep(), there was
	// an erratic behaviour where some lines where not processed.

	// Wait until the last substr is found
	_, buf := s.Grep(r, strs[len(strs)-1])
	read := buf.String()

	// Look for the previous messages in the lines read up to that last substr
	for _, st := range strs {
		if !strings.Contains(read, st) {
			fmt.Printf("'%s' is not found in output:\n", st)
			fmt.Println(read)
			fmt.Printf("\nThe complete command output:\n%s", s.logBuf.String())
			s.Stop()
			s.Suite.T().FailNow()
		}
	}

	for _, st := range noStrs {
		if strings.Contains(read, st) {
			fmt.Printf("'%s' should not be in output:\n", st)
			fmt.Println(read)
			fmt.Printf("\nThe complete command output:\n%s", s.logBuf.String())
			s.Stop()
			s.Suite.T().FailNow()
		}
	}

	return read
}

func (s *IntegrationSuite) scanAndFind(r io.Reader, fn func(*bytes.Buffer, string) bool, caller string) (bool, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	var found bool

	foundch := make(chan bool, 1)
	scanner := bufio.NewScanner(r)
	go func() {
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Fprintln(buf, t)

			if fn(buf, t) {
				found = true
				break
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading input:", err)
		}

		foundch <- found
	}()
	select {
	case <-time.After(GrepTimeout):
		if extraDebug {
			fmt.Printf(" >>>> %s Timeout reached", caller)
		}

		break
	case found = <-foundch:
	}

	if extraDebug {
		fmt.Printf("The complete command output so far:\n%s", s.logBuf.String())
	}

	return found, buf
}

func (s *IntegrationSuite) iterAndFind(str []string, fn func(string) bool, caller string) bool {
	var found bool
	for _, t := range str {
		if fn(t) {
			found = true
			break
		}
	}

	if extraDebug {
		fmt.Printf("The complete command output so far:\n%s", s.logBuf.String())
	}

	return found
}

// Grep reads from reader until finds substring with timeout
// return result and content that was read
func (s *IntegrationSuite) Grep(r io.Reader, substr string) (bool, *bytes.Buffer) {
	if extraDebug {
		fmt.Printf("Grep called for substr:\n%s", substr)
	}

	return s.scanAndFind(r, func(buf *bytes.Buffer, line string) bool {
		return strings.Contains(line, substr)
	}, "Grep")
}

// GrepFromString reads the lines in str until finds substring
// return result
func (s *IntegrationSuite) GrepFromString(str string, substr string) bool {
	if extraDebug {
		fmt.Printf("GrepFromString called for substr:\n%s", substr)
	}

	return s.iterAndFind(strings.Split(str, "\n"), func(line string) bool {
		return strings.Contains(line, substr)
	}, "GrepFromString")
}

// Egrep reads from reader until finds matching regex expression with timeout
// return result and content that was read
func (s *IntegrationSuite) Egrep(r io.Reader, expr string) (bool, *bytes.Buffer) {
	reg, _ := regexp.Compile(expr)

	if extraDebug {
		fmt.Printf("Egrep called for expr:\n%s", expr)
	}

	return s.scanAndFind(r, func(buf *bytes.Buffer, line string) bool {
		return reg.MatchString(line)
	}, "Egrep")
}

// EgrepFromString reads the lines in str until finds matching regex expression
// return result
func (s *IntegrationSuite) EgrepFromString(str string, expr string) bool {
	reg, _ := regexp.Compile(expr)

	if extraDebug {
		fmt.Printf("EgrepFromString called for expr:\n%s", expr)
	}

	return s.iterAndFind(strings.Split(str, "\n"), func(line string) bool {
		return reg.MatchString(line)
	}, "EgrepFromString")
}

// EgrepWhole reads from reader until finds matching regex expression over whole content with timeout
// return result and content that was read
func (s *IntegrationSuite) EgrepWhole(r io.Reader, expr string) (bool, *bytes.Buffer) {
	reg, _ := regexp.Compile(expr)

	if extraDebug {
		fmt.Printf("EgrepWhole called for expr:\n%s", expr)
	}

	return s.scanAndFind(r, func(buf *bytes.Buffer, line string) bool {
		return reg.MatchString(buf.String())
	}, "EgrepWhole")
}

// EgrepWholeFromString reads the lines in str until finds matching regex expression over whole content
// return result
func (s *IntegrationSuite) EgrepWholeFromString(str string, expr string) bool {
	reg, _ := regexp.Compile(expr)

	if extraDebug {
		fmt.Printf("EgrepWholeFromString called for expr:\n%s", expr)
	}

	return s.iterAndFind(strings.Split(str, "\n"), func(line string) bool {
		return reg.MatchString(str)
	}, "EgrepWholeFromString")
}
