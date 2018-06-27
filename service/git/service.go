package git

import (
	"fmt"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
)

type Service struct {
	loader server.Loader
}

var _ lookout.ChangeGetter = &Service{}

func NewService(loader server.Loader) *Service {
	return &Service{
		loader: loader,
	}
}

func (r *Service) GetChanges(req *lookout.ChangesRequest) (
	lookout.ChangeScanner, error) {
	ep, err := transport.NewEndpoint(req.Head.Repository().CloneURL)
	if err != nil {
		return nil, err
	}

	s, err := r.loader.Load(ep)
	if err != nil {
		return nil, err
	}

	if req.Head == nil {
		return nil, fmt.Errorf("head reference is mandatory")
	}

	var base, top *object.Tree
	if req.Base != nil {
		base, err = r.resolveCommitTree(s, plumbing.NewHash(req.Base.Hash))
		if err != nil {
			return nil, fmt.Errorf("error retrieving base commit %s: %s",
				req.Base, err)
		}
	}

	top, err = r.resolveCommitTree(s, plumbing.NewHash(req.Head.Hash))
	if err != nil {
		return nil, err
	}

	var scanner lookout.ChangeScanner

	if base == nil {
		scanner = NewTreeScanner(s, top)
	} else {
		scanner = NewDiffTreeScanner(s, base, top)
	}

	if req.IncludePattern != "" || req.ExcludePattern != "" {
		scanner = NewFilterScanner(scanner,
			req.IncludePattern, req.ExcludePattern)
	}

	if req.WantContents {
		scanner = NewBlobScanner(scanner, s)
	}

	return scanner, nil
}

const maxResolveLength = 20

func (r *Service) resolveCommitTree(s storer.Storer, h plumbing.Hash) (
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

func (r *Service) resolveCommit(s storer.Storer, h plumbing.Hash) (
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
