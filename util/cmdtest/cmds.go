package cmdtest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	_ "github.com/lib/pq"
)

// CmdTimeout defines timeout for a command
var CmdTimeout = time.Minute

// default path to binaries
var dummyBin = "build/bin/dummy"
var lookoutBin = "build/bin/lookout"

// function to stop running commands
// redefined in StoppableCtx
var stop func()

func init() {
	if os.Getenv("DUMMY_BIN") != "" {
		dummyBin = os.Getenv("DUMMY_BIN")
	}
	if os.Getenv("LOOKOUT_BIN") != "" {
		lookoutBin = os.Getenv("LOOKOUT_BIN")
	}
}

// StoppableCtx return ctx and stop function
func StoppableCtx() (context.Context, func()) {
	ctx, timeoutCancel := context.WithTimeout(context.Background(), CmdTimeout)

	ctx, cancel := context.WithCancel(ctx)
	stop = func() {
		timeoutCancel()
		cancel()
		fmt.Println("stopping services")
		time.Sleep(time.Second) // go needs a bit of time to kill process
	}

	return ctx, stop
}

// StartDummy starts dummy analyzer with context and optional arguments
func StartDummy(ctx context.Context, args ...string) io.Reader {
	r, outputWriter := io.Pipe()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	args = append([]string{"serve"}, args...)

	cmd := exec.CommandContext(ctx, dummyBin, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	err := cmd.Start()
	if err != nil {
		ioutil.ReadAll(tee)
		fmt.Println("can't start analyzer:")
		fmt.Println(err)
		fmt.Printf("output:\n %s", buf.String())
		os.Exit(1)
	} else {
		go func() {
			if err := cmd.Wait(); err != nil {
				// don't print error if analyzer was killed by cancel
				if ctx.Err() != context.Canceled {
					ioutil.ReadAll(tee)
					fmt.Println("analyzer exited with error:", err)
					fmt.Printf("output:\n%s", buf.String())
					failExit()
				}
			}
		}()
	}

	return tee
}

// StartServe starts lookout server with context and optional arguments
func StartServe(ctx context.Context, args ...string) (io.Reader, io.WriteCloser) {
	r, outputWriter := io.Pipe()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	args = append([]string{"serve"}, args...)

	cmd := exec.CommandContext(ctx, lookoutBin, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	w, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("can't start server:")
		fmt.Println(err)
		os.Exit(1)
	}

	err = cmd.Start()
	if err != nil {
		ioutil.ReadAll(tee)
		fmt.Println("can't start server:")
		fmt.Println(err)
		fmt.Printf("output:\n %s", buf.String())
		os.Exit(1)
	} else {
		go func() {
			if err := cmd.Wait(); err != nil {
				// don't print error if analyzer was killed by cancel
				if ctx.Err() != context.Canceled {
					ioutil.ReadAll(tee)
					fmt.Println("server exited with error:", err)
					fmt.Printf("output:\n%s", buf.String())
					failExit()
				}
			}
		}()
	}

	return tee, w
}

// RunCli runs lookout subcommand (not a server)
func RunCli(ctx context.Context, cmd string, args ...string) io.Reader {
	args = append([]string{cmd}, args...)

	var out bytes.Buffer
	reviewCmd := exec.CommandContext(ctx, lookoutBin, args...)
	reviewCmd.Stdout = &out
	reviewCmd.Stderr = &out

	err := reviewCmd.Run()
	if err != nil {
		fmt.Println("review command returned error")
		fmt.Println(err)
		fmt.Printf("output:\n %s", out.String())
		failExit()
	}

	return &out
}

// ResetDB recreates database for the test
func ResetDB() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable")
	if err != nil {
		fmt.Println("can't connect to DB:", err)
		os.Exit(1)
	}

	_, err = db.Exec("DROP SCHEMA public CASCADE;")
	noDBErr(err)
	_, err = db.Exec("CREATE SCHEMA public;")
	noDBErr(err)
	_, err = db.Exec("GRANT ALL ON SCHEMA public TO postgres;")
	noDBErr(err)
	_, err = db.Exec("GRANT ALL ON SCHEMA public TO public;")
	noDBErr(err)

	err = exec.Command(lookoutBin, "migrate").Run()
	if err != nil {
		fmt.Println("can't migrate DB:", err)
		os.Exit(1)
	}
}

func noDBErr(err error) {
	if err != nil {
		fmt.Println("can't execute query", err)
		os.Exit(1)
	}
}

func failExit() {
	stop()

	os.Exit(1)
}
