package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

func castEvent(r *repositoryInfo, e *github.Event) (lookout.Event, error) {
	switch e.GetType() {
	case "PushEvent":
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, ErrParsingEventPayload.New(err)
		}

		return castPushEvent(r, e, payload.(*github.PushEvent)), nil
	}

	return nil, nil
}

func castPushEvent(r *repositoryInfo, e *github.Event, push *github.PushEvent) *lookout.PushEvent {
	pe := &lookout.PushEvent{}
	pe.Provider = Provider
	pe.InternalID = e.GetID()
	pe.CreatedAt = e.GetCreatedAt()
	pe.Commits = uint32(push.GetSize())
	pe.DistinctCommits = uint32(push.GetDistinctSize())

	pe.Head = lookout.ReferencePointer{
		InternalRepositoryURL: r.CloneURL,
		ReferenceName:         plumbing.ReferenceName(push.GetRef()),
		Hash:                  push.GetHead(),
	}

	pe.Base = lookout.ReferencePointer{
		InternalRepositoryURL: r.CloneURL,
		ReferenceName:         plumbing.ReferenceName(push.GetRef()),
		Hash:                  push.GetBefore(),
	}

	pe.OrganizationID = r.OrganizationID

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

func castPullRequest(ctx context.Context, r *repositoryInfo, pr *github.PullRequest) *lookout.ReviewEvent {
	pre := &lookout.ReviewEvent{}
	pre.Provider = Provider
	pre.InternalID = strconv.FormatInt(pr.GetID(), 10)

	pre.Number = uint32(pr.GetNumber())
	pre.RepositoryID = uint32(pr.GetHead().GetRepo().GetID())
	pre.Source = castPullRequestBranch(ctx, pr.GetHead())
	pre.Merge = lookout.ReferencePointer{
		InternalRepositoryURL: r.CloneURL,
		ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/pull/%d/merge", pr.GetNumber())),
		Hash:                  pr.GetMergeCommitSHA(),
	}

	pre.Base = castPullRequestBranch(ctx, pr.GetBase())
	pre.Head = lookout.ReferencePointer{
		InternalRepositoryURL: r.CloneURL,
		ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/pull/%d/head", pr.GetNumber())),
		Hash:                  pr.GetHead().GetSHA(),
	}

	pre.IsMergeable = pr.GetMergeable()

	pre.OrganizationID = r.OrganizationID

	return pre
}

func castPullRequestBranch(ctx context.Context, b *github.PullRequestBranch) lookout.ReferencePointer {
	if b == nil {
		ctxlog.Get(ctx).Warningf("empty pull request branch given")
		return lookout.ReferencePointer{}
	}

	r, err := pb.ParseRepositoryInfo(b.GetRepo().GetCloneURL())
	if err != nil {
		ctxlog.Get(ctx).With(log.Fields{
			"url": b.GetRepo().GetCloneURL()},
		).Warningf("malformed repository URL on pull request branch")

		return lookout.ReferencePointer{}
	}

	return lookout.ReferencePointer{
		InternalRepositoryURL: r.CloneURL,
		ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", b.GetRef())),
		Hash:                  b.GetSHA(),
	}
}

func extractOwner(ref lookout.ReferencePointer) (owner string, err error) {
	if ref.Repository() == nil {
		err = fmt.Errorf("nil repository")
		return
	}

	owner = ref.Repository().Owner
	if owner == "" {
		err = fmt.Errorf("empty owner")
	}

	return
}

func extractRepo(ref lookout.ReferencePointer) (repo string, err error) {
	if ref.Repository() == nil {
		err = fmt.Errorf("nil repository")
		return
	}

	repo = ref.Repository().Name
	if repo == "" {
		err = fmt.Errorf("empty repository name")
	}

	return
}
