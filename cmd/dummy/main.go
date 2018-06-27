package main

import (
	"os"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/dummy"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/grpclog/glogger"
)

var parser = flags.NewParser(nil, flags.Default)

type ServeCommand struct {
	Analyzer   string `long:"analyzer" default:"ipv4://localhost:10302" env:"LOOKOUT_ANALYZER" description:"gRPC URL to bind the analyzer to"`
	DataServer string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"grPC URL of the data server"`
}

func (c *ServeCommand) Execute(args []string) error {
	var err error
	c.DataServer, err = lookout.ToGoGrpcAddress(c.DataServer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.DataServer,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.FailFast(false)),
	)
	if err != nil {
		return err
	}

	a := &dummy.Analyzer{
		DataClient: lookout.NewDataClient(conn),
	}

	server := grpc.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := lookout.Listen(c.Analyzer)
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
