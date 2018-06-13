package git

import (
	"io"

	"github.com/src-d/lookout/api"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
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

		s.val = &api.Change{}
		s.val.New = &api.File{}
		s.val.New.Path = name

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
