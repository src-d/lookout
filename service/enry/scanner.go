package enry

import (
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

// newChangeExcludeVendorScanner creates new FnChangeScanner
func newChangeExcludeVendorScanner(scanner lookout.ChangeScanner) *lookout.FnChangeScanner {
	fn := func(ch *lookout.Change) (bool, error) {
		if ch.Head == nil {
			return filterVendor(ch.Base)
		}

		return filterVendor(ch.Head)
	}
	return &lookout.FnChangeScanner{
		Scanner: scanner,
		Fn:      fn,
	}
}

// newFileExcludeVendorScanner creates new FnFileScanner
func newFileExcludeVendorScanner(scanner lookout.FileScanner) *lookout.FnFileScanner {
	return &lookout.FnFileScanner{
		Scanner: scanner,
		Fn:      filterVendor,
	}
}

func getLanguage(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	f.Language = enry.GetLanguage(f.Path, f.Content)

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

type languageFilter struct {
	allow  map[string]bool
	detect bool
}

func newLanguageFilter(langs []string, detectLang bool) *languageFilter {
	filter := languageFilter{
		allow:  make(map[string]bool, len(langs)),
		detect: detectLang,
	}
	for _, l := range langs {
		filter.allow[strings.ToLower(l)] = true
	}

	return &filter
}

func (s *languageFilter) Fn(f *lookout.File) (bool, error) {
	if f == nil {
		return true, nil
	}

	lang := f.Language
	if s.detect {
		lang = enry.GetLanguage(f.Path, f.Content)
	}

	allowed, _ := s.allow[strings.ToLower(lang)]
	return !allowed, nil
}

func newChangeFilterLanguageScanner(scanner lookout.ChangeScanner, langs []string, detectLang bool) *lookout.FnChangeScanner {
	filter := newLanguageFilter(langs, detectLang)
	fn := func(ch *lookout.Change) (bool, error) {
		if ch.Head == nil {
			return filter.Fn(ch.Base)
		}

		return filter.Fn(ch.Head)
	}

	return &lookout.FnChangeScanner{
		Scanner: scanner,
		Fn:      fn,
	}
}

func newFileFilterLanguageScanner(scanner lookout.FileScanner, langs []string, detectLang bool) *lookout.FnFileScanner {
	filter := newLanguageFilter(langs, detectLang)

	return &lookout.FnFileScanner{
		Scanner: scanner,
		Fn:      filter.Fn,
	}
}
