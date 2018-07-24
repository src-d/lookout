package main

import (
	stdlog "log"
	"os"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/src-d/go-log.v1"
)

const maxMsgSize = 1024 * 1024 * 100 // 100mb

var (
	name    = "lookout"
	version = "undefined"
	build   = "undefined"
)

var parser = flags.NewParser(nil, flags.Default)

func init() {
	log.DefaultLogger = log.New(log.Fields{"app": name})
}

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

func setGrpcLogger() {
	grpclog.SetLogger(stdlog.New(os.Stdout, "", stdlog.LstdFlags))
}
