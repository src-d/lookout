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
	"github.com/pressly/lg"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

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
		lg.RequestLog(r).Warn(err.Error())
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
		lg.RequestLog(r).Warn(err.Error())
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

	user, err := a.getUser(r.Context(), code)
	if err != nil {
		lg.RequestLog(r).Warn(fmt.Errorf("oauth get user error: %s", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	token, err := a.makeToken(user)
	if err != nil {
		lg.RequestLog(r).Warn(fmt.Errorf("make jwt token error: %s", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(callbackResp{token})
	if err != nil {
		lg.RequestLog(r).Warn(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Write(b)
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
		r = r.WithContext(SetUserID(r.Context(), claims.ID))
		next.ServeHTTP(w, r)
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
func (a *Auth) getUser(ctx context.Context, code string) (*User, error) {
	token, err := a.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange error: %s", err)
	}

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
	ID int
	jwt.StandardClaims
}

// makeToken generates token string for a user
func (a *Auth) makeToken(user *User) (string, error) {
	claims := &jwtClaim{ID: user.ID}
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

type userIDContext int

const userIDKey userIDContext = 1

// getUserInt gets the value stored in the Context for the key userIDKey, bool
// is true on success
func getUserInt(ctx context.Context) (int, bool) {
	i, ok := ctx.Value(userIDKey).(int)
	return i, ok
}

// SetUserID sets the user ID to the context
func SetUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID gets the user ID set by the JWT middleware in the Context
func GetUserID(ctx context.Context) (int, error) {
	id, ok := getUserInt(ctx)
	if !ok {
		return 0, fmt.Errorf("User ID is not set in the context")
	}

	return id, nil
}
