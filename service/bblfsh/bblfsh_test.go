package bblfsh

import (
	"context"
	"net"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	"gopkg.in/bblfsh/sdk.v1/uast"
)

type ServiceSuite struct {
	suite.Suite
	Mock         *MockBblfshServer
	BblfshServer *grpc.Server
	BblfshClient *grpc.ClientConn
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupSuite() {
	require := s.Require()
	s.Mock = &MockBblfshServer{}
	grpcServer := grpc.NewServer()
	protocol.RegisterProtocolServiceServer(grpcServer, s.Mock)

	lis, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(err)

	addr := lis.Addr().String()

	go grpcServer.Serve(lis)
	s.BblfshServer = grpcServer

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	require.NoError(err)

	s.BblfshClient = conn
}

func (s *ServiceSuite) TearDownSuite() {
	require := s.Require()

	if s.BblfshServer != nil {
		s.BblfshServer.GracefulStop()
	}

	if s.BblfshClient != nil {
		require.NoError(s.BblfshClient.Close())
	}
}

func (s *ServiceSuite) TestNoContents() {
	require := s.Require()

	underlying := &MockService{T: s.T()}
	srv := NewService(underlying, s.BblfshClient)
	require.NotNil(srv)

	expectedChanges := []*lookout.Change{
		&lookout.Change{
			Head: &lookout.File{
				Path:    "f1new",
				Content: []byte("f1 new"),
			},
		},
		&lookout.Change{
			Base: &lookout.File{
				Path:    "f2old",
				Content: []byte("f2 old"),
			},
			Head: &lookout.File{
				Path:    "f2new",
				Content: []byte("f2 new"),
			},
		}}
	req := &lookout.ChangesRequest{
		Base: &lookout.ReferencePointer{
			InternalRepositoryURL: "repo://myrepo",
			Hash: "foo",
		},
		Head: &lookout.ReferencePointer{
			InternalRepositoryURL: "repo://myrepo",
			Hash: "bar",
		},
		WantUAST: true,
	}

	underlying.ExpectedRequest = req
	underlying.ChangeScanner = &SliceChangeScanner{Changes: expectedChanges}

	s.Mock.Nodes = make(map[string]*uast.Node)
	s.Mock.Nodes["f1new"] = &uast.Node{InternalType: "f1 new"}
	s.Mock.Nodes["f2old"] = &uast.Node{InternalType: "f2 old"}
	s.Mock.Nodes["f2new"] = &uast.Node{InternalType: "f2 new"}

	scan, err := srv.GetChanges(req)
	require.NoError(err)
	require.NotNil(scan)

	var changes []*lookout.Change
	for scan.Next() {
		changes = append(changes, scan.Change())
	}

	require.NoError(scan.Err())
	require.Equal(len(expectedChanges), len(changes))
	for _, ch := range changes {
		require.Equal(s.Mock.Nodes[ch.Base.Path], ch.Base.UAST)
		require.Equal(s.Mock.Nodes[ch.Head.Path], ch.Head.UAST)
	}

	require.NoError(scan.Close())
}

type MockBblfshServer struct {
	protocol.ProtocolServiceServer
	Nodes map[string]*uast.Node
}

func (s *MockBblfshServer) Parse(ctx context.Context,
	req *protocol.ParseRequest) (*protocol.ParseResponse, error) {

	if s.Nodes == nil {
		return &protocol.ParseResponse{Response: protocol.Response{
			Status: protocol.Fatal,
		}}, nil
	}

	node, ok := s.Nodes[req.Filename]
	if !ok {
		return &protocol.ParseResponse{Response: protocol.Response{
			Status: protocol.Fatal,
		}}, nil
	}

	return &protocol.ParseResponse{
		Response: protocol.Response{Status: protocol.Ok},
		UAST:     node,
	}, nil
}

type MockService struct {
	T               *testing.T
	ExpectedRequest *lookout.ChangesRequest
	ChangeScanner   lookout.ChangeScanner
	Error           error
}

func (r *MockService) GetChanges(req *lookout.ChangesRequest) (
	lookout.ChangeScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedRequest, req)
	return r.ChangeScanner, r.Error
}

type SliceChangeScanner struct {
	Changes    []*lookout.Change
	Error      error
	ChangeTick chan struct{}
	val        *lookout.Change
}

func (s *SliceChangeScanner) Next() bool {
	if s.Error != nil {
		return false
	}

	if len(s.Changes) == 0 {
		s.val = nil
		return false
	}

	s.val, s.Changes = s.Changes[0], s.Changes[1:]
	return true
}

func (s *SliceChangeScanner) Err() error {
	return s.Error
}

func (s *SliceChangeScanner) Change() *lookout.Change {
	if s.ChangeTick != nil {
		<-s.ChangeTick
	}

	return s.val
}

func (s *SliceChangeScanner) Close() error {
	return nil
}
