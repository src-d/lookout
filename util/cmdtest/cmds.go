package cmdtest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CmdTimeout defines timeout for a command
var CmdTimeout = time.Minute

// default path to binaries
var dummyBin = "../../build/bin/dummy"
var lookoutBin = "../../build/bin/lookoutd"

type IntegrationSuite struct {
	suite.Suite
	Ctx  context.Context
	Stop func()
	wg   sync.WaitGroup

	readBuf   *bytes.Buffer
	unreadBuf *bytes.Buffer
}

func init() {
	if os.Getenv("DUMMY_BIN") != "" {
		dummyBin = os.Getenv("DUMMY_BIN")
	}
	if os.Getenv("LOOKOUT_BIN") != "" {
		lookoutBin = os.Getenv("LOOKOUT_BIN")
	}
}

// StoppableCtx return ctx and stop function
func (suite *IntegrationSuite) StoppableCtx() {
	ctx, timeoutCancel := context.WithTimeout(context.Background(), CmdTimeout)

	var cancel context.CancelFunc
	suite.Ctx, cancel = context.WithCancel(ctx)
	suite.Stop = func() {
		timeoutCancel()
		cancel()
		fmt.Println("stopping services")
		suite.wg.Wait()
	}
}

// StartDummy starts dummy analyzer with context and optional arguments
func (suite *IntegrationSuite) StartDummy(args ...string) io.Reader {
	r, outputWriter := io.Pipe()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	args = append([]string{"serve"}, args...)

	fmt.Printf("starting dummy %s\n", strings.Join(args, " "))

	cmd := exec.CommandContext(suite.Ctx, dummyBin, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	go func() {
		// cmd.Wait() will not finish until stdout is closed
		<-suite.Ctx.Done()
		outputWriter.Close()
	}()

	err := cmd.Start()
	suite.Require().NoError(err, "can't start analyzer")

	suite.wg.Add(1)
	go func() {
		defer suite.wg.Done()

		if err := cmd.Wait(); err != nil {
			// don't print error if analyzer was killed by cancel
			if suite.Ctx.Err() != context.Canceled {
				fmt.Println("analyzer exited with error:", err)
				fmt.Printf("output:\n%s", buf.String())
				// T.Fail cannot be called from a goroutine
				suite.Stop()
				os.Exit(1)
			}
		}
	}()

	return tee
}

// StartLookoutd starts lookoutd serve, or watch and work if the queue testing
// is enabled
func (suite *IntegrationSuite) StartLookoutd(configFile string) (io.Reader, io.WriteCloser) {
	if suite.IsQueueTested() {
		watcherR, watcherW := suite.StartWatcher("--provider", "json",
			"-c", configFile)

		workerR, _ := suite.StartWorker("--provider", "json",
			"-c", configFile, "--probes-addr", "0.0.0.0:8091")

		// make sure watcher server started correctly
		suite.GrepTrue(watcherR, "Starting watcher")

		// make sure worker started correctly
		suite.GrepTrue(workerR, "Worker started")

		// Write json commands to watcher, write processed output from worker
		return workerR, watcherW
	} else {
		r, w := suite.StartServe("--provider", "json",
			"-c", configFile)

		// make sure server started correctly
		suite.GrepTrue(r, "Starting watcher")

		return r, w
	}
}

// StartServe starts lookout server with context and optional arguments
func (suite *IntegrationSuite) StartServe(args ...string) (io.Reader, io.WriteCloser) {
	args = append([]string{"serve"}, args...)
	return suite.startLookoutd(args...)
}

// StartWatcher starts lookoutd watch with context and optional arguments
func (suite *IntegrationSuite) StartWatcher(args ...string) (io.Reader, io.WriteCloser) {
	args = append([]string{"watch"}, args...)
	return suite.startLookoutd(args...)
}

// StartWorker starts lookoutd work with context and optional arguments
func (suite *IntegrationSuite) StartWorker(args ...string) (io.Reader, io.WriteCloser) {
	args = append([]string{"work"}, args...)
	return suite.startLookoutd(args...)
}

func (suite *IntegrationSuite) startLookoutd(args ...string) (io.Reader, io.WriteCloser) {
	require := suite.Require()

	suite.unreadBuf = &bytes.Buffer{}
	suite.readBuf = &bytes.Buffer{}

	cmd := exec.CommandContext(suite.Ctx, lookoutBin, args...)
	cmd.Stdout = suite.unreadBuf
	cmd.Stderr = suite.unreadBuf

	r := &testReader{buf: suite.unreadBuf}
	tee := io.TeeReader(r, suite.readBuf)

	go func() {
		// cmd.Wait() will not finish until stdout is closed
		<-suite.Ctx.Done()
		r.Close()
	}()

	fmt.Printf("starting lookoutd %s\n", strings.Join(args, " "))

	w, err := cmd.StdinPipe()
	require.NoError(err, "can't start lookoutd")

	err = cmd.Start()
	require.NoError(err, "can't start lookoutd")

	suite.wg.Add(1)
	go func() {
		defer suite.wg.Done()

		if err := cmd.Wait(); err != nil {
			// don't print error if killed by cancel
			if suite.Ctx.Err() != context.Canceled {
				fmt.Println("lookoutd exited with error:", err)
				fmt.Printf("output:\n%s", suite.readBuf.String())
				// T.Fail cannot be called from a goroutine
				suite.Stop()
				os.Exit(1)
			}
		}
	}()

	return tee, w
}

// IsQueueTested returns true if LOOKOUT_TEST_QUEUE env var is set to true
func (suite *IntegrationSuite) IsQueueTested() bool {
	res := false
	qEnv := os.Getenv("LOOKOUT_TEST_QUEUE")
	if qEnv != "" {
		var err error
		res, err = strconv.ParseBool(qEnv)
		require.NoError(suite.T(), err, "failed to parse env var LOOKOUT_TEST_QUEUE, it must be a boolean")
	}

	return res
}

// RunCli runs lookout subcommand (not a server)
func (suite *IntegrationSuite) RunCli(cmd string, args ...string) io.Reader {
	out, err := suite.runCli(cmd, args...)
	suite.Require().NoErrorf(err,
		"'%s %s' command returned error. output:\n%s",
		cmd, strings.Join(args, " "), out.String())

	return out
}

// RunCliErr runs lookout subcommand that should fail
func (suite *IntegrationSuite) RunCliErr(cmd string, args ...string) io.Reader {
	out, err := suite.runCli(cmd, args...)
	suite.Require().Error(err, "'%s %s' command should return error", cmd, strings.Join(args, " "))

	return out
}

func (suite *IntegrationSuite) runCli(cmd string, args ...string) (*bytes.Buffer, error) {
	args = append([]string{cmd}, args...)

	var out bytes.Buffer
	cliCmd := exec.CommandContext(suite.Ctx, lookoutBin, args...)
	cliCmd.Stdout = &out
	cliCmd.Stderr = &out

	return &out, cliCmd.Run()
}

// ResetDB recreates database for the test
func (suite *IntegrationSuite) ResetDB() {
	require := suite.Require()

	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable")
	require.NoError(err, "can't connect to DB")

	suite.runQuery(db, "DROP SCHEMA public CASCADE;")
	suite.runQuery(db, "CREATE SCHEMA public;")
	suite.runQuery(db, "GRANT ALL ON SCHEMA public TO postgres;")
	suite.runQuery(db, "GRANT ALL ON SCHEMA public TO public;")

	fmt.Println("running lookoutd migrate")
	err = exec.Command(lookoutBin, "migrate").Run()
	require.NoError(err, "can't migrate DB")
}

func (suite *IntegrationSuite) runQuery(db *sql.DB, query string) {
	_, err := db.Exec(query)
	suite.Require().NoError(err, "can't execute SQL: %q", query)
}

// Output returns the output read so far from the reader returned in StartLookoutd
func (suite *IntegrationSuite) Output() string {
	return suite.readBuf.String()
}

// AllOutput returns the all output returned in StartLookoutd
func (suite *IntegrationSuite) AllOutput() string {
	return suite.readBuf.String() + suite.unreadBuf.String()
}

// a reader that uses bytes.Buffer but doesn't return EOF when the buffer ends
// need for compatibility with previous code
type testReader struct {
	buf    *bytes.Buffer
	closed bool
	wait   bool
}

func (r *testReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, io.EOF
	}

	if r.wait {
		for {
			n, err := r.buf.Read(p)
			if err != io.EOF {
				r.wait = false
				return n, err
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	n, err := r.buf.Read(p)
	if err == io.EOF {
		r.wait = true
		return n, nil
	}

	return n, err
}

func (r *testReader) Close() error {
	r.closed = true
	return nil
}
