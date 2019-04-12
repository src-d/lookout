package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

// set default urls to empty to make sure tests don't hit github
func init() {
	defaultBaseURL = ""
	defaultUploadBaseURL = ""
}

func TestClientPoolUpdate(t *testing.T) {
	require := require.New(t)

	p := NewClientPool()

	// add new client
	firstClient := &Client{}
	info11, _ := parseTestRepositoryInfo("github.com/foo/bar1")
	info12, _ := parseTestRepositoryInfo("github.com/foo/bar2")
	firstClientRepos := []*repositoryInfo{
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
	info21, _ := parseTestRepositoryInfo("github.com/bar/foo1")
	info22, _ := parseTestRepositoryInfo("github.com/bar/foo2")
	secondClientRepos := []*repositoryInfo{
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
	info13, _ := parseTestRepositoryInfo("github.com/foo/bar3")
	firstClientRepos = append(firstClientRepos, info13)

	p.Update(firstClient, firstClientRepos)

	require.Equal(firstClientRepos, p.ReposByClient(firstClient))

	c, ok = p.Client("foo", "bar3")
	require.True(ok)
	require.Equal(firstClient, c)

	// remove repo
	firstClientRepos = []*repositoryInfo{
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
	p.Update(firstClient, []*repositoryInfo{})
	require.Len(p.Clients(), 0)

	// update without repos once again
	p.Update(firstClient, []*repositoryInfo{})
	require.Len(p.Clients(), 0)
}

func TestClientPoolMultipleDeleteRepos(t *testing.T) {
	require := require.New(t)

	p := NewClientPool()

	// add new client
	client := &Client{}
	info1, _ := parseTestRepositoryInfo("github.com/foo/bar1")
	info2, _ := parseTestRepositoryInfo("github.com/foo/bar2")
	info3, _ := parseTestRepositoryInfo("github.com/foo/bar3")
	repos := []*repositoryInfo{
		info1,
		info2,
		info3,
	}

	p.Update(client, repos)

	require.Len(p.ReposByClient(client), 3)

	// remove repos
	newRepos := []*repositoryInfo{info2}
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

// parseTestRepositoryInfo is a convenience wrapper around pb.ParseRepositoryInfo
// for testing
func parseTestRepositoryInfo(input string) (*repositoryInfo, error) {
	r, err := pb.ParseRepositoryInfo(input)
	if err != nil {
		return nil, err
	}

	return &repositoryInfo{RepositoryInfo: *r}, nil
}

func TestValidateTokenPermissions(t *testing.T) {
	require := require.New(t)

	c := mockedClientWithScopes("")
	require.EqualError(ValidateTokenPermissions(c), "token doesn't have permission scope 'repo'")

	c = mockedClientWithScopes("a,b")
	require.EqualError(ValidateTokenPermissions(c), "token doesn't have permission scope 'repo'")

	c = mockedClientWithScopes("repo")
	require.NoError(ValidateTokenPermissions(c))

	c = mockedClientWithScopes("a,repo")
	require.NoError(ValidateTokenPermissions(c))

	c = mockedClientWithScopes("repo,b")
	require.NoError(ValidateTokenPermissions(c))
}

func TestCanPostStatus(t *testing.T) {
	require := require.New(t)

	mt := roundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path == "/user" {
			b, _ := json.Marshal(&github.User{Login: strptr("test")})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer(b)),
				Header:     make(http.Header),
			}
		}

		for _, access := range []string{"none", "read", "write", "admin"} {
			if req.URL.Path == "/repos/access/"+access+"/collaborators/test/permission" {
				b, _ := json.Marshal(github.RepositoryPermissionLevel{Permission: strptr(access)})
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(b)),
					Header:     make(http.Header),
				}
			}
		}

		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}
	})

	client := NewClient(mt, nil, "", nil, time.Millisecond)

	repo, _ := parseTestRepositoryInfo("github.com/access/none")
	require.EqualError(CanPostStatus(client, repo), "token doesn't have write access to repository access/none")

	repo, _ = parseTestRepositoryInfo("github.com/access/read")
	require.EqualError(CanPostStatus(client, repo), "token doesn't have write access to repository access/read")

	repo, _ = parseTestRepositoryInfo("github.com/access/write")
	require.NoError(CanPostStatus(client, repo))

	repo, _ = parseTestRepositoryInfo("github.com/access/admin")
	require.NoError(CanPostStatus(client, repo))
}

func mockedClientWithScopes(scopes string) *Client {
	mt := roundTripFunc(func(req *http.Request) *http.Response {
		h := make(http.Header)
		h.Add("X-Oauth-Scopes", scopes)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("{}")),
			Header:     h,
		}
	})
	return NewClient(mt, nil, "", nil, time.Millisecond)
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// returns correct permissions to pass the permissions checks
func mockPermissions(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			// set headers to pass token checks
			w.Header().Add("X-Oauth-Scopes", "repo")
			json.NewEncoder(w).Encode(&github.User{})
			return
		}

		if strings.HasSuffix(r.URL.Path, "/collaborators/permission") {
			json.NewEncoder(w).Encode(&github.RepositoryPermissionLevel{Permission: strptr("write")})
			return
		}

		h.ServeHTTP(w, r)
	})
}
