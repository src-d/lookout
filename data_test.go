package lookout

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

func setupDataServer(t *testing.T, dr *MockService) (*grpc.Server,
	pb.DataClient) {

	t.Helper()
	require := require.New(t)

	srv := &DataServerHandler{ChangeGetter: dr, FileGetter: dr}
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

func TestServerGetChangesOk(t *testing.T) {
	for i := 0; i <= 10; i++ {
		req := &ChangesRequest{
			Head: &ReferencePointer{
				InternalRepositoryURL: "repo",
				Hash: "5262fd2b59d10e335a5c941140df16950958322d",
			},
		}
		changes := generateChanges(i)
		dr := &MockService{
			T:                t,
			ExpectedCRequest: req,
			ChangeScanner:    &SliceChangeScanner{Changes: changes},
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

func TestServerGetFilesOk(t *testing.T) {
	for i := 0; i <= 10; i++ {
		req := &FilesRequest{
			Revision: &ReferencePointer{
				InternalRepositoryURL: "repo",
				Hash: "5262fd2b59d10e335a5c941140df16950958322d",
			},
		}
		files := generateFiles(i)
		dr := &MockService{
			T:                t,
			ExpectedFRequest: req,
			FileScanner:      &SliceFileScanner{Files: files},
		}
		srv, client := setupDataServer(t, dr)

		t.Run(fmt.Sprintf("size-%d", i), func(t *testing.T) {
			require := require.New(t)

			respClient, err := client.GetFiles(context.TODO(), req)
			require.NoError(err)
			require.NotNil(respClient)
			require.NoError(respClient.CloseSend())

			for _, change := range files {
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
			revision := &ReferencePointer{
				InternalRepositoryURL: "repo",
				Hash: "5262fd2b59d10e335a5c941140df16950958322d",
			}
			changesReq := &ChangesRequest{Head: revision}
			filesReq := &FilesRequest{Revision: revision}
			changes := generateChanges(i)
			files := generateFiles(i)
			changeTick := make(chan struct{}, 1)
			fileTick := make(chan struct{}, 1)
			dr := &MockService{
				T:                t,
				ExpectedCRequest: changesReq,
				ExpectedFRequest: filesReq,
				ChangeScanner: &SliceChangeScanner{
					Changes:    changes,
					ChangeTick: changeTick,
				},
				FileScanner: &SliceFileScanner{
					Files:    files,
					FileTick: fileTick,
				},
			}
			srv, client := setupDataServer(t, dr)

			t.Run(fmt.Sprintf("get-changes-size-%d-cancel-at-%d", i, j),
				func(t *testing.T) {
					require := require.New(t)

					ctx, cancel := context.WithCancel(context.Background())
					respClient, err := client.GetChanges(ctx, changesReq)
					require.NoError(err)
					require.NotNil(respClient)
					require.NoError(respClient.CloseSend())

					for idx, change := range changes {
						if idx >= j {
							break
						}

						changeTick <- struct{}{}
						actualResp, err := respClient.Recv()
						require.NoError(err)
						require.Equal(change, actualResp)
					}

					cancel()
					changeTick <- struct{}{}
					actualResp, err := respClient.Recv()
					require.Error(err)
					require.Contains(err.Error(), "context cancel")
					require.Zero(actualResp)
				})

			t.Run(fmt.Sprintf("get-files-size-%d-cancel-at-%d", i, j),
				func(t *testing.T) {
					require := require.New(t)

					ctx, cancel := context.WithCancel(context.Background())
					respClient, err := client.GetFiles(ctx, filesReq)
					require.NoError(err)
					require.NotNil(respClient)
					require.NoError(respClient.CloseSend())

					for idx, file := range files {
						if idx >= j {
							break
						}

						fileTick <- struct{}{}
						actualResp, err := respClient.Recv()
						require.NoError(err)
						require.Equal(file, actualResp)
					}

					cancel()
					fileTick <- struct{}{}
					actualResp, err := respClient.Recv()
					require.Error(err)
					require.Contains(err.Error(), "context cancel")
					require.Zero(actualResp)
				})

			close(changeTick)
			close(fileTick)
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
		T:                t,
		ExpectedCRequest: req,
		Error:            ExpectedError,
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

func TestServerGetFilesError(t *testing.T) {
	req := &FilesRequest{
		Revision: &ReferencePointer{
			InternalRepositoryURL: "repo",
			Hash: "5262fd2b59d10e335a5c941140df16950958322d",
		},
	}
	files := generateFiles(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockService{
		T:                t,
		ExpectedFRequest: req,
		Error:            ExpectedError,
		FileScanner:      &SliceFileScanner{Files: files},
	}
	srv, client := setupDataServer(t, dr)

	t.Run("test", func(t *testing.T) {
		require := require.New(t)
		respClient, err := client.GetFiles(context.TODO(), req)
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
		T:                t,
		ExpectedCRequest: req,
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

func TestServerGetFilesIterError(t *testing.T) {
	req := &FilesRequest{
		Revision: &ReferencePointer{
			InternalRepositoryURL: "repo",
			Hash: "5262fd2b59d10e335a5c941140df16950958322d",
		},
	}
	files := generateFiles(10)
	ExpectedError := fmt.Errorf("TEST ERROR")
	dr := &MockService{
		T:                t,
		ExpectedFRequest: req,
		FileScanner: &SliceFileScanner{
			Files: files,
			Error: ExpectedError,
		},
	}
	srv, client := setupDataServer(t, dr)

	t.Run("test", func(t *testing.T) {
		require := require.New(t)
		respClient, err := client.GetFiles(context.TODO(), req)
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

func generateFiles(size int) []*File {
	var files []*File
	for i := 0; i < size; i++ {
		files = append(files, &File{
			Path: fmt.Sprintf("myfile%d", i),
		})
	}

	return files
}

type MockService struct {
	T                *testing.T
	ExpectedCRequest *ChangesRequest
	ExpectedFRequest *FilesRequest
	ChangeScanner    ChangeScanner
	FileScanner      FileScanner
	Error            error
}

func (r *MockService) GetChanges(ctx context.Context, req *ChangesRequest) (
	ChangeScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedCRequest, req)
	return r.ChangeScanner, r.Error
}

func (r *MockService) GetFiles(ctx context.Context, req *FilesRequest) (
	FileScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedFRequest, req)
	return r.FileScanner, r.Error
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

type SliceFileScanner struct {
	Files    []*File
	Error    error
	FileTick chan struct{}
	val      *File
}

func (s *SliceFileScanner) Next() bool {
	if s.Error != nil {
		return false
	}

	if len(s.Files) == 0 {
		s.val = nil
		return false
	}

	s.val, s.Files = s.Files[0], s.Files[1:]
	return true
}

func (s *SliceFileScanner) Err() error {
	return s.Error
}

func (s *SliceFileScanner) File() *File {
	if s.FileTick != nil {
		<-s.FileTick
	}

	return s.val
}

func (s *SliceFileScanner) Close() error {
	return nil
}

func TestFnFileScanner(t *testing.T) {
	require := require.New(t)

	files := generateFiles(3)

	sliceScanner := &SliceFileScanner{Files: files}

	fn := func(f *File) (bool, error) {
		if strings.HasSuffix(f.Path, "2") {
			return true, nil
		}
		return false, nil
	}

	s := FnFileScanner{
		Scanner: sliceScanner,
		Fn:      fn,
	}

	var scannedFiles []*File
	for s.Next() {
		scannedFiles = append(scannedFiles, s.File())
	}

	require.False(s.Next())
	require.NoError(s.Err())
	require.NoError(s.Close())

	require.Len(scannedFiles, 2)
}

func TestFnChangeScanner(t *testing.T) {
	require := require.New(t)

	changes := generateChanges(3)

	sliceScanner := &SliceChangeScanner{Changes: changes}

	fn := func(c *Change) (bool, error) {
		if strings.HasSuffix(c.Head.Path, "2") {
			return true, nil
		}
		return false, nil
	}

	s := FnChangeScanner{
		Scanner: sliceScanner,
		Fn:      fn,
	}

	var scannedChanges []*Change
	for s.Next() {
		scannedChanges = append(scannedChanges, s.Change())
	}

	require.False(s.Next())
	require.NoError(s.Err())
	require.NoError(s.Close())

	require.Len(scannedChanges, 2)
}
