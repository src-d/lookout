package enry

import (
	"context"

	goenry "gopkg.in/src-d/enry.v1"

	"github.com/src-d/lookout"
)

// Service implements data service interface which adds language information to the responses
type Service struct {
	changes lookout.ChangeGetter
	files   lookout.FileGetter
}

var _ lookout.ChangeGetter = &Service{}
var _ lookout.FileGetter = &Service{}

// NewService creates new enry Service
func NewService(changes lookout.ChangeGetter, files lookout.FileGetter) *Service {
	return &Service{
		changes: changes,
		files:   files,
	}
}

// GetChanges returns a ChangeScanner that scans all changes according to the request.
func (s *Service) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (lookout.ChangeScanner, error) {
	if req.WantLanguage {
		req.WantContents = true
	}

	changes, err := s.changes.GetChanges(ctx, req)
	if err != nil {
		return nil, err
	}

	if !req.WantLanguage {
		return changes, nil
	}

	return newChangeLanguageScanner(changes), nil
}

// GetFiles returns a FilesScanner that scans all files according to the request.
func (s *Service) GetFiles(ctx context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	if req.WantLanguage {
		req.WantContents = true
	}

	files, err := s.files.GetFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	if !req.WantLanguage {
		return files, nil
	}

	return newFileLanguageScanner(files), nil
}

func getLanguage(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	f.Language = goenry.GetLanguage(f.Path, f.Content)

	return false, nil
}

func newChangeLanguageScanner(scanner lookout.ChangeScanner) *lookout.FnChangeScanner {
	fn := func(ch *lookout.Change) (bool, error) {
		getLanguage(ch.Base)
		getLanguage(ch.Head)
		return false, nil
	}
	return &lookout.FnChangeScanner{
		Scanner: scanner,
		Fn:      fn,
	}
}

func newFileLanguageScanner(scanner lookout.FileScanner) *lookout.FnFileScanner {
	return &lookout.FnFileScanner{
		Scanner: scanner,
		Fn:      getLanguage,
	}
}
