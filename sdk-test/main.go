package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// changes might be huge and it will take time to process them
var timeout = time.Minute

// default path to binaries
var dummyBin = "build/bin/dummy"
var lookoutBin = "build/bin/lookout"

// we need to stop analyzer that runs in background
var stop func()

var analyzerOut bytes.Buffer

func main() {
	readEnv()

	fmt.Println("start integration testing")
	ctx, timeoutCancel := context.WithTimeout(context.Background(), timeout)
	defer timeoutCancel()

	startDummy(ctx)

	fmt.Print("testing review...")
	out := runCli(ctx, "review", "ipv4://localhost:10302")
	grepFailedExit(out.String(), "posting analysis", 1)
	fmt.Println("OK!")

	fmt.Print("testing push...")
	out = runCli(ctx, "push", "ipv4://localhost:10302")
	grepFailedExit(out.String(), "dummy comment for push event", 1)
	fmt.Println("OK!")

	// next tests require analyzier started with UAST, restart analyzer
	stop()
	ctx, timeoutCancel = context.WithTimeout(context.Background(), timeout)
	defer timeoutCancel()
	startDummy(ctx, "--uast")

	fmt.Print("should return error without bblfsh...")
	out = runCli(ctx, "review", "ipv4://localhost:10302", "--bblfshd=ipv4://localhost:0000")
	grepFailedExit(out.String(), "WantUAST isn't allowed", 1)
	fmt.Println("OK!")

	fmt.Print("should notify about lack of uast...")
	out = runCli(ctx, "review", "ipv4://localhost:10302",
		"--from=66924f49aa9987273a137857c979ee5f0e709e30",
		"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
	grepFailedExit(out.String(), "The file doesn't have UAST", 1)
	fmt.Println("OK!")

	stop()
}

func runCli(ctx context.Context, cmd string, args ...string) bytes.Buffer {
	args = append([]string{cmd}, args...)

	var out bytes.Buffer
	reviewCmd := exec.CommandContext(ctx, lookoutBin, args...)
	reviewCmd.Stdout = &out
	reviewCmd.Stderr = &out
	handleErr(reviewCmd.Run(), "review command returned error", out)
	return out
}

func startDummy(ctx context.Context, args ...string) {
	args = append([]string{"serve"}, args...)

	analyzerOut.Reset()

	ctx, cancel := context.WithCancel(ctx)
	stop = func() {
		cancel()
		fmt.Println("stopping sevices")
		time.Sleep(time.Second) // go needs a bit of time to kill process
	}

	analyzerCmd := exec.CommandContext(ctx, dummyBin, args...)
	analyzerCmd.Stdout = &analyzerOut
	analyzerCmd.Stderr = &analyzerOut
	err := analyzerCmd.Start()
	if err != nil {
		fmt.Println("can't start analyzer:")
		fmt.Println(err)
		fmt.Printf("output:\n %s", analyzerOut.String())
		os.Exit(1)
	} else {
		go func() {
			if err := analyzerCmd.Wait(); err != nil {
				// don't print error if analyzer was killed by cancel
				if ctx.Err() != context.Canceled {
					fmt.Println("analyzer exited with error:", err)
					fmt.Printf("output:\n%s", analyzerOut.String())
					failExit()
				}
			}
		}()
	}
}

func handleErr(err error, desc string, out bytes.Buffer) {
	if err == nil {
		return
	}

	fmt.Printf("%s:\n", desc)
	fmt.Println(err)
	fmt.Printf("output:\n %s", out.String())
	fmt.Printf("analyzer output\n %s", analyzerOut.String())
	failExit()
}

func grepFailedExit(content, msg string, times int) {
	if strings.Count(content, msg) != times {
		fmt.Printf("'%s' not found in:\n%s", msg, content)
		fmt.Printf("analyzer output\n %s", analyzerOut.String())
		failExit()
	}
}

func failExit() {
	stop()

	os.Exit(1)
}

func readEnv() {
	if os.Getenv("DUMMY_BIN") != "" {
		dummyBin = os.Getenv("DUMMY_BIN")
	}
	if os.Getenv("LOOKOUT_BIN") != "" {
		lookoutBin = os.Getenv("LOOKOUT_BIN")
	}
}
