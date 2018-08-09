package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/cmdtest"
	"github.com/src-d/lookout/util/grpchelper"
	log "gopkg.in/src-d/go-log.v1"
)

func main() {
	fmt.Println("start integration testing")

	grepTrue := cmdtest.GrepTrue
	grepAndNot := cmdtest.GrepAndNot

	cmdtest.ResetDB()

	ctx, stop := cmdtest.StoppableCtx()
	cmdtest.StartDummy(ctx)
	r, w := cmdtest.StartServe(ctx, "--provider", "json", "dummy-repo-url")

	// make sure server started correctly
	grepTrue(r, "Starting watcher")

	successJSON := `{"event":"review", "internal_id": "1", "number": 1, "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"4eebef102d7979570aadf69ff54ae1ffcca7ce00"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"d304499cb2a9cad3ea260f06ad59c1658db4763d"}}}`
	testCase("success review", func() {
		sendEvent(w, successJSON)
		grepTrue(r, "processing pull request")
		grepTrue(r, `{"analyzer-name":"Dummy","file":"provider/common.go","text":"The file has increased in 5 lines."}`)
		grepTrue(r, `msg="New status" status=success`)
	})

	testCase("skip review event", func() {
		sendEvent(w, successJSON)
		grepTrue(r, `event successfully processed, skipping...`)
	})

	testCase("process review but don't post anything", func() {
		json := `{"event":"review", "internal_id": "2", "number": 1, "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"4eebef102d7979570aadf69ff54ae1ffcca7ce00"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"d304499cb2a9cad3ea260f06ad59c1658db4763d"}}}`
		sendEvent(w, json)
		grepTrue(r, "processing pull request")
		grepAndNot(r, `msg="New status" status=success`, `posting analysis`)
	})

	testCase("wrong commit revision", func() {
		json := `{"event":"review", "internal_id": "3", "number": 3, "commit_revision": {"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"0000000000000000000000000000000000000000"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"0000000000000000000000000000000000000000"}}}`
		sendEvent(w, json)
		grepTrue(r, `event processing failed`)
	})

	testCase("success push", func() {
		successPushJSON := `{"event":"push", "internal_id": "1", "commit_revision":{"base":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"4eebef102d7979570aadf69ff54ae1ffcca7ce00"},"head":{"internal_repository_url":"https://github.com/src-d/lookout.git","reference_name":"refs/heads/master","hash":"d304499cb2a9cad3ea260f06ad59c1658db4763d"}}}`
		sendEvent(w, successPushJSON)
		grepTrue(r, "processing push")
		grepTrue(r, "comments can belong only to review event but 1 is given")
		grepTrue(r, `msg="New status" status=success`)
	})

	// restart server with multiple analyzers
	stop()
	cmdtest.ResetDB()

	ctx, stop = cmdtest.StoppableCtx()
	cmdtest.StartDummy(ctx)
	cmdtest.StartDummy(ctx, "--analyzer", "ipv4://localhost:10303")
	r, w = cmdtest.StartServe(ctx, "--provider", "json", "-c", "fixtures/double_dummy_config.yml", "dummy-repo-url")

	grepTrue(r, "Starting watcher")

	testCase("multiple analyzers", func() {
		sendEvent(w, successJSON)
		grepTrue(r, "processing pull request")
		grepTrue(r, "posting analysis")
		found, buf := cmdtest.Grep(r, `msg="New status" status=success`)
		if !found {
			fmt.Printf("'%s' not found in:\n%s", `msg="New status" status=success`, buf.String())
			stop()
			os.Exit(1)
		}
		if !strings.Contains(
			buf.String(),
			`{"analyzer-name":"Dummy1","file":"provider/common.go","text":"The file has increased in 5 lines."}`) {
			fmt.Println("no comments from the first analyzer")
			stop()
			os.Exit(1)
		}
		if !strings.Contains(
			buf.String(),
			`{"analyzer-name":"Dummy2","file":"provider/common.go","text":"The file has increased in 5 lines."}`) {
			fmt.Println("no comments from the second analyzer")
			stop()
			os.Exit(1)
		}
	})

	// restart server with error anylyzer
	stop()
	cmdtest.ResetDB()

	ctx, stop = cmdtest.StoppableCtx()
	startAnalyzer(ctx, &errAnalyzer{})
	r, w = cmdtest.StartServe(ctx, "--provider", "json", "dummy-repo-url")
	testCase("error from analyzer", func() {
		sendEvent(w, successJSON)
		grepTrue(r, `msg="analysis failed" analyzer=Dummy app=lookout error="rpc error: code = Unknown desc = review error"`)
	})

	stop()
}

func startAnalyzer(ctx context.Context, a lookout.AnalyzerServer) error {
	log.DefaultFactory = &log.LoggerFactory{
		Level: log.ErrorLevel,
	}
	log.DefaultLogger = log.New(log.Fields{"app": "test"})

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := grpchelper.Listen("ipv4://localhost:10302")
	if err != nil {
		return err
	}

	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.Stop()
	}()
	return nil
}

type errAnalyzer struct{}

func (a *errAnalyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("review error")
}

func (a *errAnalyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	return nil, errors.New("push error")
}

func sendEvent(w io.Writer, json string) {
	_, err := fmt.Fprintln(w, json)
	if err != nil {
		fmt.Println("can't write event:", err)
		os.Exit(1)
	}
}

func testCase(name string, fn func()) {
	fmt.Print(name + "...")
	fn()
	fmt.Println("OK!")
}
