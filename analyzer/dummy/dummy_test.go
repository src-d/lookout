package dummy

import (
	"context"
	"net"
	"testing"

	"github.com/src-d/lookout/api"
	"github.com/src-d/lookout/git"
	apisrv "github.com/src-d/lookout/server"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git-fixtures.v3"
	gitsrv "gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type DummySuite struct {
	suite.Suite
	Basic     *fixtures.Fixture
	apiServer *grpc.Server
}

func TestDummySuite(t *testing.T) {
	suite.Run(t, new(DummySuite))
}

func (s *DummySuite) SetupSuite() {
	require := s.Require()

	err := fixtures.Init()
	require.NoError(err)

	fixture := fixtures.Basic().One()
	s.Basic = fixture
	fs := fixture.DotGit()
	sto, err := filesystem.NewStorage(fs)
	require.NoError(err)

	s.apiServer = grpc.NewServer()
	server := apisrv.NewServer(git.NewService(
		gitsrv.MapLoader{
			"repo:///fixture/basic": sto,
		},
	))
	api.RegisterDataServer(s.apiServer, server)

	lis, err := net.Listen("tcp", "0.0.0.0:9991")
	require.NoError(err)

	go s.apiServer.Serve(lis)
}

func (s *DummySuite) TearDownSuite() {
	require := s.Require()

	if s.apiServer != nil {
		s.apiServer.Stop()
	}

	err := fixtures.Clean()
	require.NoError(err)
}

func (s *DummySuite) Test() {
	require := s.Require()

	a := &Analyzer{}
	require.NoError(a.Listen())

	conn, err := grpc.Dial("0.0.0.0:9995", grpc.WithInsecure())
	require.NoError(err)

	client := api.NewAnalyzerClient(conn)
	resp, err := client.Analyze(context.TODO(), &api.AnalysisRequest{
		Repository: "repo:///fixture/basic",
		NewHash:    s.Basic.Head.String(),
	})
	require.NoError(err)
	require.NotNil(resp)
}
