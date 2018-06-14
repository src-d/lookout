package lookout

import (
	"context"
	"fmt"
	"io"

	"github.com/src-d/lookout/pb"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"

	"google.golang.org/grpc"
)

type ChangesRequest = pb.ChangesRequest
type Change = pb.Change
type File = pb.File

type ChangeGetter interface {
	GetChanges(*ChangesRequest) (ChangeScanner, error)
}

func RegisterDataServer(s *grpc.Server, srv *DataServerHandler) {
	pb.RegisterDataServer(s, srv)
}

type ChangeScanner interface {
	Next() bool
	Err() error
	Change() *Change
	Close() error
}

type DataServerHandler struct {
	ChangeGetter ChangeGetter
}

var _ pb.DataServer = &DataServerHandler{}

func (s *DataServerHandler) GetChanges(req *ChangesRequest,
	srv pb.Data_GetChangesServer) (err error) {

	ctx := srv.Context()
	cancel := ctx.Done()
	iter, err := s.ChangeGetter.GetChanges(req)
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(iter, &err)

	for iter.Next() {
		select {
		case <-cancel:
			return fmt.Errorf("request canceled: %s", ctx.Err())
		default:
		}

		if err := srv.Send(iter.Change()); err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return err
}

type DataClient struct {
	dataClient pb.DataClient
}

func NewDataClient(cc *grpc.ClientConn) *DataClient {
	return &DataClient{
		dataClient: pb.NewDataClient(cc),
	}
}

func (c *DataClient) GetChanges(ctx context.Context, in *ChangesRequest, opts ...grpc.CallOption) (
	ChangeScanner, error) {

	resp, err := c.dataClient.GetChanges(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	if err := resp.CloseSend(); err != nil {
		return nil, err
	}

	return &ClientChangeScanner{
		client: resp,
	}, nil

}

type ClientChangeScanner struct {
	client pb.Data_GetChangesClient
	val    *Change
	err    error
	done   bool
}

func (s *ClientChangeScanner) Next() bool {
	if s.done {
		return false
	}

	s.val, s.err = s.client.Recv()
	if s.err == io.EOF {
		s.err = nil
		s.done = true
		return false
	}

	if s.err != nil {
		s.done = true
		return false
	}

	return true
}

func (s *ClientChangeScanner) Err() error {
	return s.err
}

func (s *ClientChangeScanner) Change() *Change {
	return s.val
}

func (s *ClientChangeScanner) Close() error {
	return nil
}
