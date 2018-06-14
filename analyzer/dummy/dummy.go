package dummy

import (
	"context"
	"io"

	"github.com/src-d/lookout/api"
	lserver "github.com/src-d/lookout/server"

	"google.golang.org/grpc"
)

type Analyzer struct {
	grpcServer *grpc.Server
}

func New() *Analyzer {
	return &Analyzer{}
}

var Default = New()

func (a *Analyzer) Serve(listen, dataServer string) error {
	dataServer, err := lserver.ToGoGrpcAddress(dataServer)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(dataServer,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.FailFast(false)))
	if err != nil {
		return err
	}

	dc := api.NewDataClient(conn)

	lis, err := lserver.Listen(listen)
	if err != nil {
		return err
	}

	a.grpcServer = grpc.NewServer()
	api.RegisterAnalyzerServer(a.grpcServer, &server{
		data: dc,
	})

	return a.grpcServer.Serve(lis)
}

func (a *Analyzer) Stop() {
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
}

type server struct {
	data api.DataClient
}

var _ api.AnalyzerServer = &server{}

func (s *server) Analyze(ctx context.Context, req *api.AnalysisRequest) (
	*api.AnalysisResponse, error) {

	client, err := s.data.GetChanges(ctx, &api.ChangesRequest{
		Repository: req.Repository,
		Top:        req.NewHash,
	})
	if err != nil {
		return nil, err
	}

	for {
		_, err := client.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			client.CloseSend()
			return nil, err
		}
	}

	return &api.AnalysisResponse{}, nil
}
