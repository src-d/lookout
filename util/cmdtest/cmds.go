package cmdtest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/lib/pq"
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
		time.Sleep(time.Second) // go needs a bit of time to kill process
	}
}

// StartDummy starts dummy analyzer with context and optional arguments
func (suite *IntegrationSuite) StartDummy(args ...string) io.Reader {
	r, outputWriter := io.Pipe()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	args = append([]string{"serve"}, args...)

	cmd := exec.CommandContext(suite.Ctx, dummyBin, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	err := cmd.Start()
	suite.Require().NoError(err, "can't start analyzer")

	go func() {
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

// StartServe starts lookout server with context and optional arguments
func (suite *IntegrationSuite) StartServe(args ...string) (io.Reader, io.WriteCloser) {
	require := suite.Require()

	r, outputWriter := io.Pipe()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	args = append([]string{"serve"}, args...)

	cmd := exec.CommandContext(suite.Ctx, lookoutBin, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	w, err := cmd.StdinPipe()
	require.NoError(err, "can't start server")

	err = cmd.Start()
	require.NoError(err, "can't start server")

	go func() {
		if err := cmd.Wait(); err != nil {
			// don't print error if analyzer was killed by cancel
			if suite.Ctx.Err() != context.Canceled {
				fmt.Println("server exited with error:", err)
				fmt.Printf("output:\n%s", buf.String())
				// T.Fail cannot be called from a goroutine
				suite.Stop()
				os.Exit(1)
			}
		}
	}()

	return tee, w
}

// RunCli runs lookout subcommand (not a server)
func (suite *IntegrationSuite) RunCli(cmd string, args ...string) io.Reader {
	args = append([]string{cmd}, args...)

	var out bytes.Buffer
	cliCmd := exec.CommandContext(suite.Ctx, lookoutBin, args...)
	cliCmd.Stdout = &out
	cliCmd.Stderr = &out

	err := cliCmd.Run()
	suite.Require().NoErrorf(err,
		"'lookout %s' command returned error. output:\n%s",
		strings.Join(args, " "), out.String())

	return &out
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

	suite.T().Logf("running %s", lookoutBin)
	err = exec.Command(lookoutBin, "migrate").Run()
	require.NoError(err, "can't migrate DB")
}

func (suite *IntegrationSuite) runQuery(db *sql.DB, query string) {
	_, err := db.Exec(query)
	suite.Require().NoError(err, "can't execute SQL: %q", query)
}
