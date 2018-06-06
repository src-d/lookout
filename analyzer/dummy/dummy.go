package dummy

import (
	"context"
	"io"
	"net"

	"github.com/src-d/lookout/api"

	"google.golang.org/grpc"
)

type Analyzer struct {
}

func New() *Analyzer {
	return &Analyzer{}
}

var Default = New()

func (a *Analyzer) Listen() error {
	conn, err := grpc.Dial("0.0.0.0:9991", grpc.WithInsecure())
	if err != nil {
		return err
	}

	dc := api.NewDataClient(conn)

	lis, err := net.Listen("tcp", "0.0.0.0:9995")
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	api.RegisterAnalyzerServer(srv, &server{
		data: dc,
	})
	go srv.Serve(lis)

	return nil
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
