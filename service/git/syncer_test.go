package git

import (
	"context"
	"testing"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

func TestLibrary_Sync(t *testing.T) {
	require := require.New(t)
	library := NewLibrary(memfs.New())
	syncer := NewSyncer(library, nil, 0)

	url, _ := pb.ParseRepositoryInfo("https://github.com/src-d/lookout")
	err := syncer.Sync(context.TODO(), lookout.ReferencePointer{
		InternalRepositoryURL: url.CloneURL,
		ReferenceName:         "refs/pull/1/head",
		Hash:                  "80a9810a027672a098b07efda3dc305409c9329d",
	})

	require.NoError(err)
	has, err := library.Has(url)
	require.NoError(err)
	require.True(has)
}

var _ AuthProvider = &testAuthProvider{}

type testAuthProvider struct{}

var authCalls int

func (p testAuthProvider) GitAuth(ctx context.Context, repoInfo *lookout.RepositoryInfo) transport.AuthMethod {
	authCalls++

	return &githttp.BasicAuth{
		Username: "",
		Password: "",
	}
}

func TestLibrary_Auth(t *testing.T) {
	require := require.New(t)

	require.Equal(0, authCalls)

	library := NewLibrary(memfs.New())
	syncer := NewSyncer(library, testAuthProvider{}, 0)

	url, _ := pb.ParseRepositoryInfo("https://github.com/src-d/lookout")
	err := syncer.Sync(context.TODO(), lookout.ReferencePointer{
		InternalRepositoryURL: url.CloneURL,
		ReferenceName:         "refs/pull/1/head",
		Hash:                  "80a9810a027672a098b07efda3dc305409c9329d",
	})

	require.NoError(err)
	has, err := library.Has(url)
	require.NoError(err)
	require.True(has)

	require.Equal(1, authCalls)
}
