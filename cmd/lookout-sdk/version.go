package main

import "github.com/src-d/lookout/util/cli"

func init() {
	if _, err := app.AddCommand("version", "show version information", "",
		&cli.VersionCommand{
			Name:    name,
			Version: version,
			Build:   build,
		}); err != nil {
		panic(err)
	}
}
