package git

import (
	"context"
	"testing"

	"github.com/src-d/lookout"
	"github.com/stretchr/testify/require"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

func TestLibrary_Sync(t *testing.T) {
	require := require.New(t)
	library := NewLibrary(memfs.New())
	syncer := NewSyncer(library)

	url, _ := vcsurl.Parse("http://github.com/src-d/lookout")
	err := syncer.Sync(context.TODO(), &lookout.CommitRevision{
		Head: lookout.ReferencePointer{
			InternalRepositoryURL: url.CloneURL,
			ReferenceName:         "refs/pull/1/head",
			Hash:                  "80a9810a027672a098b07efda3dc305409c9329d",
		},
	})

	require.NoError(err)
	has, err := library.Has(url)
	require.NoError(err)
	require.True(has)
}
