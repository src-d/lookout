package server_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/src-d/lookout/api"
	. "github.com/src-d/lookout/server"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func setupDataServer(t *testing.T, dr *MockDataReader) (*grpc.Server,
	api.DataClient) {

	t.Helper()
	require := require.New(t)

	srv := NewServer(dr)
	grpcServer := grpc.NewServer()
	api.RegisterDataServer(grpcServer, srv)

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	address := lis.Addr().String()

	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	require.NoError(err)

	client := api.NewDataClient(conn)

	return grpcServer, client
}

func tearDownDataServer(t *testing.T, srv *grpc.Server) {
	if srv != nil {
		srv.Stop()
	}
}
func TestServerOk(t *testing.T) {
	for i := 0; i <= 10; i++ {
		req := &api.ChangesRequest{
			Repository: "repo",
			Top:        "5262fd2b59d10e335a5c941140df16950958322d",
		}
		changes := generateChanges(i)
		dr := &MockDataReader{
			T:               t,
			ExpectedRequest: req,
			ChangeScanner:   &SliceChangeScanner{Changes: changes},
		}
		srv, client := setupDataServer(t, dr)

		t.Run(fmt.Sprintf("size-%d", i), func(t *testing.T) {
			require := require.New(t)

			respClient, err := client.GetChanges(context.TODO(), req)
			require.NoError(err)
			require.NotNil(respClient)
			require.NoError(respClient.CloseSend())

			for _, change := range changes {
				actualResp, err := respClient.Recv()
				require.NoError(err)
				require.Equal(change, actualResp)
			}

			actualResp, err := respClient.Recv()
			require.Equal(io.EOF, err)
			require.Zero(actualResp)
		})

		tearDownDataServer(t, srv)
	}
}

func TestServerCancel(t *testing.T) {
	for i := 0; i <= 10; i++ {
		for j := 0; j < i; j++ {
			req := &api.ChangesRequest{
				Repository: "repo",
				Top:        "5262fd2b59d10e335a5c941140df16950958322d",
			}
			changes := generateChanges(i)
			tick := make(chan struct{}, 1)
			dr := &MockDataReader{
				T:               t,
				ExpectedRequest: req,
				ChangeScanner: &SliceChangeScanner{
					Changes:    changes,
					ChangeTick: tick,
				},
			}
			srv, client := setupDataServer(t, dr)

			t.Run(fmt.Sprintf("size-%d-cancel-at-%d", i, j),
				func(t *testing.T) {
					require := require.New(t)

					ctx, cancel := context.WithCancel(context.Background())
					respClient, err := client.GetChanges(ctx, req)
					require.NoError(err)
					require.NotNil(respClient)
					require.NoError(respClient.CloseSend())

					for idx, change := range changes {
						if idx >= j {
							break
						}

						tick <- struct{}{}
						actualResp, err := respClient.Recv()
						require.NoError(err)
						require.Equal(change, actualResp)
					}

					cancel()
					tick <- struct{}{}
					actualResp, err := respClient.Recv()
					require.Error(err)
					require.Contains(err.Error(), "context cancel")
					require.Zero(actualResp)
				})

			close(tick)
			tearDownDataServer(t, srv)
		}
	}
}

func TestServerGetChangesError(t *testing.T) {
	req := &api.ChangesRequest{
		Repository: "repo",
		Top:        "5262fd2b59d10e335a5c941140df16950958322d",
	}
	changes := generateChanges(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockDataReader{
		T:               t,
		ExpectedRequest: req,
		Error:           ExpectedError,
		ChangeScanner: &SliceChangeScanner{
			Changes: changes,
		},
	}
	srv, client := setupDataServer(t, dr)

	t.Run("test", func(t *testing.T) {
		require := require.New(t)
		respClient, err := client.GetChanges(context.TODO(), req)
		require.NoError(err)
		require.NotNil(respClient)

		change, err := respClient.Recv()
		require.Error(err)
		require.Contains(err.Error(), ExpectedError.Error())
		require.Zero(change)
	})

	tearDownDataServer(t, srv)
}

func TestServerGetChangesIterError(t *testing.T) {
	req := &api.ChangesRequest{
		Repository: "repo",
		Top:        "5262fd2b59d10e335a5c941140df16950958322d",
	}
	changes := generateChanges(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockDataReader{
		T:               t,
		ExpectedRequest: req,
		ChangeScanner: &SliceChangeScanner{
			Changes: changes,
			Error:   ExpectedError,
		},
	}
	srv, client := setupDataServer(t, dr)

	t.Run("test", func(t *testing.T) {
		require := require.New(t)
		respClient, err := client.GetChanges(context.TODO(), req)
		require.NoError(err)
		require.NotNil(respClient)

		change, err := respClient.Recv()
		require.Error(err)
		require.Contains(err.Error(), ExpectedError.Error())
		require.Zero(change)
	})

	tearDownDataServer(t, srv)
}

func generateChanges(size int) []*api.ChangesResponse {
	var changes []*api.ChangesResponse
	for i := 0; i < size; i++ {
		changes = append(changes, &api.ChangesResponse{
			Change: &api.Change{
				New: &api.File{
					Path: fmt.Sprintf("myfile%d", i),
				},
			},
		})
	}

	return changes
}

type MockDataReader struct {
	T               *testing.T
	ExpectedRequest *api.ChangesRequest
	ChangeScanner   ChangeScanner
	Error           error
}

func (r *MockDataReader) GetChanges(req *api.ChangesRequest) (
	ChangeScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedRequest, req)
	return r.ChangeScanner, r.Error
}

type SliceChangeScanner struct {
	Changes    []*api.ChangesResponse
	Error      error
	ChangeTick chan struct{}
	val        *api.ChangesResponse
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

func (s *SliceChangeScanner) Change() *api.ChangesResponse {
	if s.ChangeTick != nil {
		<-s.ChangeTick
	}

	return s.val
}

func (s *SliceChangeScanner) Close() error {
	return nil
}
