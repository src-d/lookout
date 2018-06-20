package github

import (
	"fmt"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-log.v1"
)

func castEvent(r *vcsurl.RepoInfo, e *github.Event) (lookout.Event, error) {
	switch e.GetType() {
	case "PushEvent":
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, ErrParsingEventPayload.New(err)
		}

		return castPushEvent(r, e, payload.(*github.PushEvent)), nil
	case "PullRequestEvent":
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, ErrParsingEventPayload.New(err)
		}

		return castPullRequestEvent(r, e, payload.(*github.PullRequestEvent)), nil
	}

	return nil, nil
}

func castPushEvent(r *vcsurl.RepoInfo, e *github.Event, push *github.PushEvent) *lookout.PushEvent {
	pe := &lookout.PushEvent{}
	pe.Provider = Provider
	pe.InternalID = e.GetID()
	pe.CreatedAt = e.GetCreatedAt()
	pe.Commits = push.GetSize()
	pe.DistinctCommits = push.GetDistinctSize()

	pe.Head = lookout.ReferencePointer{
		Repository: r,
		Reference:  plumbing.NewReferenceFromStrings(push.GetRef(), push.GetHead()),
	}

	pe.Base = lookout.ReferencePointer{
		Repository: r,
		Reference:  plumbing.NewReferenceFromStrings(push.GetRef(), push.GetBefore()),
	}

	return pe
}

func castReferenceName(ref *string) plumbing.ReferenceName {
	if ref == nil {
		return ""
	}

	return plumbing.ReferenceName(*ref)
}

func castHash(sha1 *string) plumbing.Hash {
	if sha1 == nil {
		return plumbing.ZeroHash
	}

	return plumbing.NewHash(*sha1)
}

func castPullRequestEvent(r *vcsurl.RepoInfo, e *github.Event, pr *github.PullRequestEvent) *lookout.PullRequestEvent {
	if pr.PullRequest == nil && pr.PullRequest.GetID() != 0 {
		log.Warningf("missing pull request information in pull request event")
		return nil
	}

	pre := &lookout.PullRequestEvent{}
	pre.Provider = Provider
	pre.InternalID = e.GetID()
	pre.Source = castPullRequestBranch(pr.PullRequest.GetHead())
	pre.Merge = lookout.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/pull/%d/merge", pr.PullRequest.GetNumber()),
			pr.PullRequest.GetMergeCommitSHA(),
		),
	}

	pre.Base = castPullRequestBranch(pr.PullRequest.GetBase())
	pre.Head = lookout.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/pull/%d/head", pr.PullRequest.GetNumber()),
			pr.PullRequest.GetHead().GetSHA(),
		),
	}

	pre.IsMergeable = pr.PullRequest.GetMergeable()

	pr.PullRequest.GetHead().GetRepo().GetURL()

	return pre
}

func castPullRequestBranch(b *github.PullRequestBranch) lookout.ReferencePointer {
	if b == nil {
		log.Warningf("empty pull request branch given")
		return lookout.ReferencePointer{}
	}

	r, err := vcsurl.Parse(b.GetRepo().GetCloneURL())
	if err != nil {
		log.Warningf("malformed repository URL on pull request branch")
		return lookout.ReferencePointer{}
	}

	return lookout.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/heads/%s", b.GetRef()),
			b.GetSHA(),
		),
	}
}
