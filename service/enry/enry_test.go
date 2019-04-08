package enry

import (
	"context"
	"fmt"
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

func (s *ServiceSuite) TestChangesExcludeVendored() {
	require := s.Require()

	req := &lookout.ChangesRequest{ExcludeVendored: true}
	inputChanges := []*lookout.Change{
		// non-vendor files should survive
		{
			Head: &lookout.File{Path: "f1new.go"},
		},
		{
			Base: &lookout.File{Path: "f1old.py"},
		},
		{
			Base: &lookout.File{Path: "f2old.py"},
			Head: &lookout.File{Path: "f2new.js"},
		},
		// the change that used to be vendor but isn't anymore should survive
		{
			Base: &lookout.File{Path: "vendor/f2old.py"},
			Head: &lookout.File{Path: "f2new.js"},
		},
		// vendor files should be filtered out
		{
			Head: &lookout.File{Path: "vendor/f1new.go"},
		},
		{
			Base: &lookout.File{Path: "vendor/f1old.py"},
		},
		{
			Base: &lookout.File{Path: "node_modules/f2old.py"},
			Head: &lookout.File{Path: "node_modules/f2new.js"},
		},
		{
			Base: &lookout.File{Path: "f2old.py"},
			Head: &lookout.File{Path: "vendor/f2new.js"},
		},
	}
	scan := s.makeChangeScanner(inputChanges, req)
	var changes []*lookout.Change
	for scan.Next() {
		changes = append(changes, scan.Change())
	}

	require.NoError(scan.Err())
	require.Len(changes, 4)
	require.Equal(inputChanges[0], changes[0])
	require.Equal(inputChanges[1], changes[1])
	require.Equal(inputChanges[2], changes[2])
	require.Equal(inputChanges[3], changes[3])
}

func (s *ServiceSuite) TestChangesWantLanguage() {
	require := s.Require()

	req := &lookout.ChangesRequest{WantLanguage: true}
	inputChanges := []*lookout.Change{
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
		},
	}
	scan := s.makeChangeScanner(inputChanges, req)

	var changes []*lookout.Change
	for scan.Next() {
		changes = append(changes, scan.Change())
	}

	require.NoError(scan.Err())
	require.Equal(len(inputChanges), len(changes))

	require.Equal("Go", changes[0].Head.Language)
	require.Equal("Python", changes[1].Base.Language)
	require.Equal("JavaScript", changes[1].Head.Language)

	require.NoError(scan.Close())
}

func (s *ServiceSuite) TestChangesIncludeLanguages() {
	require := s.Require()

	req := &lookout.ChangesRequest{
		IncludeLanguages: []string{"JavaScript"},
		WantLanguage:     true,
	}
	inputChanges := []*lookout.Change{
		// should survive
		{
			Head: &lookout.File{Path: "f1.js"},
		},
		{
			Base: &lookout.File{Path: "f2.js"},
		},
		{
			Base: &lookout.File{Path: "f3.py"},
			Head: &lookout.File{Path: "f3.js"},
		},
		// filtered
		{
			Head: &lookout.File{Path: "f1.go"},
		},
		{
			Base: &lookout.File{Path: "f2.py"},
		},
		{
			Base: &lookout.File{Path: "f3.js"},
			Head: &lookout.File{Path: "f3.py"},
		},
	}
	scan := s.makeChangeScanner(inputChanges, req)
	var changes []*lookout.Change
	for scan.Next() {
		changes = append(changes, scan.Change())
	}

	require.NoError(scan.Err())
	require.Len(changes, 3)
	require.Equal(inputChanges[0].Head.Path, changes[0].Head.Path)
	require.Equal(inputChanges[1].Base.Path, changes[1].Base.Path)
	require.Equal(inputChanges[2].Head.Path, changes[2].Head.Path)
}

