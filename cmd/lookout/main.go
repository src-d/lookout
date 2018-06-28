package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	_ "google.golang.org/grpc/grpclog/glogger"
)

var parser = flags.NewParser(nil, flags.Default)

func main() {
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
