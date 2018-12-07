package main

import "gopkg.in/src-d/go-cli.v0"

var (
	name    = "lookoutd"
	version = "undefined"
	build   = "undefined"
)

var app = cli.New(name, version, build, "A service for assisted code review")

func main() {
	app.RunMain()
}
