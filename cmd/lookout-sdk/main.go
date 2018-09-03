package main

import "github.com/src-d/lookout/util/cli"

var (
	name    = "lookout-sdk"
	version = "undefined"
	build   = "undefined"
)

var app = cli.New(name)

func main() {
	app.RunMain()
}
