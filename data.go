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

// ChangeGetter is used to retrieve code changes.
type ChangeGetter interface {
	// GetChanges returns a ChangeScanner that scans all changes according
	// to the request.
	GetChanges(context.Context, *ChangesRequest) (ChangeScanner, error)
}

func RegisterDataServer(s *grpc.Server, srv *DataServerHandler) {
	pb.RegisterDataServer(s, srv)
}

// ChangeScanner is a scanner for changes.
type ChangeScanner interface {
	// Next advances the scanner to the next change. It returns true if a new
	// change is found, and false otherwise. After the user is done scanning,
	// Err must be called to check if all changes were consumed or there was an
	// error.
	Next() bool
	// Err returns any error found during scanning.
	Err() error
	// Change returns the current change.
	Change() *Change
	// Close closes the scanner.
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
	iter, err := s.ChangeGetter.GetChanges(ctx, req)
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
