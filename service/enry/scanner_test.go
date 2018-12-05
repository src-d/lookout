package enry

import (
	"testing"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/mock"
	"github.com/stretchr/testify/suite"
)

type ScannerSuite struct {
	suite.Suite
}

func TestScannerSuiteSuite(t *testing.T) {
	suite.Run(t, new(ScannerSuite))
}

func (s *ScannerSuite) TestFileChangeVendorScanner() {
	require := s.Require()

	underlyingChanges := []*lookout.Change{
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

	cs := newChangeExcludeVendorScanner(&mock.SliceChangeScanner{Changes: underlyingChanges})

	var changes []*lookout.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 4)
}

func (s *ScannerSuite) TestFileExcludeVendorScanner() {
	require := s.Require()

	underlyingFiles := []*lookout.File{
		// non-vendor files should survive
		{Path: "f1new.go"},
		// vendor files should be filtered out
		{Path: "vendor/f1new.go"},
		{Path: "node_modules/f2old.py"},
	}

	cs := newFileExcludeVendorScanner(&mock.SliceFileScanner{Files: underlyingFiles})

	var files []*lookout.File
	for cs.Next() {
		files = append(files, cs.File())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(files, 1)
}
