package git

import (
	"io"
	"io/ioutil"
	"regexp"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/src-d/lookout/api"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	gitioutil "gopkg.in/src-d/go-git.v4/utils/ioutil"
)

type TreeScanner struct {
	storer storer.EncodedObjectStorer
	tree   *object.Tree
	tw     *object.TreeWalker
	val    *api.Change
	err    error
	done   bool
}

func NewTreeScanner(storer storer.EncodedObjectStorer,
	tree *object.Tree) *TreeScanner {

	return &TreeScanner{
		storer: storer,
		tree:   tree,
		tw:     object.NewTreeWalker(tree, true, nil),
	}
}

func (s *TreeScanner) Next() bool {
	if s.done {
		return false
	}

	for {
		name, entry, err := s.tw.Next()
		if err == io.EOF {
			s.done = true
			return false
		}

		if err != nil {
			s.done = true
			s.err = err
			return false
		}

		if !entry.Mode.IsFile() {
			continue
		}

		ch := &api.Change{New: &api.File{
			Path: name,
			Mode: uint32(entry.Mode),
			Sha1: entry.Hash.String(),
		}}

		s.val = ch
		return true
	}
}

func (s *TreeScanner) Err() error {
	return s.err
}

func (s *TreeScanner) Change() *api.Change {
	return s.val
}

func (s *TreeScanner) Close() error {
	if s.tw != nil {
		s.tw.Close()
	}

	return nil
}

type DiffTreeScanner struct {
	storer   storer.EncodedObjectStorer
	old, new *object.Tree
	val      *object.Change
	err      error
	started  bool
	changes  object.Changes
}

func NewDiffTreeScanner(storer storer.EncodedObjectStorer,
	old, new *object.Tree) *DiffTreeScanner {

	return &DiffTreeScanner{
		storer: storer,
		old:    old,
		new:    new,
	}
}

func (s *DiffTreeScanner) Next() bool {
	if !s.started {
		defer func() { s.started = true }()
		changes, err := object.DiffTree(s.old, s.new)
		if err != nil {
			s.err = err
			return false
		}

		s.changes = changes
	}

	for {
		if len(s.changes) == 0 {
			return false
		}

		s.val, s.changes = s.changes[0], s.changes[1:]
		return true
	}
}

func (s *DiffTreeScanner) Err() error {
	return s.err
}

func (s *DiffTreeScanner) Change() *api.Change {
	return &api.Change{
		Old: gitChangeEntryToApiFile(s.val.From),
		New: gitChangeEntryToApiFile(s.val.To),
	}
}

func (s *DiffTreeScanner) Close() error {
	return nil
}

func gitChangeEntryToApiFile(entry object.ChangeEntry) *api.File {
	if entry.Name == "" {
		return nil
	}

	return &api.File{
		Path: entry.Name,
		Mode: uint32(entry.TreeEntry.Mode),
		Sha1: entry.TreeEntry.Hash.String(),
	}
}

type FilterScanner struct {
	Scanner           api.ChangeScanner
	includePatternRaw string
	excludePatternRaw string
	includePattern    *regexp.Regexp
	excludePattern    *regexp.Regexp
	val               *api.Change
	started           bool
	done              bool
	err               error
}

func NewFilterScanner(
	scanner api.ChangeScanner,
	include, exclude string) *FilterScanner {
	return &FilterScanner{
		Scanner:           scanner,
		includePatternRaw: include,
		excludePatternRaw: exclude,
	}
}

func (s *FilterScanner) Next() bool {
	if s.done {
		return false
	}

	if !s.started {
		defer func() { s.started = true }()

		s.includePattern, s.err = s.compile(s.includePatternRaw)
		if s.err != nil {
			s.done = true
			return false
		}

		s.excludePattern, s.err = s.compile(s.excludePatternRaw)
		if s.err != nil {
			s.done = true
			return false
		}
	}

	for s.Scanner.Next() {
		ch := s.Scanner.Change()

		if !s.matchInclude(ch.New.Path) {
			continue
		}

		if s.matchExclude(ch.New.Path) {
			continue
		}

		s.val = ch
		return true
	}

	s.done = true
	return false
}

func (s *FilterScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *FilterScanner) Change() *api.Change {
	return s.val
}

func (s *FilterScanner) Close() error {
	return s.Scanner.Close()
}

func (s *FilterScanner) compile(pat string) (*regexp.Regexp, error) {
	if pat == "" {
		return nil, nil
	}

	return regexp.Compile(pat)
}

func (s *FilterScanner) matchInclude(p string) bool {
	if s.includePattern == nil {
		return true
	}

	return s.includePattern.MatchString(p)
}

func (s *FilterScanner) matchExclude(p string) bool {
	if s.excludePattern == nil {
		return false
	}

	return s.excludePattern.MatchString(p)
}

type BlobScanner struct {
	Scanner api.ChangeScanner
	Storer  storer.EncodedObjectStorer
	val     *api.Change
	done    bool
	err     error
}

func NewBlobScanner(
	scanner api.ChangeScanner,
	storer storer.EncodedObjectStorer) *BlobScanner {
	return &BlobScanner{
		Scanner: scanner,
		Storer:  storer,
	}
}

func (s *BlobScanner) Next() bool {
	if s.done {
		return false
	}

	for s.Scanner.Next() {
		ch := s.Scanner.Change()
		if err := s.addBlob(ch.GetOld()); err != nil {
			s.done = true
			s.err = err
			return false
		}

		if err := s.addBlob(ch.GetNew()); err != nil {
			s.done = true
			s.err = err
			return false
		}

		s.val = ch
		return true
	}

	s.done = true
	return false
}

func (s *BlobScanner) addBlob(f *api.File) (err error) {
	if f == nil {
		return nil
	}

	sha1 := f.GetSha1()
	if sha1 == "" {
		return nil
	}

	obj, err := s.Storer.EncodedObject(
		plumbing.BlobObject, plumbing.NewHash(sha1))
	if err != nil {
		return err
	}

	blob, err := object.DecodeBlob(obj)
	if err != nil {
		return err
	}

	r, err := blob.Reader()
	if err != nil {
		return err
	}

	defer gitioutil.CheckClose(r, &err)

	f.Content, err = ioutil.ReadAll(r)
	return err

}

func (s *BlobScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *BlobScanner) Change() *api.Change {
	return s.val
}

func (s *BlobScanner) Close() error {
	return s.Scanner.Close()
}
