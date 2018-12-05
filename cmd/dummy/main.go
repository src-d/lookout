package main

import (
	"context"
	"net/http"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/dummy"
	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/util/grpchelper"
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var (
	version = "local_build_1"
	app     = cli.New("dummy")
)

type ServeCommand struct {
	cli.CommonOptions
	Analyzer         string `long:"analyzer" default:"ipv4://0.0.0.0:9930" env:"LOOKOUT_ANALYZER" description:"gRPC URL to bind the analyzer to"`
	DataServer       string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL of the data server"`
	RequestUAST      bool   `long:"uast" env:"LOOKOUT_REQUEST_UAST" description:"analyzer will request UAST from the data server"`
	RequestFilesPush bool   `long:"files" env:"LOOKOUT_REQUEST_FILES" description:"on push events the analyzer will request files from HEAD, and return comments"`
	ProbesAddr       string `long:"probes-addr" default:"0.0.0.0:8091" env:"LOOKOUT_ANALYZER_PROBES_ADDRESS" description:"TCP address to bind the health probe endpoints"`
}

func (c *ServeCommand) Execute(args []string) error {
	c.initHealthProbes()

	var err error
	c.DataServer, err = pb.ToGoGrpcAddress(c.DataServer)
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
		Version:          version,
		DataClient:       lookout.NewDataClient(conn),
		RequestUAST:      c.RequestUAST,
		RequestFilesPush: c.RequestFilesPush,
	}

	server := grpchelper.NewServer()
	lookout.RegisterAnalyzerServer(server, a)

	lis, err := pb.Listen(c.Analyzer)
	if err != nil {
		return err
	}

	log.Infof("server has started on '%s'", c.Analyzer)
	return server.Serve(lis)
}

func (c *ServeCommand) initHealthProbes() {
	livenessPath := "/health/liveness"
	http.HandleFunc(livenessPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	readinessPath := "/health/readiness"
	http.HandleFunc(readinessPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))

	})

	go func() {
		log.With(log.Fields{
			"addr":  c.ProbesAddr,
			"paths": []string{livenessPath, readinessPath},
		}).Debugf("listening health probe HTTP requests")

		err := http.ListenAndServe(c.ProbesAddr, nil)
		if err != nil {
			log.Errorf(err, "ListenAndServe failed")
		}
	}()
}

func main() {
	if _, err := app.AddCommand("serve", "", "",
		&ServeCommand{}); err != nil {
		panic(err)
	}

	app.RunMain()
}
