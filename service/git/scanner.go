package git

import (
	"io"
	"io/ioutil"
	"regexp"

	"github.com/src-d/lookout"

	enry "gopkg.in/src-d/enry.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	gitioutil "gopkg.in/src-d/go-git.v4/utils/ioutil"
)

// TreeScanner is a scanner for files of git tree
type TreeScanner struct {
	storer storer.EncodedObjectStorer
	tree   *object.Tree
	tw     *object.TreeWalker
	val    *lookout.File
	err    error
	done   bool
}

// NewTreeScanner creates new TreeScanner
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

		s.val = &lookout.File{
			Path: name,
			Mode: uint32(entry.Mode),
			Hash: entry.Hash.String(),
		}
		return true
	}
}

func (s *TreeScanner) Err() error {
	return s.err
}

func (s *TreeScanner) File() *lookout.File {
	return s.val
}

func (s *TreeScanner) Change() *lookout.Change {
	return &lookout.Change{Head: s.val}
}

func (s *TreeScanner) Close() error {
	if s.tw != nil {
		s.tw.Close()
	}

	return nil
}

// DiffTreeScanner is a scanner for files of diff between git trees
type DiffTreeScanner struct {
	base, head *object.Tree
	val        *object.Change
	err        error
	started    bool
	changes    object.Changes
}

// NewDiffTreeScanner creates new DiffTreeScanner
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

type BaseFilterScanner struct {
	includePatternRaw string
	excludePatternRaw string
	includePattern    *regexp.Regexp
	excludePattern    *regexp.Regexp
	started           bool
	done              bool
	err               error
}

func (s *BaseFilterScanner) Next(fn func() bool) bool {
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

	hasNext := fn()
	if !hasNext {
		s.done = true
	}

	return hasNext
}

func (s *BaseFilterScanner) compile(pat string) (*regexp.Regexp, error) {
	if pat == "" {
		return nil, nil
	}

	return regexp.Compile(pat)
}

func (s *BaseFilterScanner) matchInclude(p string) bool {
	if s.includePattern == nil {
		return true
	}

	return s.includePattern.MatchString(p)
}

func (s *BaseFilterScanner) matchExclude(p string) bool {
	if s.excludePattern == nil {
		return false
	}

	return s.excludePattern.MatchString(p)
}

// ChangeFilterScanner filters results of ChangeScanner based on regexp file name patterns
type ChangeFilterScanner struct {
	BaseFilterScanner
	Scanner lookout.ChangeScanner
	val     *lookout.Change
}

// NewChangeFilterScanner creates new ChangeFilterScanner
func NewChangeFilterScanner(
	scanner lookout.ChangeScanner,
	include, exclude string) *ChangeFilterScanner {
	return &ChangeFilterScanner{
		BaseFilterScanner: BaseFilterScanner{
			includePatternRaw: include,
			excludePatternRaw: exclude,
		},
		Scanner: scanner,
	}
}

func (s *ChangeFilterScanner) Next() bool {
	return s.BaseFilterScanner.Next(s.next)
}

func (s *ChangeFilterScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *ChangeFilterScanner) Change() *lookout.Change {
	return s.val
}

func (s *ChangeFilterScanner) Close() error {
	return s.Scanner.Close()
}

func (s *ChangeFilterScanner) next() bool {
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
	return false
}

// FileFilterScanner filters results of FileScanner based on regexp file name patterns
type FileFilterScanner struct {
	BaseFilterScanner
	Scanner lookout.FileScanner
	val     *lookout.File
}

// NewFileFilterScanner creates new FileFilterScanner
func NewFileFilterScanner(
	scanner lookout.FileScanner,
	include, exclude string) *FileFilterScanner {
	return &FileFilterScanner{
		BaseFilterScanner: BaseFilterScanner{
			includePatternRaw: include,
			excludePatternRaw: exclude,
		},
		Scanner: scanner,
	}
}

func (s *FileFilterScanner) Next() bool {
	return s.BaseFilterScanner.Next(s.next)
}

func (s *FileFilterScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *FileFilterScanner) File() *lookout.File {
	return s.val
}

func (s *FileFilterScanner) Close() error {
	return s.Scanner.Close()
}

