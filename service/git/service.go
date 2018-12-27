package git

import (
	"context"
	"fmt"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	errors "gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// Service implements data service interface on top of go-git
type Service struct {
	loader CommitLoader
}

var _ lookout.ChangeGetter = &Service{}
var _ lookout.FileGetter = &Service{}

// NewService creates new git Service
func NewService(loader CommitLoader) *Service {
	return &Service{
		loader: loader,
	}
}

var ErrRefValidation = errors.NewKind("reference %v does not have a %s")

// validateReferences checks if all the References have enough information to clone a repo.
// Validation of the reference name is optional.
func validateReferences(ctx context.Context, validateRefName bool, refs ...*lookout.ReferencePointer) error {
	ctxlog.Get(ctx).Debugf("validating refs: %v, validateRefName: %v", refs, validateRefName)
	for _, ref := range refs {
		if nil == ref {
			continue
		}
		if "" == ref.Hash {
			return ErrRefValidation.New(ref, "Hash")
		}

		if "" == ref.InternalRepositoryURL {
			return ErrRefValidation.New(ref, "InternalRepositoryURL")
		}

		if validateRefName && "" == ref.ReferenceName {
			return ErrRefValidation.New(ref, "ReferenceName")
		}
	}
	return nil
}

// GetChanges returns a ChangeScanner that scans all changes according to the request.
func (r *Service) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (
	lookout.ChangeScanner, error) {
	err := validateReferences(ctx, true, req.Base, req.Head)
	if err != nil {
		return nil, err
	}

	base, head, err := r.loadTrees(ctx, req.Base, req.Head)
	if err != nil {
		return nil, err
	}

	var scanner lookout.ChangeScanner

	if base == nil {
		scanner = NewTreeScanner(head)
	} else {
		scanner = NewDiffTreeScanner(base, head)
	}

	if req.IncludePattern != "" || req.ExcludePattern != "" {
		scanner = NewChangeFilterScanner(scanner,
			req.IncludePattern, req.ExcludePattern)
	}

	if req.WantContents {
		scanner = NewChangeBlobScanner(ctx, scanner, base, head)
	}

	return scanner, nil
}

// GetFiles returns a FilesScanner that scans all files according to the request.
func (r *Service) GetFiles(ctx context.Context, req *lookout.FilesRequest) (
	lookout.FileScanner, error) {
	err := validateReferences(ctx, false, req.Revision)
	if err != nil {
		return nil, err
	}

	_, tree, err := r.loadTrees(ctx, nil, req.Revision)
	if err != nil {
		return nil, err
	}

	var scanner lookout.FileScanner
	scanner = NewTreeScanner(tree)

	if req.IncludePattern != "" || req.ExcludePattern != "" {
		scanner = NewFileFilterScanner(ctx, scanner,
			req.IncludePattern, req.ExcludePattern)
	}

	if req.WantContents {
		scanner = NewFileBlobScanner(ctx, scanner, tree)
	}

	return scanner, nil
}

const maxResolveLength = 20

func (r *Service) loadTrees(ctx context.Context,
	base, head *lookout.ReferencePointer) (*object.Tree, *object.Tree, error) {

	var rps []lookout.ReferencePointer
	if base != nil {
		rps = append(rps, *base)
	}

	rps = append(rps, *head)

	ctxlog.Get(ctx).Debugf("load trees for references: %v", rps)

	commits, err := r.loader.LoadCommits(ctx, rps...)
	if err != nil {
		return nil, nil, err
	}

	trees := make([]*object.Tree, len(commits))
	for i, c := range commits {
		t, err := c.Tree()
		if err != nil {
			return nil, nil, err
		}

		trees[i] = t
	}

	if base == nil {
		return nil, trees[0], nil
	}

	return trees[0], trees[1], nil
}

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
