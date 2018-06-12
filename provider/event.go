package provider

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/sourcegraph/go-vcsurl.v1"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type PullRequestEvent struct {
	ID        string //TODO: improve ID references
	Action    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Mergeable bool
	PushEvent
}

func (e *PullRequestEvent) String() string {
	return fmt.Sprintf(
		"[pull-request] %s -{%d}-> %s",
		e.Head.Reference.String(), len(e.Commits), e.Base.Reference.String(),
	)
}

type PushEvent struct {
	Head ReferencePointer
	// Head of the reference before pushing.
	Base ReferencePointer
	// Commits included on this push event.
	Commits CommitPointers
}

func (e *PushEvent) String() string {
	return fmt.Sprintf(
		"[push] %s -{%d}-> %s",
		e.Head.Reference.String(), len(e.Commits), e.Base.Reference.String(),
	)
}

type ReferencePointer struct {
	Repository *vcsurl.RepoInfo //TODO(mcuadros): improve repository references
	Reference  *plumbing.Reference
}

type CommitPointer struct {
	// Hahs of the commit.
	Hash plumbing.Hash
	// Author of the commit.
	Author object.Signature
	// Message is the commit message.
	Message string
	// Distinct whether this commit is distinct from any that have been pushed
	// before.
	Distinct bool
}

func (c *CommitPointer) String() string {
	return fmt.Sprintf(
		"%s %s\nAuthor: %s\n\n%s\n",
		plumbing.CommitObject, c.Hash, c.Author.String(),
		indent(c.Message),
	)
}

type CommitPointers []CommitPointer

func (c CommitPointers) String() string {
	output := ""
	for _, commit := range c {
		output += commit.String()
	}

	return output
}

func indent(t string) string {
	var output []string
	for _, line := range strings.Split(t, "\n") {
		if len(line) != 0 {
			line = "    " + line
		}

		output = append(output, line)
	}

	return strings.Join(output, "\n")
}
