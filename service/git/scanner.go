package git

import (
	"io"
	"io/ioutil"
	"regexp"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	gitioutil "gopkg.in/src-d/go-git.v4/utils/ioutil"
)

type TreeScanner struct {
	storer storer.EncodedObjectStorer
	tree   *object.Tree
	tw     *object.TreeWalker
	val    *lookout.Change
	err    error
	done   bool
}

func NewTreeScanner(tree *object.Tree) *TreeScanner {

	return &TreeScanner{
		tree: tree,
		tw:   object.NewTreeWalker(tree, true, nil),
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

		ch := &lookout.Change{Head: &lookout.File{
			Path: name,
			Mode: uint32(entry.Mode),
			Hash: entry.Hash.String(),
		}}

		s.val = ch
		return true
	}
}

func (s *TreeScanner) Err() error {
	return s.err
}

func (s *TreeScanner) Change() *lookout.Change {
	return s.val
}

func (s *TreeScanner) Close() error {
	if s.tw != nil {
		s.tw.Close()
	}

	return nil
}

type DiffTreeScanner struct {
	base, head *object.Tree
	val        *object.Change
	err        error
	started    bool
	changes    object.Changes
}

func NewDiffTreeScanner(base, head *object.Tree) *DiffTreeScanner {

	return &DiffTreeScanner{
		base: base,
		head: head,
	}
}

func (s *DiffTreeScanner) Next() bool {
	if !s.started {
		defer func() { s.started = true }()
		changes, err := object.DiffTree(s.base, s.head)
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

func (s *DiffTreeScanner) Change() *lookout.Change {
	return &lookout.Change{
		Base: gitChangeEntryToApiFile(s.val.From),
		Head: gitChangeEntryToApiFile(s.val.To),
	}
}

func (s *DiffTreeScanner) Close() error {
	return nil
}

func gitChangeEntryToApiFile(entry object.ChangeEntry) *lookout.File {
	if entry.Name == "" {
		return nil
	}

	return &lookout.File{
		Path: entry.Name,
		Mode: uint32(entry.TreeEntry.Mode),
		Hash: entry.TreeEntry.Hash.String(),
	}
}

type FilterScanner struct {
	Scanner           lookout.ChangeScanner
	includePatternRaw string
	excludePatternRaw string
	includePattern    *regexp.Regexp
	excludePattern    *regexp.Regexp
	val               *lookout.Change
	started           bool
	done              bool
	err               error
}

func NewFilterScanner(
	scanner lookout.ChangeScanner,
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

		if !s.matchInclude(ch.Head.Path) {
			continue
		}

		if s.matchExclude(ch.Head.Path) {
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

func (s *FilterScanner) Change() *lookout.Change {
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
	Scanner    lookout.ChangeScanner
	Base, Head *object.Tree
	val        *lookout.Change
	done       bool
	err        error
}

func NewBlobScanner(
	scanner lookout.ChangeScanner,
	base, head *object.Tree) *BlobScanner {
	return &BlobScanner{
		Scanner: scanner,
		Base:    base,
		Head:    head,
	}
}

func (s *BlobScanner) Next() bool {
	if s.done {
		return false
	}

	for s.Scanner.Next() {
		ch := s.Scanner.Change()
		if err := s.addBlob(s.Base, ch.Base); err != nil {
			s.done = true
			s.err = err
			return false
		}

		if err := s.addBlob(s.Head, ch.Head); err != nil {
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

func (s *BlobScanner) addBlob(t *object.Tree, f *lookout.File) (err error) {
	if f == nil {
		return nil
	}

	if f.Hash == "" {
		return nil
	}

	of, err := t.File(f.Path)
	if err != nil {
		return err
	}

	r, err := of.Blob.Reader()
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

func (s *BlobScanner) Change() *lookout.Change {
	return s.val
}

func (s *BlobScanner) Close() error {
	return s.Scanner.Close()
}
