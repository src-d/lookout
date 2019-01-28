package github

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-github/github"
	"github.com/src-d/lookout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

func TestClientPoolUpdate(t *testing.T) {
	require := require.New(t)

	p := NewClientPool()

	// add new client
	firstClient := &Client{}
	info11, _ := pb.ParseRepositoryInfo("github.com/foo/bar1")
	info12, _ := pb.ParseRepositoryInfo("github.com/foo/bar2")
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
	info21, _ := pb.ParseRepositoryInfo("github.com/bar/foo1")
	info22, _ := pb.ParseRepositoryInfo("github.com/bar/foo2")
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
	info13, _ := pb.ParseRepositoryInfo("github.com/foo/bar3")
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
	info1, _ := pb.ParseRepositoryInfo("github.com/foo/bar1")
	info2, _ := pb.ParseRepositoryInfo("github.com/foo/bar2")
	info3, _ := pb.ParseRepositoryInfo("github.com/foo/bar3")
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
func TestErrorResponseDoesNotPanic(t *testing.T) {
	require := require.New(t)

	url, _ := url.Parse("http://example.com")

	mockResponse := &github.Response{Response: &http.Response{
		StatusCode: http.StatusOK,
		Request:    &http.Request{Method: "GET", URL: url},
	}}

	apiResponseErrWithoutEmbededResponse := &github.ErrorResponse{Response: nil}

	apiResponseErrWithEmbededResponseWithoutRequest := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Request:    nil,
		},
	}

	apiResponseErrWithProperEmbededResponse := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Request:    &http.Request{Method: "GET", URL: url},
		},
	}

	processAPIError := func(apiErr error) assert.PanicTestFunc {
		return func() {
			err := handleAPIError(mockResponse, apiErr, "")
			msg := fmt.Sprintf("%s", err.Error())
			msg += ""
		}
	}

	require.NotPanics(processAPIError(apiResponseErrWithProperEmbededResponse), "API error should not panic when stringed")
	require.NotPanics(processAPIError(apiResponseErrWithEmbededResponseWithoutRequest), "API error with embedded empty response should not panic when stringed")
	require.NotPanics(processAPIError(apiResponseErrWithoutEmbededResponse), "empty API error should not panic when stringed")
}
