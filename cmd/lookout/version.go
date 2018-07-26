package main

import "fmt"

// VersionCommand represents the `version` command of lookout CLI.
type VersionCommand struct {
	// Name of the binary
	Name string
	// Version of the binary
	Version string
	// Build of the binary
	Build string
}

func init() {
	if _, err := app.AddCommand("version", "show version information", "",
		&VersionCommand{
			Name:    name,
			Version: version,
			Build:   build,
		}); err != nil {
		panic(err)
	}
}

// Execute prints the build information provided at the compilation time.
func (v *VersionCommand) Execute(args []string) error {
	fmt.Printf("%s %s built on %s\n", v.Name, v.Version, v.Build)
	return nil
}
