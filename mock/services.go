package mock

import (
	"context"
	"testing"

	"github.com/src-d/lookout"
	"github.com/stretchr/testify/require"
)

type MockChangesService struct {
	T               *testing.T
	ExpectedRequest *lookout.ChangesRequest
	ChangeScanner   lookout.ChangeScanner
	Error           error
	ModifyReq       func(req *lookout.ChangesRequest)
}

func (r *MockChangesService) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (lookout.ChangeScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedRequest, req)
	if r.ModifyReq != nil {
		r.ModifyReq(req)
	}
	return r.ChangeScanner, r.Error
}

type MockFilesService struct {
	T               *testing.T
	ExpectedRequest *lookout.FilesRequest
	FileScanner     lookout.FileScanner
	Error           error
	ModifyReq       func(req *lookout.FilesRequest)
}

func (r *MockFilesService) GetFiles(ctx context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	require := require.New(r.T)
	require.Equal(r.ExpectedRequest, req)
	if r.ModifyReq != nil {
		r.ModifyReq(req)
	}
	return r.FileScanner, r.Error
}
