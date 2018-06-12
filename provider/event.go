package provider

import (
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type PushEvent struct {
	Reference plumbing.ReferenceName
	// Head of the reference after pushing.
	Head plumbing.Hash
	// Head of the reference before pushing.
	Base plumbing.Hash
	// Commits included on this push event.
	Commits []CommitPointer
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
