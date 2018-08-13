package purge

import (
	"context"
	"testing"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/mock"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) TestChanges() {
	require := s.Require()

	underlying := &mock.MockChangesService{
		T: s.T(),
		ModifyReq: func(req *lookout.ChangesRequest) {
			req.WantContents = true
		},
	}
	srv := NewService(underlying, nil)
	require.NotNil(srv)

	expectedChanges := []*lookout.Change{
		&lookout.Change{
			Head: &lookout.File{
				Path:    "f1new.go",
				Content: []byte("f1 new"),
			},
		},
		&lookout.Change{
			Base: &lookout.File{
				Path:    "f2old.py",
				Content: []byte("f2 old"),
			},
			Head: &lookout.File{
				Path:    "f2new.js",
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
	}

	underlying.ExpectedRequest = req
	underlying.ChangeScanner = &mock.SliceChangeScanner{Changes: expectedChanges}

	scan, err := srv.GetChanges(context.TODO(), req)
	require.NoError(err)
	require.NotNil(scan)

	var changes []*lookout.Change
	for scan.Next() {
		changes = append(changes, scan.Change())
	}

	require.NoError(scan.Err())
	require.Equal(len(expectedChanges), len(changes))

	require.Nil(changes[0].Head.Content)
	require.Nil(changes[1].Base.Content)
	require.Nil(changes[1].Head.Content)

	require.NoError(scan.Close())
}

func (s *ServiceSuite) TestFiles() {
	require := s.Require()

	underlying := &mock.MockFilesService{
		T: s.T(),
		ModifyReq: func(req *lookout.FilesRequest) {
			req.WantContents = true
		},
	}
	srv := NewService(nil, underlying)
	require.NotNil(srv)

	expectedFiles := []*lookout.File{
		{
			Path:    "f1new.go",
			Content: []byte("f1 new"),
		},
		{
			Path:    "f2new.js",
			Content: []byte("f2 new"),
		}}
	req := &lookout.FilesRequest{
		Revision: &lookout.ReferencePointer{
			InternalRepositoryURL: "repo://myrepo",
			Hash: "foo",
		},
		WantLanguage: true,
	}

	underlying.ExpectedRequest = req
	underlying.FileScanner = &mock.SliceFileScanner{Files: expectedFiles}

	scan, err := srv.GetFiles(context.TODO(), req)
	require.NoError(err)
	require.NotNil(scan)

	var files []*lookout.File
	for scan.Next() {
		files = append(files, scan.File())
	}

	require.NoError(scan.Err())
	require.Equal(len(expectedFiles), len(files))

	require.Nil(files[0].Content)
	require.Nil(files[1].Content)

	require.NoError(scan.Close())
}
