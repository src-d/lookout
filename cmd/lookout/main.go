package main

import (
	stdlog "log"
	"os"

	"github.com/src-d/lookout/util/flags"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/src-d/go-log.v1"
)

var (
	name    = "lookout"
	version = "undefined"
	build   = "undefined"
)

var parser = flags.NewParser()

func init() {
	log.DefaultLogger = log.New(log.Fields{"app": name})
}

func main() {
	flags.RunMain(parser)
}

func setGrpcLogger() {
	grpclog.SetLogger(stdlog.New(os.Stdout, "", stdlog.LstdFlags))
}
