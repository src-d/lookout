package api

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
)

type GitDataReader struct {
	loader server.Loader
}

var _ DataReader = &GitDataReader{}

func NewGitDataReader(loader server.Loader) *GitDataReader {
	return &GitDataReader{
		loader: loader,
	}
}

func (r *GitDataReader) GetChanges(req *ChangesRequest) ChangeScanner {
	ep, err := transport.NewEndpoint(req.GetRepository())
	if err != nil {
		return NewErrorGitChangeScanner(err)
	}

	s, err := r.loader.Load(ep)
	if err != nil {
		return NewErrorGitChangeScanner(err)
	}

	if req.GetTop() == "" {
		return NewErrorGitChangeScanner(fmt.Errorf("top commit is mandatory"))
	}

	var base, top *object.Tree
	if req.GetBase() == "" {
		base, err = r.resolveCommitTree(s, plumbing.NewHash(req.GetBase()))
		if err != nil {
			return NewErrorGitChangeScanner(err)
		}
	}

	top, err = r.resolveCommitTree(s, plumbing.NewHash(req.GetTop()))
	if err != nil {
		return NewErrorGitChangeScanner(err)
	}

	return NewGitChangeScanner(s, base, top)
}

const maxResolveLength = 20

func (r *GitDataReader) resolveCommitTree(s storer.Storer, h plumbing.Hash) (
	*object.Tree, error) {

	c, err := r.resolveCommit(s, h)
	if err != nil {
		return nil, err
	}

	t, err := c.Tree()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (r *GitDataReader) resolveCommit(s storer.Storer, h plumbing.Hash) (
	*object.Commit, error) {

	for i := 0; i < maxResolveLength; i++ {
		obj, err := s.EncodedObject(plumbing.AnyObject, h)
		if err != nil {
			return nil, err
		}

		switch obj.Type() {
		case plumbing.TagObject:
			tag, err := object.DecodeTag(s, obj)
			if err != nil {
				return nil, err
			}

			h = tag.Target
		case plumbing.CommitObject:
			commit, err := object.DecodeCommit(s, obj)
			if err != nil {
				return nil, err
			}

			return commit, nil
		default:
			return nil, fmt.Errorf("bad object type: %s", obj.Type().String())
		}
	}

	return nil, fmt.Errorf("maximum length of tag chain exceeded: %d", maxResolveLength)
}
