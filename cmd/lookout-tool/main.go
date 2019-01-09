package main

import cli "gopkg.in/src-d/go-cli.v0"

var (
	name    = "lookout-tool"
	version = "undefined"
	build   = "undefined"
)

var app = cli.New(name, version, build, "Simplified version of the lookout server that works with a local git repository and does not need access to GitHub")

func main() {
	app.RunMain()
}
