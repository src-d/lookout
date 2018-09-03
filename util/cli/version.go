package cli

import "fmt"

// VersionCommand represents the `version` command of the CLI.
type VersionCommand struct {
	// Name of the binary
	Name string
	// Version of the binary
	Version string
	// Build of the binary
	Build string
}

// Execute prints the build information provided at the compilation time.
func (v *VersionCommand) Execute(args []string) error {
	fmt.Printf("%s %s built on %s\n", v.Name, v.Version, v.Build)
	return nil
}
