package lookout

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

func TestLibrary_Has(t *testing.T) {
	require := require.New(t)

	url, _ := vcsurl.Parse("http://github.com/foo/bar")
	library := NewLibrary(memfs.New())
	has, err := library.Has(url)
	require.NoError(err)
	require.False(has)
}

func TestLibrary_Init(t *testing.T) {
	require := require.New(t)

	url, _ := vcsurl.Parse("http://github.com/foo/bar")
	library := NewLibrary(memfs.New())

	r, err := library.Init(url)
	require.NoError(err)
	require.NotNil(r)

	remote, err := r.Remote("origin")
	require.NoError(err)
	require.NotNil(remote)

	has, err := library.Has(url)
	require.NoError(err)
	require.True(has)
}

func TestLibrary_InitExists(t *testing.T) {
	require := require.New(t)

	url, _ := vcsurl.Parse("http://github.com/foo/bar")
	library := NewLibrary(memfs.New())

	r, err := library.Init(url)
	require.NoError(err)
	require.NotNil(r)

	r, err = library.Init(url)
	require.True(ErrRepositoryExists.Is(err))
	require.Nil(r)
}

func TestLibrary_Get(t *testing.T) {
	require := require.New(t)

	url, _ := vcsurl.Parse("http://github.com/foo/bar")
	library := NewLibrary(memfs.New())

	_, err := library.Init(url)
	require.NoError(err)

	r, err := library.Get(url)
	require.NoError(err)
	require.NotNil(r)
}

func TestLibrary_GetOrInit(t *testing.T) {
	require := require.New(t)

	url, _ := vcsurl.Parse("http://github.com/foo/bar")
	library := NewLibrary(memfs.New())

	r, err := library.GetOrInit(url)
	require.NoError(err)
	require.NotNil(r)

	r, err = library.GetOrInit(url)
	require.NoError(err)
	require.NotNil(r)
}
