package git

import (
	"fmt"

	"github.com/src-d/lookout"

	"gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

func Example() {
	if err := fixtures.Init(); err != nil {
		panic(err)
	}
	defer fixtures.Clean()

	fixture := fixtures.Basic().One()
	fs := fixture.DotGit()
	storer, err := filesystem.NewStorage(fs)
	if err != nil {
		panic(err)
	}

	// Create the git service with a repository loader that allows it to find
	// a repository by ID.
	srv := NewService(server.MapLoader{
		"repo://myrepo": storer,
	})

	changes, err := srv.GetChanges(&lookout.ChangesRequest{
		Base: &lookout.ReferencePointer{
			InternalRepositoryURL: "repo:///myrepo",
			Hash: "af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
		},
		Head: &lookout.ReferencePointer{
			InternalRepositoryURL: "repo:///myrepo",
			Hash: "6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		},
	})
	if err != nil {
		panic(err)
	}

	for changes.Next() {
		change := changes.Change()
		fmt.Printf("changed: %s\n", change.Head.Path)
	}

	if err := changes.Err(); err != nil {
		panic(err)
	}

	if err := changes.Close(); err != nil {
		panic(err)
	}

	// Output: changed: go/example.go
	// changed: php/crappy.php
	// changed: vendor/foo.go
}
