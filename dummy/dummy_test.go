package dummy

import (
	"context"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/git"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git-fixtures.v3"
	gitsrv "gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

type DummySuite struct {
	suite.Suite
	Basic          *fixtures.Fixture
	analyzerServer *grpc.Server
	apiServer      *grpc.Server
	apiConn        *grpc.ClientConn
	apiClient      *lookout.DataClient
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
	server := &lookout.DataServerHandler{
		ChangeGetter: git.NewService(
			gitsrv.MapLoader{
				"repo:///fixture/basic": sto,
			},
		),
	}
	lookout.RegisterDataServer(s.apiServer, server)

	lis, err := lookout.Listen("ipv4://0.0.0.0:9991")
	require.NoError(err)

	go s.apiServer.Serve(lis)

	s.apiConn, err = grpc.Dial("0.0.0.0:9991", grpc.WithInsecure())
	require.NoError(err)

	s.apiClient = lookout.NewDataClient(s.apiConn)
}

func (s *DummySuite) TearDownSuite() {
	assert := s.Assert()

	if s.analyzerServer != nil {
		s.analyzerServer.Stop()
	}

	if s.apiServer != nil {
		s.apiServer.Stop()
	}

	if s.apiConn != nil {
		err := s.apiConn.Close()
		assert.NoError(err)
	}

	err := fixtures.Clean()
	assert.NoError(err)
}

func (s *DummySuite) Test() {
	require := s.Require()

	a := &Analyzer{
		DataClient: s.apiClient,
	}

	s.analyzerServer = grpc.NewServer()
	lookout.RegisterAnalyzerServer(s.analyzerServer, a)

	lis, err := lookout.Listen("ipv4://0.0.0.0:9995")
	require.NoError(err)

	done := make(chan error)
	go func() {
		done <- s.analyzerServer.Serve(lis)
	}()

	conn, err := grpc.Dial("0.0.0.0:9995", grpc.WithInsecure())
	require.NoError(err)

	client := lookout.NewAnalyzerClient(conn)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	resp, err := client.Analyze(ctx, &lookout.AnalysisRequest{
		Repository: "repo:///fixture/basic",
		NewHash:    s.Basic.Head.String(),
	})
	require.NoError(err)
	require.NotNil(resp)

	s.analyzerServer.Stop()
	require.NoError(<-done)
}
