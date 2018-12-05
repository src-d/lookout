package enry

import (
	"context"

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

	scanner, err := s.changes.GetChanges(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.ExcludeVendored {
		scanner = newChangeExcludeVendorScanner(scanner)
	}

	if req.WantLanguage {
		scanner = newChangeLanguageScanner(scanner)
	}

	if len(req.OnlyLanguages) > 0 {
		scanner = newChangeFilterLanguageScanner(scanner, req.OnlyLanguages, !req.WantLanguage)
	}

	return scanner, nil
}

// GetFiles returns a FilesScanner that scans all files according to the request.
func (s *Service) GetFiles(ctx context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	if req.WantLanguage {
		req.WantContents = true
	}

	scanner, err := s.files.GetFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.ExcludeVendored {
		scanner = newFileExcludeVendorScanner(scanner)
	}

	if req.WantLanguage {
		scanner = newFileLanguageScanner(scanner)
	}

	if len(req.OnlyLanguages) > 0 {
		scanner = newFileFilterLanguageScanner(scanner, req.OnlyLanguages, !req.WantLanguage)
	}

	return scanner, nil
}