func (s *FileFilterScanner) next() bool {
	for s.Scanner.Next() {
		f := s.Scanner.File()

		if !s.matchInclude(f.Path) {
			continue
		}

		if s.matchExclude(f.Path) {
			continue
		}

		s.val = f
		return true
	}
	return false
}

type BaseBlobScanner struct {
	done bool
	err  error
}

func (s *BaseBlobScanner) addBlob(t *object.Tree, f *lookout.File) (err error) {
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

// ChangeBlobScanner adds blobs to results of ChangeScanner
type ChangeBlobScanner struct {
	BaseBlobScanner
	Scanner    lookout.ChangeScanner
	Base, Head *object.Tree
	val        *lookout.Change
}

// NewChangeBlobScanner creates new ChangeBlobScanner
func NewChangeBlobScanner(
	scanner lookout.ChangeScanner,
	base, head *object.Tree) *ChangeBlobScanner {
	return &ChangeBlobScanner{
		Scanner: scanner,
		Base:    base,
		Head:    head,
	}
}

func (s *ChangeBlobScanner) Next() bool {
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

func (s *ChangeBlobScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *ChangeBlobScanner) Change() *lookout.Change {
	return s.val
}

func (s *ChangeBlobScanner) Close() error {
	return s.Scanner.Close()
}

// FileBlobScanner adds blobs to results of FileScanner
type FileBlobScanner struct {
	BaseBlobScanner
	Scanner lookout.FileScanner
	Tree    *object.Tree
	val     *lookout.File
}

// NewFileBlobScanner creates new FileBlobScanner
func NewFileBlobScanner(scanner lookout.FileScanner, tree *object.Tree) *FileBlobScanner {
	return &FileBlobScanner{
		Scanner: scanner,
		Tree:    tree,
	}
}

func (s *FileBlobScanner) Next() bool {
	if s.done {
		return false
	}

	for s.Scanner.Next() {
		f := s.Scanner.File()
		if err := s.addBlob(s.Tree, f); err != nil {
			s.done = true
			s.err = err
			return false
		}

		s.val = f
		return true
	}

	s.done = true
	return false
}

func (s *FileBlobScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *FileBlobScanner) File() *lookout.File {
	return s.val
}

func (s *FileBlobScanner) Close() error {
	return s.Scanner.Close()
}

// ChangeExcludeVendorScanner filters vendored results of ChangeScanner
type ChangeExcludeVendorScanner struct {
	Scanner lookout.ChangeScanner
	val     *lookout.Change
	done    bool
	err     error
}

// NewChangeExcludeVendorScanner creates new ChangeExcludeVendorScanner
func NewChangeExcludeVendorScanner(scanner lookout.ChangeScanner) *ChangeExcludeVendorScanner {
	return &ChangeExcludeVendorScanner{
		Scanner: scanner,
	}
}

func (s *ChangeExcludeVendorScanner) Next() bool {
	if s.done {
		return false
	}

	for s.Scanner.Next() {
		ch := s.Scanner.Change()
		if enry.IsVendor(ch.Head.Path) {
			continue
		}

		s.val = ch
		return true
	}

	s.done = true
	return false
}

func (s *ChangeExcludeVendorScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}
func (s *ChangeExcludeVendorScanner) Change() *lookout.Change {
	return s.val
}

func (s *ChangeExcludeVendorScanner) Close() error {
	return s.Scanner.Close()
}

// FileExcludeVendorScanner filters vendored results of FileScanner
type FileExcludeVendorScanner struct {
	Scanner lookout.FileScanner
	val     *lookout.File
	done    bool
	err     error
}

// NewFileExcludeVendorScanner creates new FileExcludeVendorScanner
func NewFileExcludeVendorScanner(scanner lookout.FileScanner) *FileExcludeVendorScanner {
	return &FileExcludeVendorScanner{
		Scanner: scanner,
	}
}

func (s *FileExcludeVendorScanner) Next() bool {
	if s.done {
		return false
	}

	for s.Scanner.Next() {
		f := s.Scanner.File()
		if enry.IsVendor(f.Path) {
			continue
		}

		s.val = f
		return true
	}

	s.done = true
	return false
}

func (s *FileExcludeVendorScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}
func (s *FileExcludeVendorScanner) File() *lookout.File {
	return s.val
}

func (s *FileExcludeVendorScanner) Close() error {
	return s.Scanner.Close()
}
