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

func (s *ScannerSuite) TestFileFilterLanguageScanner() {
	s.testFileFilterLangScanner([]string{"JavaScript"}, true, 1)
	// support lower case
	s.testFileFilterLangScanner([]string{"javascript"}, true, 1)
	// test we filter out only detected languages
	s.testFileFilterLangScanner([]string{"Go"}, false, 1)
	s.testFileFilterLangScanner([]string{"JavaScript"}, false, 0)
	// test multiple languages
	s.testFileFilterLangScanner([]string{"JavaScript", "Go"}, true, 2)
}

func (s *ScannerSuite) testFileFilterLangScanner(langs []string, detectLang bool, expectFiles int) {
	require := s.Require()

	underlyingFiles := []*lookout.File{
		{Path: "f1.go", Language: "Go"},
		{Path: "f2.py"},
		{Path: "f3.js"},
	}

	cs := newFileFilterLanguageScanner(
		&mock.SliceFileScanner{Files: underlyingFiles},
		langs, detectLang,
	)

	var files []*lookout.File
	for cs.Next() {
		files = append(files, cs.File())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(files, expectFiles)
}

func (s *ScannerSuite) TestChangeFilterLanguageScanner() {
	require := s.Require()

	underlyingChanges := []*lookout.Change{
		// should survive
		{
			Head: &lookout.File{Path: "f1.go"},
		},
		{
			Base: &lookout.File{Path: "f2.py"},
		},
		{
			Base: &lookout.File{Path: "f3.py"},
			Head: &lookout.File{Path: "f3.js"},
		},
		// filtered
		{
			Head: &lookout.File{Path: "f1.js"},
		},
		{
			Base: &lookout.File{Path: "f2.js"},
		},
		{
			Base: &lookout.File{Path: "f3.js"},
			Head: &lookout.File{Path: "f3.py"},
		},
	}

	cs := newChangeFilterLanguageScanner(
		&mock.SliceChangeScanner{Changes: underlyingChanges},
		[]string{"JavaScript"},
		true,
	)

	var changes []*lookout.Change
	for cs.Next() {
		changes = append(changes, cs.Change())
	}

	require.False(cs.Next())
	require.NoError(cs.Err())
	require.NoError(cs.Close())

	require.Len(changes, 3)
}
