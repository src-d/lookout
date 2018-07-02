package lookout

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/src-d/lookout/pb"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func setupDataServer(t *testing.T, dr *MockService) (*grpc.Server,
	pb.DataClient) {

	t.Helper()
	require := require.New(t)

	srv := &DataServerHandler{ChangeGetter: dr}
	grpcServer := grpc.NewServer()
	pb.RegisterDataServer(grpcServer, srv)

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	address := lis.Addr().String()

	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	require.NoError(err)

	client := pb.NewDataClient(conn)

	return grpcServer, client
}

func tearDownDataServer(t *testing.T, srv *grpc.Server) {
	if srv != nil {
		srv.Stop()
	}
}
func TestServerOk(t *testing.T) {
	for i := 0; i <= 10; i++ {
		req := &ChangesRequest{
			Head: &ReferencePointer{
				InternalRepositoryURL: "repo",
				Hash: "5262fd2b59d10e335a5c941140df16950958322d",
			},
		}
		changes := generateChanges(i)
		dr := &MockService{
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
			req := &ChangesRequest{
				Head: &ReferencePointer{
					InternalRepositoryURL: "repo",
					Hash: "5262fd2b59d10e335a5c941140df16950958322d",
				},
			}
			changes := generateChanges(i)
			tick := make(chan struct{}, 1)
			dr := &MockService{
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
	req := &ChangesRequest{
		Head: &ReferencePointer{
			InternalRepositoryURL: "repo",
			Hash: "5262fd2b59d10e335a5c941140df16950958322d",
		},
	}
	changes := generateChanges(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockService{
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
	req := &ChangesRequest{
		Head: &ReferencePointer{
			InternalRepositoryURL: "repo",
			Hash: "5262fd2b59d10e335a5c941140df16950958322d",
		},
	}
	changes := generateChanges(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockService{
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

func generateChanges(size int) []*Change {
	var changes []*Change
	for i := 0; i < size; i++ {
		changes = append(changes, &Change{
			Head: &File{
				Path: fmt.Sprintf("myfile%d", i),
			},
		})
	}

	return changes
}

type MockService struct {
	T               *testing.T
	ExpectedRequest *ChangesRequest
	ChangeScanner   ChangeScanner
	Error           error
}

func (r *MockService) GetChanges(ctx context.Context, req *ChangesRequest) (
	ChangeScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedRequest, req)
	return r.ChangeScanner, r.Error
}

type SliceChangeScanner struct {
	Changes    []*Change
	Error      error
	ChangeTick chan struct{}
	val        *Change
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

func (s *SliceChangeScanner) Change() *Change {
	if s.ChangeTick != nil {
		<-s.ChangeTick
	}

	return s.val
}

func (s *SliceChangeScanner) Close() error {
	return nil
}
