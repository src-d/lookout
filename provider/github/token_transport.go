package github

import (
	"net/http"

	log "gopkg.in/src-d/go-log.v1"
)

// Transport returns RoundTripper for a repository
type Transport interface {
	Get(repo string) http.RoundTripper
}

// TODO replace with github installation transport later

// TokenTransport returns RoundTripper for a repository based on pre-configured tokens
type TokenTransport struct {
	UserToken
	perRepo map[string]UserToken
	rts     map[string]http.RoundTripper
}

// UserToken holds github username and token
type UserToken struct {
	user  string
	token string
}

// NewTokenTransport creates new TokenTransport
func NewTokenTransport(user, token string, perRepo map[string]UserToken) *TokenTransport {
	if perRepo == nil {
		perRepo = make(map[string]UserToken)
	}

	return &TokenTransport{UserToken: UserToken{user, token}, perRepo: perRepo}
}

// Get implements Transport interface
func (t *TokenTransport) Get(repo string) http.RoundTripper {
	rt, ok := t.rts[repo]
	if !ok {
		rt = t.create(repo)
		t.rts[repo] = rt
	}

	return rt
}

func (t *TokenTransport) create(repo string) http.RoundTripper {
	user := t.user
	token := t.token

	ut, ok := t.perRepo[repo]
	if ok {
		user = ut.user
		token = ut.token
	}

	return &roundTripper{
		Log:      log.DefaultLogger,
		User:     user,
		Password: token,
	}
}

type roundTripper struct {
	Log      log.Logger
	Base     http.RoundTripper
	User     string
	Password string
}

func (t *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Log.With(log.Fields{
		"url":  req.URL.String(),
		"user": t.User,
	}).Debugf("http request")

	if t.User != "" {
		req.SetBasicAuth(t.User, t.Password)
	}

	rt := t.Base
	if rt == nil {
		rt = http.DefaultTransport
	}

	return rt.RoundTrip(req)
}

var _ http.RoundTripper = &roundTripper{}
