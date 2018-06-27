package bblfsh

import (
	"context"

	"github.com/src-d/lookout"

	"google.golang.org/grpc"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	"gopkg.in/bblfsh/sdk.v1/uast"
)

type Service struct {
	underlying lookout.ChangeGetter
	client     protocol.ProtocolServiceClient
}

var _ lookout.ChangeGetter = &Service{}

func NewService(underlying lookout.ChangeGetter, conn *grpc.ClientConn) *Service {
	return &Service{
		underlying: underlying,
		client:     protocol.NewProtocolServiceClient(conn),
	}
}

func (s *Service) GetChanges(req *lookout.ChangesRequest) (
	lookout.ChangeScanner, error) {

	wantContents := req.WantContents
	if req.WantUAST {
		req.WantContents = true
	}

	changes, err := s.underlying.GetChanges(req)
	if err != nil {
		return nil, err
	}

	if !req.WantUAST {
		return changes, nil
	}

	return &ChangeScanner{
		underlying:    changes,
		client:        s.client,
		purgeContents: !wantContents,
	}, nil
}

type ChangeScanner struct {
	underlying    lookout.ChangeScanner
	client        protocol.ProtocolServiceClient
	purgeContents bool
	val           *lookout.Change
	err           error
}

func (s *ChangeScanner) Next() bool {
	if s.err != nil {
		return false
	}

	if !s.underlying.Next() {
		return false
	}

	s.val = s.underlying.Change()

	if err := s.processFile(s.val.Base); err != nil {
		s.err = err
		return false
	}

	if err := s.processFile(s.val.Head); err != nil {
		s.err = err
		return false
	}

	return true
}

func (s *ChangeScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.underlying.Err()
}

func (s *ChangeScanner) Change() *lookout.Change {
	return s.val
}

func (s *ChangeScanner) Close() error {
	return s.underlying.Close()
}

func (s *ChangeScanner) processFile(f *lookout.File) error {
	if f == nil {
		return nil
	}

	var err error
	f.UAST, err = s.parseFile(f)
	if err != nil {
		return err
	}

	if s.purgeContents {
		f.Content = nil
	}

	return nil
}

func (s *ChangeScanner) parseFile(f *lookout.File) (
	*uast.Node, error) {
	if f.Path == "" {
		return nil, nil
	}

	req := &protocol.ParseRequest{
		Filename: f.Path,
		Content:  string(f.Content),
		Encoding: protocol.UTF8,
	}
	resp, err := s.client.Parse(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	if resp.Status != protocol.Ok {
		return nil, nil
	}

	return resp.UAST, nil
}
