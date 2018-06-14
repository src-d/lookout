package main

import (
	"os"

	"github.com/src-d/lookout/analyzer/dummy"

	"github.com/jessevdk/go-flags"
	_ "google.golang.org/grpc/grpclog/glogger"
)

var parser = flags.NewParser(nil, flags.Default)

type ServeCommand struct {
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL to bind the analyzer to"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"grPC URL of the data server"`
}

func (c *ServeCommand) Execute(args []string) error {
	return dummy.Default.Serve(c.Analyzer, c.DataServer)
}

func main() {
	if _, err := parser.AddCommand("serve", "", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.Parse(); err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				os.Exit(0)
			}

			parser.WriteHelp(os.Stdout)
		}

		os.Exit(1)
	}
}
