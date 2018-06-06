package server

import (
	"io"

	"github.com/src-d/lookout/api"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type GitChangeScanner struct {
	storer    storer.EncodedObjectStorer
	base, top *object.Tree
	tw        *object.TreeWalker
	val       *api.ChangesResponse
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

func (s *GitChangeScanner) Next() bool {
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

		s.val = &api.ChangesResponse{}
		s.val.Change = &api.Change{}
		s.val.Change.New = &api.File{}
		s.val.Change.New.Path = name

		return true
	}
}

func (s *GitChangeScanner) Err() error {
	return s.err
}

func (s *GitChangeScanner) Change() *api.ChangesResponse {
	return s.val
}

func (s *GitChangeScanner) Close() error {
	if s.tw != nil {
		s.tw.Close()
	}

	return nil
}
