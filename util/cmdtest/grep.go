package cmdtest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// GrepTimeout defines timeout grep is waiting for substring
var GrepTimeout = time.Minute

// GrepTrue reads from reader until finds substring with timeout or fails printing read lines
func GrepTrue(r io.Reader, substr string) {
	found, buf := Grep(r, substr)
	if !found {
		fmt.Printf("'%s' is not found in output:\n", substr)
		fmt.Println(buf.String())
		failExit()
	}
}

// GrepAndNot reads from reader until finds substring with timeout and checks noSubstr was read
// or fails printing read lines
func GrepAndNot(r io.Reader, substr, noSubstr string) {
	found, buf := Grep(r, substr)
	if !found {
		fmt.Printf("'%s' is not found in output:\n", substr)
		fmt.Println(buf.String())
		failExit()
		return
	}
	if strings.Contains(buf.String(), noSubstr) {
		fmt.Printf("'%s' should not be in output:\n", noSubstr)
		fmt.Println(buf.String())
		failExit()
	}
}

// Grep reads from reader until finds substring with timeout
// return result and content that was read
func Grep(r io.Reader, substr string) (bool, *bytes.Buffer) {
	buf := &bytes.Buffer{}

	foundch := make(chan bool, 1)
	scanner := bufio.NewScanner(r)
	go func() {
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Fprintln(buf, t)
			if strings.Contains(t, substr) {
				foundch <- true
				break
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading input:", err)
		}
	}()

	select {
	case <-time.After(GrepTimeout):
		return false, buf
	case found := <-foundch:
		return found, buf
	}
}
