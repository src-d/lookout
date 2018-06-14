package dummy

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/src-d/lookout/api"
	lserver "github.com/src-d/lookout/server"

	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git.v4/utils/binary"
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
		Repository:   req.Repository,
		Top:          req.NewHash,
		WantContents: true,
	})
	if err != nil {
		return nil, err
	}

	client.CloseSend()
	resp := &api.AnalysisResponse{}
	for {
		change, err := client.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		resp.Comments = append(resp.Comments, s.lineIncrease(change)...)
		resp.Comments = append(resp.Comments, s.maxLineWidth(change)...)
	}

	return resp, nil
}

func (a *server) lineIncrease(ch *api.Change) []*api.Comment {
	if a.isBinary(ch.GetOld()) || a.isBinary(ch.GetNew()) {
		return nil
	}

	diff := a.countLines(ch.GetNew()) - a.countLines(ch.GetOld())
	if diff <= 0 {
		return nil
	}

	return []*api.Comment{{
		File: ch.GetNew().Path,
		Line: int32(0),
		Text: fmt.Sprintf("The file has increased in %d lines.", diff),
	}}
}

func (a *server) maxLineWidth(ch *api.Change) []*api.Comment {
	lines := bytes.Split(ch.GetNew().GetContent(), []byte("\n"))
	var comments []*api.Comment
	for i, line := range lines {
		if len(line) > 80 {
			comments = append(comments, &api.Comment{
				File: ch.GetNew().GetPath(),
				Line: int32(i + 1),
				Text: "This line exceeded 80 bytes.",
			})
		}
	}

	return comments
}

func (a *server) isBinary(f *api.File) bool {
	contents := f.GetContent()
	ok, err := binary.IsBinary(bytes.NewReader(contents))
	return err != nil || ok
}

func (a *server) countLines(f *api.File) int {
	return bytes.Count(f.GetContent(), []byte("\n"))
}
