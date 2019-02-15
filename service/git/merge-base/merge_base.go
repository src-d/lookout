package merge_base

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// errIsReachable is thrown when first commit is an ancestor of the second
var errIsReachable = fmt.Errorf("first is reachable from second")

// MergeBase mimics the behavior of `git merge-base first second`, returning one
// of the best common ancestor of the first and the second passed commits
// The best common ancestor can not be reached from other common ancestors
func MergeBase(
	// REVIEWER: store param wouldn't be needed if MergeBase were part of go-git/git.Repository
	store storer.EncodedObjectStorer,
	first *object.Commit,
	second *object.Commit,
) ([]*object.Commit, error) {

	secondHistory, err := ancestorsIndex(first, second)
	if err == errIsReachable {
		return []*object.Commit{first}, nil
	}

	if err != nil {
		return nil, err
	}

	var res []*object.Commit
	inSecondHistory := isInIndexCommitFilter(secondHistory)
	// REVIEWER: store argument wouldn't be needed if this were part of go-git/plumbing/object package
	resIter := NewFilterCommitIter(store, first, &inSecondHistory, &inSecondHistory)
	err = resIter.ForEach(func(commit *object.Commit) error {
		res = append(res, commit)
		return nil
	})

	return independents(res, 0)
}

// ancestorsIndex returns a map with the ancestors of the first commit if the
// second one is not one of them. It returns errIsReachable if the second one is
// ancestor, or another error if the history is not transversable
func ancestorsIndex(first, second *object.Commit) (map[plumbing.Hash]bool, error) {
	if first.Hash.String() == second.Hash.String() {
		return nil, errIsReachable
	}

	secondHistory := map[plumbing.Hash]bool{}
	secondIter := object.NewCommitIterBSF(second, nil, nil)
	err := secondIter.ForEach(func(commit *object.Commit) error {
		if commit.Hash == first.Hash {
			return errIsReachable
		}

		secondHistory[commit.Hash] = true
		return nil
	})

	if err == errIsReachable {
		return nil, errIsReachable
	}

	if err != nil {
		return nil, err
	}

	return secondHistory, nil
}

// independents returns a subset of the passed commits, that are not reachable
// from any other. Since A can be not reachable from B, but B can be reached
// from A, it must be checked both directions, traversing the history of all
// passed commits, and removing those that are reachable from any traversed history.
// Every time a history is traversed, and the ancestor found have been removed
// from the subset of independents commits, the function is called recursively
// with the new subset.
func independents(commits []*object.Commit, start int) ([]*object.Commit, error) {
	if len(commits) == 1 {
		return commits, nil
	}

	res := commits
	for i := start; i < len(commits); i++ {
		from := commits[i]
		fromHistoryIter := object.NewCommitIterBSF(from, nil, nil)
		err := fromHistoryIter.ForEach(func(fromAncestor *object.Commit) error {
			for _, other := range commits {
				if from.Hash != other.Hash && fromAncestor.Hash == other.Hash {
					res = remove(res, other)
				}
			}

			if len(res) == 1 {
				return storer.ErrStop
			}

			return nil
		})

		if err != nil {
			return nil, err
		}

		if len(res) < len(commits) {
			return independents(res, start)
		}

	}

	return commits, nil
}

func remove(commits []*object.Commit, toDelete *object.Commit) []*object.Commit {
	var res []*object.Commit
	for _, commit := range commits {
		if toDelete.Hash != commit.Hash {
			res = append(res, commit)
		}
	}

	return res
}

// isInIndexCommitFilter returns a commitFilter that returns true
// if the commit is in the passed index.
func isInIndexCommitFilter(index map[plumbing.Hash]bool) CommitFilter {
	return func(c *object.Commit) bool {
		return index[c.Hash]
	}
}
