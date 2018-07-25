package main

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/dummy"
	"github.com/src-d/lookout/util/flags"
	"github.com/src-d/lookout/util/grpchelper"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/grpclog/glogger"
)

var (
	version = "local_build_1"
	parser  = flags.NewParser()
)

type ServeCommand struct {
	flags.CommonOptions
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL to bind the analyzer to"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"grPC URL of the data server"`
}

func (c *ServeCommand) Execute(args []string) error {
	var err error
	c.DataServer, err = grpchelper.ToGoGrpcAddress(c.DataServer)
	if err != nil {
		return err
	}

	conn, err := grpchelper.DialContext(
		context.Background(),
		c.DataServer,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.FailFast(false)),
	)
	if err != nil {
		return err
	}

	a := &dummy.Analyzer{
		Version:    version,
		DataClient: lookout.NewDataClient(conn),
	}

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := grpchelper.Listen(c.Analyzer)
	if err != nil {
		return err
	}

	return server.Serve(lis)
}

func main() {
	if _, err := parser.AddCommand("serve", "", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}

	flags.RunMain(parser)
}
