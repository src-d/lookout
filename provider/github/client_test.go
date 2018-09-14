package github

import (
	"testing"

	"github.com/src-d/lookout"
	"github.com/stretchr/testify/require"
	vcsurl "gopkg.in/sourcegraph/go-vcsurl.v1"
)

func TestClientPoolUpdate(t *testing.T) {
	require := require.New(t)

	p := NewClientPool()

	// add new client
	firstClient := &Client{}
	info11, _ := vcsurl.Parse("github.com/foo/bar1")
	info12, _ := vcsurl.Parse("github.com/foo/bar2")
	firstClientRepos := []*lookout.RepositoryInfo{
		info11,
		info12,
	}

	p.Update(firstClient, firstClientRepos)

	require.Len(p.Clients(), 1)

	c, ok := p.Client("foo", "bar1")
	require.True(ok)
	require.Equal(firstClient, c)

	c, ok = p.Client("foo", "bar2")
	require.True(ok)
	require.Equal(firstClient, c)

	require.Equal(firstClientRepos, p.ReposByClient(firstClient))

	// add one more client
	secondClient := &Client{}
	info21, _ := vcsurl.Parse("github.com/bar/foo1")
	info22, _ := vcsurl.Parse("github.com/bar/foo2")
	secondClientRepos := []*lookout.RepositoryInfo{
		info21,
		info22,
	}

	p.Update(secondClient, secondClientRepos)

	require.Len(p.Clients(), 2)

	c, ok = p.Client("bar", "foo1")
	require.True(ok)
	require.Equal(secondClient, c)

	c, ok = p.Client("bar", "foo2")
	require.True(ok)
	require.Equal(secondClient, c)

	require.Equal(secondClientRepos, p.ReposByClient(secondClient))

	// add new repo
	info13, _ := vcsurl.Parse("github.com/foo/bar3")
	firstClientRepos = append(firstClientRepos, info13)

	p.Update(firstClient, firstClientRepos)

	require.Equal(firstClientRepos, p.ReposByClient(firstClient))

	c, ok = p.Client("foo", "bar3")
	require.True(ok)
	require.Equal(firstClient, c)

	// remove repo
	firstClientRepos = []*lookout.RepositoryInfo{
		info11,
		info13,
	}
	p.Update(firstClient, firstClientRepos)

	require.Equal(firstClientRepos, p.ReposByClient(firstClient))

	_, ok = p.Client("foo", "bar2")
	require.False(ok)

	// remove client
	p.RemoveClient(secondClient)

	require.Len(p.Clients(), 1)
	_, ok = p.Client("bar", "foo1")
	require.False(ok)

	// update without repos
	p.Update(firstClient, []*lookout.RepositoryInfo{})
	require.Len(p.Clients(), 0)

	// update without repos once again
	p.Update(firstClient, []*lookout.RepositoryInfo{})
	require.Len(p.Clients(), 0)
}

func TestClientPoolMultipleDeleteRepos(t *testing.T) {
	require := require.New(t)

	p := NewClientPool()

	// add new client
	client := &Client{}
	info1, _ := vcsurl.Parse("github.com/foo/bar1")
	info2, _ := vcsurl.Parse("github.com/foo/bar2")
	info3, _ := vcsurl.Parse("github.com/foo/bar3")
	repos := []*lookout.RepositoryInfo{
		info1,
		info2,
		info3,
	}

	p.Update(client, repos)

	require.Len(p.ReposByClient(client), 3)

	// remove repos
	newRepos := []*lookout.RepositoryInfo{info2}
	p.Update(client, newRepos)

	require.Equal(newRepos, p.ReposByClient(client))
}
