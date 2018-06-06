package api

import (
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type GitChangeScanner struct {
	storer    storer.EncodedObjectStorer
	base, top *object.Tree
	tw        *object.TreeWalker
	val       *ChangesResponse
	err       error
	done      bool
}

func NewGitChangeScanner(storer storer.EncodedObjectStorer,
	base, top *object.Tree) *GitChangeScanner {

	return &GitChangeScanner{
		storer: storer,
		base:   base,
		top:    top,
		tw:     object.NewTreeWalker(top, true, nil),
	}
}
func NewErrorGitChangeScanner(err error) *GitChangeScanner {
	return &GitChangeScanner{
		done: true,
		err:  err,
	}
}

func (s *GitChangeScanner) Next() bool {
	if s.done {
		return false
	}

	name, _, err := s.tw.Next()
	if err == io.EOF {
		s.done = true
		return false
	}

	if err != nil {
		s.done = true
		s.err = err
		return false
	}

	s.val = &ChangesResponse{}
	s.val.Change = &Change{}
	s.val.Change.New = &File{}
	s.val.Change.New.Path = name

	return true
}

func (s *GitChangeScanner) Err() error {
	return s.err
}

func (s *GitChangeScanner) Change() *ChangesResponse {
	return s.val
}

func (s *GitChangeScanner) Close() error {
	s.tw.Close()
	return nil
}
