package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v24/github"
	"github.com/gregjones/httpcache"
	"github.com/src-d/lookout/util/cache"
	"github.com/stretchr/testify/require"
)

func TestInstallationsGetRepos(t *testing.T) {
	require := require.New(t)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	var calls int32
	path := "/installation/repositories"
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		q := r.URL.Query()
		require.Equal("100", q.Get("per_page"))
		curPage := q.Get("page")
		if curPage == "" {
			curPage = "1"
		}

		if curPage == "1" {
			nextLink := fmt.Sprintf(
				`<%s%s?per_page=%s&page=%s>; rel="next"`,
				server.URL, path, q.Get("per_page"), "2",
			)
			w.Header().Add("Link", nextLink)
		}

		b, _ := json.Marshal(struct {
			Repositories []github.Repository
		}{
			Repositories: []github.Repository{
				{
					HTMLURL: strptr("https://github.com/test/repo" + curPage),
				},
			},
		})

		w.Write(b)
	})

	githubURL, _ := url.Parse(server.URL + "/")
	client := NewClient(nil, cache.NewValidableCache(httpcache.NewMemoryCache()), "", nil, 0)
	client.BaseURL = githubURL

	inst := Installations{}

	repos, err := inst.getRepos(client)
	require.NoError(err)
	require.Equal(int32(2), atomic.LoadInt32(&calls))
	require.Len(repos, 2)
	require.Equal("test/repo1", repos[0].FullName)
	require.Equal("test/repo2", repos[1].FullName)
}
