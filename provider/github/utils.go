package github

import (
	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/src-d/lookout/provider"
)

func castEvent(e *github.Event) (interface{}, error) {
	switch *e.Type {
	case "PushEvent":
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, ErrParsingEventPayload.New(err)
		}

		return castPushEvent(payload.(*github.PushEvent))
	}

	return nil, nil
}

func castPushEvent(e *github.PushEvent) (*provider.PushEvent, error) {
	return &provider.PushEvent{
		Reference: castReferenceName(e.Ref),
		Head:      castHash(e.Head),
		Base:      castHash(e.Before),
		Commits:   castPushEventCommits(e.Commits),
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
