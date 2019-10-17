package git

import (
	"fmt"
	"sort"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// errIsReachable is thrown when first commit is an ancestor of the second
var errIsReachable = fmt.Errorf("first is reachable from second")

// MergeBase mimics the behavior of `git merge-base first second`, returning the
// best common ancestor of the two passed commits
// The best common ancestors can not be reached from other common ancestors
func MergeBase(
	first *object.Commit,
	second *object.Commit,
) ([]*object.Commit, error) {

	// use sortedByCommitDateDesc strategy
	sorted := sortByCommitDateDesc(first, second)
	newer := sorted[0]
	older := sorted[1]

	newerHistory, err := ancestorsIndex(older, newer)
	if err == errIsReachable {
		return []*object.Commit{older}, nil
	}

	if err != nil {
		return nil, err
	}

	var res []*object.Commit
	inNewerHistory := isInIndexCommitFilter(newerHistory)
	resIter := object.NewFilterCommitIter(older, &inNewerHistory, &inNewerHistory)
	err = resIter.ForEach(func(commit *object.Commit) error {
		res = append(res, commit)
		return nil
	})

	return Independents(res)
}

// IsAncestor returns true if the candidate commit is ancestor of the target one
// It returns an error if the history is not transversable
// It mimics the behavior of `git merge --is-ancestor candidate target`
func IsAncestor(
	candidate *object.Commit,
	target *object.Commit,
) (bool, error) {
	_, err := ancestorsIndex(candidate, target)
	if err == errIsReachable {
		return true, nil
	}

	return false, nil
}

// ancestorsIndex returns a map with the ancestors of the starting commit if the
// excluded one is not one of them. It returns errIsReachable if the excluded commit
// is ancestor of the starting, or another error if the history is not transversable.
func ancestorsIndex(excluded, starting *object.Commit) (map[plumbing.Hash]struct{}, error) {
	if excluded.Hash.String() == starting.Hash.String() {
		return nil, errIsReachable
	}

	startingHistory := map[plumbing.Hash]struct{}{}
	startingIter := object.NewCommitIterBSF(starting, nil, nil)
	err := startingIter.ForEach(func(commit *object.Commit) error {
		if commit.Hash == excluded.Hash {
			return errIsReachable
		}

		startingHistory[commit.Hash] = struct{}{}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return startingHistory, nil
}

// Independents returns a subset of the passed commits, that are not reachable the others
// It mimics the behavior of `git merge-base --independent commit...`.
func Independents(commits []*object.Commit) ([]*object.Commit, error) {
	// use sortedByCommitDateDesc strategy
	cleaned := sortByCommitDateDesc(commits...)
	cleaned = removeDuplicated(cleaned)
	return independents(cleaned, map[plumbing.Hash]bool{}, 0)
}

func independents(
	candidates []*object.Commit,
	excluded map[plumbing.Hash]bool,
	start int,
) ([]*object.Commit, error) {
	if len(candidates) == 1 {
		return candidates, nil
	}

	res := candidates
	for i := start; i < len(candidates); i++ {
		from := candidates[i]
		others := remove(res, from)
		fromHistoryIter := object.NewCommitIterBSF(from, excluded, nil)
		err := fromHistoryIter.ForEach(func(fromAncestor *object.Commit) error {
			for _, other := range others {
				if fromAncestor.Hash == other.Hash {
					res = remove(res, other)
					others = remove(others, other)
				}
			}

			if len(res) == 1 {
				return storer.ErrStop
			}

			excluded[fromAncestor.Hash] = true
			return nil
		})

		if err != nil {
			return nil, err
		}

		if len(res) < len(candidates) {
			return independents(res, excluded, indexOf(res, from)+1)
		}

	}

	return res, nil
}

// sortByCommitDateDesc returns the passed commits, sorted by `committer.When desc`
//
// Following this strategy, it is tried to reduce the time needed when walking
// the history from one commit to reach the others. It is assumed that ancestors
// use to be committed before its descendant;
// That way `Independents(A^, A)` will be processed as being `Independents(A, A^)`;
// so starting by `A` it will be reached `A^` way sooner than walking from `A^`
// to the initial commit, and then from `A` to `A^`.
func sortByCommitDateDesc(commits ...*object.Commit) []*object.Commit {
	sorted := make([]*object.Commit, len(commits))
	copy(sorted, commits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Committer.When.After(sorted[j].Committer.When)
	})

	return sorted
}

// indexOf returns the first position where target was found in the passed commits
func indexOf(commits []*object.Commit, target *object.Commit) int {
	for i, commit := range commits {
		if target.Hash == commit.Hash {
			return i
		}
	}

	return -1
}

// remove returns the passed commits excluding the commit toDelete
func remove(commits []*object.Commit, toDelete *object.Commit) []*object.Commit {
	res := make([]*object.Commit, len(commits))
	j := 0
	for _, commit := range commits {
		if commit.Hash == toDelete.Hash {
			continue
		}

		res[j] = commit
		j++
	}

	return res[:j]
}

// removeDuplicated removes duplicated commits from the passed slice of commits
func removeDuplicated(commits []*object.Commit) []*object.Commit {
	seen := make(map[plumbing.Hash]struct{}, len(commits))
	res := make([]*object.Commit, len(commits))
	j := 0
	for _, commit := range commits {
		if _, ok := seen[commit.Hash]; ok {
			continue
		}

		seen[commit.Hash] = struct{}{}
		res[j] = commit
		j++
	}

	return res[:j]
}

// isInIndexCommitFilter returns a commitFilter that returns true
// if the commit is in the passed index.
func isInIndexCommitFilter(index map[plumbing.Hash]struct{}) object.CommitFilter {
	return func(c *object.Commit) bool {
		_, ok := index[c.Hash]
		return ok
	}
}
