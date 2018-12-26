package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/sessions"
	"github.com/src-d/lookout/util/ctxlog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// TODO: move/rewrite it when we have other endpoints

type successResp struct {
	Data interface{} `json:"data"`
}
type errorResp struct {
	Errors []error `json:"errors"`
}

func successJSON(w http.ResponseWriter, r *http.Request, data interface{}) {
	b, err := json.Marshal(successResp{data})
	if err != nil {
		ctxlog.Get(r.Context()).Warningf(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func errorJSON(w http.ResponseWriter, r *http.Request, code int, errors ...error) {
	b, err := json.Marshal(errorResp{errors})
	if err != nil {
		ctxlog.Get(r.Context()).Warningf(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
	w.Write(b)
}

// Auth is http service to authorize users, it uses oAuth and JWT underneath
type Auth struct {
	config     *oauth2.Config
	store      *sessions.CookieStore
	signingKey []byte
	userGetter func(client *http.Client) (*User, error)
}

// NewAuth create new Auth service
func NewAuth(clientID, clientSecret string, signingKey string) *Auth {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"read:user", "read:org"},
		Endpoint:     github.Endpoint,
	}

	return &Auth{
		config:     config,
		store:      sessions.NewCookieStore([]byte(clientSecret)),
		signingKey: []byte(signingKey),
		userGetter: getGithubUser,
	}
}

// User represents the user response returned by provider
type User struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Username  string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// Login handler redirects user to oauth provider
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	url, err := a.makeAuthURL(w, r)
	if err != nil {
		ctxlog.Get(r.Context()).Warningf(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// callbackResp defines successful response of Callback endpoint
type callbackResp struct {
	Token string `json:"token"`
}

// Callback makes exchange with oauth provider and redirects to index page with JWT token
func (a *Auth) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if err := a.validateState(r, state); err != nil {
		ctxlog.Get(r.Context()).Warningf(err.Error())
		http.Error(w, "The state passed by github is incorrect or expired", http.StatusPreconditionFailed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		errorText := r.URL.Query().Get("error_description")
		if errorText == "" {
			errorText = "OAuth provider didn't send code in callback"
		}

		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	oauthToken, err := a.config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("oauth exchange error: %s", err), http.StatusBadRequest)
		return
	}

	user, err := a.getUser(r.Context(), oauthToken)
	if err != nil {
		ctxlog.Get(r.Context()).Warningf("oauth get user error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	token, err := a.makeToken(*oauthToken, user)
	if err != nil {
		ctxlog.Get(r.Context()).Warningf("make jwt token error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	successJSON(w, r, callbackResp{token})
}

// Me endpoint make request to provider and returns user details
func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	t, err := getOAuthToken(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := a.getUser(r.Context(), t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	successJSON(w, r, u)
}

// Middleware return http.Handler which validates token and set user id in context
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var claims jwtClaim
		_, err := request.ParseFromRequestWithClaims(r, extractor, &claims, func(token *jwt.Token) (interface{}, error) {
			return a.signingKey, nil
		})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.ID)
		ctx = context.WithValue(ctx, userOAuthToken, &claims.OAuthToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// makeAuthURL returns string for redirect to provider
func (a *Auth) makeAuthURL(w http.ResponseWriter, r *http.Request) (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	session, err := a.store.Get(r, "sess")
	if err != nil {
		return "", fmt.Errorf("could not save state under the current session: %s", err)
	}

	session.Values["state"] = state
	if err := session.Save(r, w); err != nil {
		return "", fmt.Errorf("could not save state under the current session: %s", err)
	}

	return a.config.AuthCodeURL(state), nil
}

// validateState protects the user from CSRF attacks
func (a *Auth) validateState(r *http.Request, state string) error {
	session, err := a.store.Get(r, "sess")
	if err != nil {
		return fmt.Errorf("can't get session: %s", err)
	}

	expectedState := session.Values["state"]
	if state != expectedState {
		return fmt.Errorf("incorrect state: %s; expected: %s", state, expectedState)
	}

	return nil
}

// getUser gets user from provider and return user model
func (a *Auth) getUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return a.userGetter(a.config.Client(ctx, token))
}

func getGithubUser(client *http.Client) (*User, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("can't get user from github: %s", err)
	}
	defer resp.Body.Close()

	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("can't parse github user response: %s", err)
	}

	return &user, nil
}

type jwtClaim struct {
	ID         int
	OAuthToken oauth2.Token
	jwt.StandardClaims
}

// makeToken generates token string for a user
func (a *Auth) makeToken(ot oauth2.Token, user *User) (string, error) {
	claims := &jwtClaim{ID: user.ID, OAuthToken: ot}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := t.SignedString(a.signingKey)
	if err != nil {
		return "", fmt.Errorf("can't sign jwt token: %s", err)
	}
	return ss, nil
}

// Strips 'Bearer ' prefix from bearer token string
func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}

var extractor = &request.PostExtractionFilter{
	Extractor: &request.MultiExtractor{
		request.HeaderExtractor{"Authorization"},
		request.ArgumentExtractor{"jwt_token"},
	},
	Filter: stripBearerPrefixFromTokenString,
}

type userContext int

const userIDKey userContext = 1
const userOAuthToken userContext = 2

// getUserInt gets the value stored in the Context for the key userIDKey, bool
// is true on success
func getUserInt(ctx context.Context) (int, bool) {
	i, ok := ctx.Value(userIDKey).(int)
	return i, ok
}

// GetUserID gets the user ID set by the JWT middleware in the Context
func GetUserID(ctx context.Context) (int, error) {
	id, ok := getUserInt(ctx)
	if !ok {
		return 0, fmt.Errorf("User ID is not set in the context")
	}

	return id, nil
}

func getOAuthToken(ctx context.Context) (*oauth2.Token, error) {
	t, ok := ctx.Value(userOAuthToken).(*oauth2.Token)
	if !ok {
		return nil, fmt.Errorf("OAuth token is not set in the context")
	}

	return t, nil
}
