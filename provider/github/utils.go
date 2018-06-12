package github

import (
	"fmt"

	"github.com/google/go-github/github"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
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

		return castPushEvent(r, payload.(*github.PushEvent))
	case "PullRequestEvent":
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, ErrParsingEventPayload.New(err)
		}

		return castPullRequestEvent(r, payload.(*github.PullRequestEvent))
	}

	return nil, nil
}

func castPushEvent(r *vcsurl.RepoInfo, e *github.PushEvent) (*provider.PushEvent, error) {
	return &provider.PushEvent{
		Head: provider.ReferencePointer{
			Repository: r,
			Reference:  plumbing.NewReferenceFromStrings(e.GetRef(), e.GetHead()),
		},
		Base: provider.ReferencePointer{
			Repository: r,
			Reference:  plumbing.NewReferenceFromStrings(e.GetRef(), e.GetBefore()),
		},
		Commits: castPushEventCommits(e.Commits),
	}, nil
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

func castPushEventCommits(commits []github.PushEventCommit) []provider.CommitPointer {
	output := make([]provider.CommitPointer, len(commits))
	for i, c := range commits {
		output[i] = castPushEventCommit(c)
	}

	return output
}

func castPushEventCommit(c github.PushEventCommit) provider.CommitPointer {
	return provider.CommitPointer{
		Hash:     castHash(c.SHA),
		Author:   castCommitAuthor(c.Author),
		Message:  c.GetMessage(),
		Distinct: c.GetDistinct(),
	}
}

func castCommitAuthor(a *github.CommitAuthor) object.Signature {
	return object.Signature{
		Name:  a.GetName(),
		Email: a.GetEmail(),
	}
}

func castPullRequestEvent(r *vcsurl.RepoInfo, e *github.PullRequestEvent) (*provider.PullRequestEvent, error) {
	if e.PullRequest == nil {
		log.Warningf("missing pull request information in pull request event")
		return nil, nil
	}

	pre := &provider.PullRequestEvent{}
	pre.Head = castPullRequestBranch(e.PullRequest.GetHead())
	pre.Base = castPullRequestBranch(e.PullRequest.GetBase())
	pre.Mergeable = *e.PullRequest.Mergeable

	e.PullRequest.GetHead().GetRepo().GetURL()

	return pre, nil
}

func castPullRequestBranch(b *github.PullRequestBranch) provider.ReferencePointer {
	if b == nil {
		log.Warningf("empty pull request branch given")
		return provider.ReferencePointer{}
	}

	r, err := vcsurl.Parse(b.GetRepo().GetURL())
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
