package git

import (
	"context"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	errors "gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4"
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

func (r *Service) getFrom(
	ctx context.Context,
	base, head *lookout.ReferencePointer,
) (
	*lookout.ReferencePointer, error,
) {
	commits, err := r.loader.LoadCommits(ctx, *base, *head)
	if err != nil {
		return nil, err
	}

	var commonAncestor *object.Commit
	res, err := git.MergeBase(commits[0], commits[1])
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		// If there is no common ancestor between two commits it means that they
		// don't have a common history. That PR won't be able to be created in
		// GitHub, but in other situations (ie. with lookout-sdk), an analyzer
		// could be interested in analyzing the difference between both commints.
		return base, nil
	}

	commonAncestor = res[0]

	/*
		// TODO(dpordomingo): uncomment this after testing that it's not a problem
		// returning commits without ReferenceName (see comment below)
		if base.Hash == commonAncestor.Hash.String() {
			return base, nil
		}
	*/

	return &lookout.ReferencePointer{
		InternalRepositoryURL: base.InternalRepositoryURL,
		// ReferenceName can be undefined for a random commit inside that repository
		ReferenceName: "",
		Hash:          commonAncestor.Hash.String(),
	}, nil
}

// GetChanges returns a ChangeScanner that scans all changes according to the request.
func (r *Service) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (
	lookout.ChangeScanner, error) {
	err := validateReferences(ctx, true, req.Base, req.Head)
	if err != nil {
		return nil, err
	}

	var from *lookout.ReferencePointer

	// The standard behavior for getting the changes between two commits is like doing
	// `git diff base...head`, also `git diff $(git merge-base base head) head`
	// (as it appears in `Changes` tab in GitHub PRs)
	// If it is desired to get all changes between `base` and `head`,
	// (as done by `git diff base..head`) it must be sent `req.TwoDotsMode` as true
	if req.TwoDotsMode {
		from = req.Base
	} else {
		if req.Base != nil {
			from, err = r.getFrom(ctx, req.Base, req.Head)
			if err != nil {
				return nil, err
			}
		}
	}

	base, head, err := r.loadTrees(ctx, from, req.Head)
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
