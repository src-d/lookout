package main

import (
	"fmt"

	"github.com/src-d/lookout/util/cmdtest"
)

func main() {
	fmt.Println("start integration testing")

	ctx, stop := cmdtest.StoppableCtx()
	cmdtest.StartDummy(ctx)
	testCase("testing review", func() {
		r := cmdtest.RunCli(ctx, "review", "ipv4://localhost:10302")
		cmdtest.GrepTrue(r, "posting analysis")
	})

	testCase("testing push", func() {
		r := cmdtest.RunCli(ctx, "push", "ipv4://localhost:10302")
		cmdtest.GrepTrue(r, "dummy comment for push event")
	})

	// next tests require analyzier started with UAST, restart analyzer
	stop()
	ctx, stop = cmdtest.StoppableCtx()
	cmdtest.StartDummy(ctx, "--uast")

	testCase("should return error without bblfsh", func() {
		r := cmdtest.RunCli(ctx, "review", "ipv4://localhost:10302", "--bblfshd=ipv4://localhost:0000")
		cmdtest.GrepTrue(r, "WantUAST isn't allowed")
	})

	testCase("should notify about lack of uast", func() {
		r := cmdtest.RunCli(ctx, "review", "ipv4://localhost:10302",
			"--from=66924f49aa9987273a137857c979ee5f0e709e30",
			"--to=2c9f56bcb55be47cf35d40d024ec755399f699c7")
		cmdtest.GrepTrue(r, "The file doesn't have UAST")
	})

	stop()
}

func testCase(name string, fn func()) {
	fmt.Print(name + "...")
	fn()
	fmt.Println("OK!")
}
