package enry

import (
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
