package git

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	gitioutil "gopkg.in/src-d/go-git.v4/utils/ioutil"
	log "gopkg.in/src-d/go-log.v1"
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

type regexpFilter struct {
	includePatternRaw string
	excludePatternRaw string
	includePattern    *regexp.Regexp
	excludePattern    *regexp.Regexp
}

func (s *regexpFilter) OnStart() error {
	var err error
	s.includePattern, err = s.compile(s.includePatternRaw)
	if err != nil {
		return err
	}

	s.excludePattern, err = s.compile(s.excludePatternRaw)
	return err
}

func (s *regexpFilter) Fn(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	if !s.matchInclude(f.Path) {
		return true, nil
	}

	if s.matchExclude(f.Path) {
		return true, nil
	}

	return false, nil
}

func (s *regexpFilter) compile(pat string) (*regexp.Regexp, error) {
	if pat == "" {
		return nil, nil
	}

	return regexp.Compile(pat)
}

func (s *regexpFilter) matchInclude(p string) bool {
	if s.includePattern == nil {
		return true
	}

	return s.includePattern.MatchString(p)
}

func (s *regexpFilter) matchExclude(p string) bool {
	if s.excludePattern == nil {
		return false
	}

	return s.excludePattern.MatchString(p)
}

// NewChangeFilterScanner creates new FnChangeScanner
func NewChangeFilterScanner(scanner lookout.ChangeScanner, include, exclude string) *lookout.FnChangeScanner {
	filter := regexpFilter{
		includePatternRaw: include,
		excludePatternRaw: exclude,
	}

	fn := func(ch *lookout.Change) (bool, error) {
		if ch.Head == nil {
			return filter.Fn(ch.Base)
		}

		return filter.Fn(ch.Head)
	}

	return &lookout.FnChangeScanner{
		Scanner: scanner,
		OnStart: filter.OnStart,
		Fn:      fn,
	}
}

// NewFileFilterScanner creates new FnFileScanner
func NewFileFilterScanner(ctx context.Context, scanner lookout.FileScanner, include, exclude string) *lookout.FnFileScanner {
	filter := regexpFilter{
		includePatternRaw: include,
		excludePatternRaw: exclude,
	}
	return &lookout.FnFileScanner{
		Scanner: scanner,
		OnStart: filter.OnStart,
		Fn:      filter.Fn,
	}
}

type blobAdder struct {
	tree *object.Tree
	ctx  context.Context
}

func (b *blobAdder) Fn(f *lookout.File) (bool, error) {
	if f == nil {
		return false, nil
	}

	if f.Hash == "" {
		return false, nil
	}

	of, err := b.tree.File(f.Path)
	if err != nil {
		ctxlog.Get(b.ctx).With(log.Fields{
			"path": f.Path,
			"err":  err,
		}).Warningf("skipping - cannot get file")
		return true, nil
	}

	r, err := of.Blob.Reader()
	if err != nil {
		return true, fmt.Errorf("cannot get reader for file:'%v', %v", f.Path, err)
	}

	defer gitioutil.CheckClose(r, &err)

	f.Content, err = ioutil.ReadAll(r)
	if err != nil {
		return true, fmt.Errorf("cannot read file:'%v', %v", f.Path, err)
	}

	return false, nil
}

// NewFileBlobScanner creates new FnFileScanner
func NewFileBlobScanner(ctx context.Context, scanner lookout.FileScanner, tree *object.Tree) *lookout.FnFileScanner {
	adder := blobAdder{tree, ctx}
	return &lookout.FnFileScanner{
		Scanner: scanner,
		Fn:      adder.Fn,
	}
}

// NewChangeBlobScanner creates new FnChangeScanner
func NewChangeBlobScanner(ctx context.Context, scanner lookout.ChangeScanner, base, head *object.Tree) *lookout.FnChangeScanner {
	baseAdder := blobAdder{base, ctx}
	headAdder := blobAdder{head, ctx}

	fn := func(ch *lookout.Change) (bool, error) {
		skip, err := baseAdder.Fn(ch.Base)
		if err != nil || skip == true {
			return skip, err
		}

		return headAdder.Fn(ch.Head)
	}

	return &lookout.FnChangeScanner{
		Scanner: scanner,
		Fn:      fn,
	}
}