func (s *ServiceSuite) TestFilesExcludeVendored() {
	require := s.Require()

	req := &lookout.FilesRequest{ExcludeVendored: true}
	inputFiles := []*lookout.File{
		// non-vendor files should survive
		{Path: "f1new.go"},
		// vendor files should be filtered out
		{Path: "vendor/f1new.go"},
		{Path: "node_modules/f2old.py"},
	}
	scan := s.makeFileScanner(inputFiles, req)

	var files []*lookout.File
	for scan.Next() {
		files = append(files, scan.File())
	}

	require.Len(files, 1)
	require.Equal(inputFiles[0].Path, files[0].Path)
}

func (s *ServiceSuite) TestFilesWantLanguage() {
	require := s.Require()

	req := &lookout.FilesRequest{WantLanguage: true}
	inputFiles := []*lookout.File{
		{
			Path:    "f1new.go",
			Content: []byte("f1 new"),
		},
		{
			Path:    "f2new.js",
			Content: []byte("f2 new"),
		},
	}
	scan := s.makeFileScanner(inputFiles, req)

	var files []*lookout.File
	for scan.Next() {
		files = append(files, scan.File())
	}

	require.NoError(scan.Err())
	require.Equal(len(inputFiles), len(files))

	require.Equal("Go", files[0].Language)
	require.Equal("JavaScript", files[1].Language)

	require.NoError(scan.Close())
}

func (s *ServiceSuite) TestFilesIncludeLanguages() {
	cases := []struct {
		langs        []string
		WantLanguage bool
		expectFiles  int
	}{
		{[]string{"JavaScript"}, false, 1},
		// support lower case
		{[]string{"javascript"}, false, 1},
		// test multiple languages
		{[]string{"Go", "JavaScript"}, false, 2},
		// test we filter out only detected languages
		{[]string{"Unknown"}, true, 0},
		// test language set correctly
		{[]string{"Go"}, true, 1},
	}

	for i, c := range cases {
		s.T().Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			require := s.Require()

			req := &lookout.FilesRequest{
				IncludeLanguages: c.langs,
				WantLanguage:     c.WantLanguage,
			}
			inputFiles := []*lookout.File{
				{Path: "f1.go"},
				{Path: "f2.unknown"},
				{Path: "f3.js"},
			}
			scan := s.makeFileScanner(inputFiles, req)

			var files []*lookout.File
			for scan.Next() {
				files = append(files, scan.File())
			}

			require.NoError(scan.Err())
			require.Equal(c.expectFiles, len(files))
			for j := range files {
				if c.WantLanguage {
					require.Equal(files[j].Language, c.langs[j])
				} else {
					require.Equal(files[j].Language, "")
				}
			}
		})
	}
}

func (s *ServiceSuite) makeChangeScanner(inputChanges []*lookout.Change, req *lookout.ChangesRequest) lookout.ChangeScanner {
	require := s.Require()

	underlying := &mock.MockChangesService{T: s.T()}
	srv := NewService(underlying, nil)
	require.NotNil(srv)

	req.Base = &lookout.ReferencePointer{
		InternalRepositoryURL: "repo://myrepo",
		Hash:                  "foo",
	}
	req.Head = &lookout.ReferencePointer{
		InternalRepositoryURL: "repo://myrepo",
		Hash:                  "bar",
	}

	underlying.ExpectedRequest = req
	underlying.ChangeScanner = &mock.SliceChangeScanner{Changes: inputChanges}

	scan, err := srv.GetChanges(context.TODO(), req)
	require.NoError(err)
	require.NotNil(scan)

	return scan
}

func (s *ServiceSuite) makeFileScanner(inputFiles []*lookout.File, req *lookout.FilesRequest) lookout.FileScanner {
	require := s.Require()

	underlying := &mock.MockFilesService{T: s.T()}
	srv := NewService(nil, underlying)
	require.NotNil(srv)

	req.Revision = &lookout.ReferencePointer{
		InternalRepositoryURL: "repo://myrepo",
		Hash:                  "foo",
	}

	underlying.ExpectedRequest = req
	underlying.FileScanner = &mock.SliceFileScanner{Files: inputFiles}

	scan, err := srv.GetFiles(context.TODO(), req)
	require.NoError(err)
	require.NotNil(scan)

	return scan
}
