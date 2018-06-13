package github

import (
	"fmt"

	"github.com/google/go-github/github"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-log.v1"

	"github.com/src-d/lookout/provider"
)

func castEvent(r *vcsurl.RepoInfo, e *github.Event) (interface{}, error) {
	switch *e.Type {
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

		return castPullRequestEvent(r, payload.(*github.PullRequestEvent)), nil
	}

	return nil, nil
}

func castPushEvent(r *vcsurl.RepoInfo, e *github.Event, push *github.PushEvent) *provider.PushEvent {
	pe := &provider.PushEvent{}
	pe.CreatedAt = e.GetCreatedAt()
	pe.Commits = push.GetSize()
	pe.DistinctCommits = push.GetDistinctSize()

	pe.Head = provider.ReferencePointer{
		Repository: r,
		Reference:  plumbing.NewReferenceFromStrings(push.GetRef(), push.GetHead()),
	}

	pe.Base = provider.ReferencePointer{
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

func castPullRequestEvent(r *vcsurl.RepoInfo, e *github.PullRequestEvent) *provider.PullRequestEvent {
	if e.PullRequest == nil && e.PullRequest.GetID() != 0 {
		log.Warningf("missing pull request information in pull request event")
		return nil
	}

	pre := &provider.PullRequestEvent{}
	pre.Source = castPullRequestBranch(e.PullRequest.GetHead())
	pre.Merge = provider.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/pull/%d/merge", e.PullRequest.GetNumber()),
			e.PullRequest.GetMergeCommitSHA(),
		),
	}

	pre.Base = castPullRequestBranch(e.PullRequest.GetBase())
	pre.Head = provider.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/pull/%d/head", e.PullRequest.GetNumber()),
			e.PullRequest.GetHead().GetSHA(),
		),
	}

	pre.IsMergeable = e.PullRequest.GetMergeable()

	e.PullRequest.GetHead().GetRepo().GetURL()

	return pre
}

func castPullRequestBranch(b *github.PullRequestBranch) provider.ReferencePointer {
	if b == nil {
		log.Warningf("empty pull request branch given")
		return provider.ReferencePointer{}
	}

	r, err := vcsurl.Parse(b.GetRepo().GetCloneURL())
	if err != nil {
		log.Warningf("malformed repository URL on pull request branch")
		return provider.ReferencePointer{}
	}

	return provider.ReferencePointer{
		Repository: r,
		Reference: plumbing.NewReferenceFromStrings(
			fmt.Sprintf("refs/heads/%s", b.GetRef()),
			b.GetSHA(),
		),
	}
}
