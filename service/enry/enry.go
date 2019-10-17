package enry

import (
	"context"
	"strings"

	"github.com/src-d/lookout"
	enry "gopkg.in/src-d/enry.v1"
)

func filterVendor(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	return enry.IsVendor(f.Path), nil
}

func getLanguage(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	f.Language = enry.GetLanguage(f.Path, f.Content)

	return false, nil
}

func newLanguageFilter(langs []string, detectLang bool) func(f *lookout.File) (bool, error) {
	allow := make(map[string]bool, len(langs))
	for _, l := range langs {
		allow[strings.ToLower(l)] = true
	}

	return func(f *lookout.File) (bool, error) {
		if f == nil {
			return true, nil
		}

		lang := f.Language
		if detectLang {
			lang = enry.GetLanguage(f.Path, f.Content)
		}

		allowed, _ := allow[strings.ToLower(lang)]
		return !allowed, nil
	}
}

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
		scanner = lookout.NewFnChangeScanner(scanner, func(ch *lookout.Change) (bool, error) {
			if ch.Head == nil {
				return filterVendor(ch.Base)
			}

			return filterVendor(ch.Head)
		})
	}

	if req.WantLanguage {
		scanner = lookout.NewFnChangeScanner(scanner, func(ch *lookout.Change) (bool, error) {
			getLanguage(ch.Base)
			getLanguage(ch.Head)
			return false, nil
		})
	}

	if len(req.IncludeLanguages) > 0 {
		fn := newLanguageFilter(req.IncludeLanguages, !req.WantLanguage)
		scanner = lookout.NewFnChangeScanner(scanner, func(ch *lookout.Change) (bool, error) {
			if ch.Head == nil {
				return fn(ch.Base)
			}

			return fn(ch.Head)
		})
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
		scanner = lookout.NewFnFileScanner(scanner, filterVendor)
	}

	if req.WantLanguage {
		scanner = lookout.NewFnFileScanner(scanner, getLanguage)
	}

	if len(req.IncludeLanguages) > 0 {
		scanner = lookout.NewFnFileScanner(scanner, newLanguageFilter(req.IncludeLanguages, !req.WantLanguage))
	}

	return scanner, nil
}
