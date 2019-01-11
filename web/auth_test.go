package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestLogin(t *testing.T) {
	require := require.New(t)
	auth := NewAuth("client-id", "client-secret", "signing-key")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/login", nil)
	auth.Login(w, r)

	require.Equal(http.StatusTemporaryRedirect, w.Code)

	loc := w.Header().Get("Location")
	require.NotEmpty(loc)
	require.True(strings.HasPrefix(loc, "https://github.com/login/oauth/authorize"))

	cookie := w.Header().Get("Set-Cookie")
	require.NotEmpty(cookie)
	require.True(strings.HasPrefix(cookie, "sess="))
}

func TestCallbackSuccess(t *testing.T) {
	require := require.New(t)

	oauthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken: "access-token",
		}

		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	github := httptest.NewServer(oauthHandler)
	defer github.Close()

	testUser := &User{
		ID:       1,
		Username: "test-name",
	}

	auth := NewAuth("client-id", "client-secret", "signing-key")
	auth.config.Endpoint = oauth2.Endpoint{
		AuthURL:  github.URL,
		TokenURL: github.URL,
	}
	auth.userGetter = func(c *http.Client) (*User, error) {
		return testUser, nil
	}

	state := "test-state"
	code := "test-code"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/callback?state="+state+"&code="+code, nil)

	session, _ := auth.store.Get(r, "sess")
	session.Values["state"] = state

	auth.Callback(w, r)

	require.Equal(http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Token string
		}
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(err)

	oauthToken := oauth2.Token{AccessToken: "access-token"}
	token, err := auth.makeToken(oauthToken, testUser)
	require.NoError(err)
	require.Equal(token, resp.Data.Token)
}

func TestMiddlewareSuccess(t *testing.T) {
	require := require.New(t)
	auth := NewAuth("client-id", "client-secret", "signing-key")

	testUser := &User{
		ID:       1,
		Username: "test-name",
	}

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := GetUserID(r.Context())
		require.NoError(err)
		require.Equal(testUser.ID, id)
	}))

	token, err := auth.makeToken(oauth2.Token{}, testUser)
	require.NoError(err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(w, r)

	require.Equal(http.StatusOK, w.Code)
}

func TestMiddlewareUnauthorized(t *testing.T) {
	require := require.New(t)
	auth := NewAuth("client-id", "client-secret", "signing-key")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	require.Equal(http.StatusUnauthorized, w.Code)
}
