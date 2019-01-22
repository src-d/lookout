package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	github_provider "github.com/src-d/lookout/provider/github"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/go-chi/chi"
	"github.com/google/go-github/github"
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/ctxlog"
	yaml "gopkg.in/yaml.v2"
)

// GitHub is an HTTP service to call GitHub endpoints
type GitHub struct {
	AppID          int
	PrivateKey     string
	OrganizationOp store.OrganizationOperator
}

func (g *GitHub) appClient() (*github.Client, error) {
	appTr, err := ghinstallation.NewAppsTransportKeyFromFile(
		http.DefaultTransport, g.AppID, g.PrivateKey)
	if err != nil {
		return nil, err
	}

	return github.NewClient(&http.Client{Transport: appTr}), nil
}

func (g *GitHub) installations(ctx context.Context) ([]*github.Installation, error) {
	appClient, err := g.appClient()
	if err != nil {
		return nil, err
	}

	var installations []*github.Installation
	opts := &github.ListOptions{
		PerPage: 100,
	}
	for {
		installs, resp, err := appClient.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return nil, err
		}

		installations = append(installations, installs...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return installations, nil
}

var errNotFound = errors.New("not found")

// installation returns the GitHub App Installation corresponding to the given
// organization name. If the installation is not found the error is errNotFound
func (g *GitHub) installation(ctx context.Context, orgName string) (*github.Installation, error) {
	appClient, err := g.appClient()
	if err != nil {
		return nil, err
	}

	installation, _, err := appClient.Apps.FindOrganizationInstallation(ctx, orgName)
	if err != nil {
		if err.(*github.ErrorResponse).Response.StatusCode == http.StatusNotFound {
			return nil, errNotFound
		}

		return nil, err
	}

	return installation, nil
}

func (g *GitHub) isAdmin(ctx context.Context, installation *github.Installation, login string) (bool, error) {
	// New transport for each installation
	itr, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport, g.AppID, int(installation.GetID()), g.PrivateKey)

	if err != nil {
		return false, fmt.Errorf("failed to initialize the GitHub App installation client: %s", err)
	}

	// Use installation transport in a new client
	client := github.NewClient(&http.Client{Transport: itr})

	org := installation.GetAccount().GetLogin()
	mem, _, err := client.Organizations.GetOrgMembership(ctx, login, org)
	if err != nil {
		return false, fmt.Errorf("failed to get GitHub user %s membership to organization %s: %s", login, org, err)
	}

	return mem.GetRole() == "admin", nil
}

// orgsListItem is the response type used by the organizations list handler
type orgsListItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Orgs writes in the response the list of organizations where the
// logged-in user is an admin
func (g *GitHub) Orgs(w http.ResponseWriter, r *http.Request) {
	login, err := GetUserLogin(r.Context())
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to get user login from context")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	installations, err := g.installations(r.Context())
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to retrieve the GitHub App installations")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// initialized as empty array because otherwise json response will be null
	// instead of []
	orgs := []orgsListItem{}

	for _, installation := range installations {
		// a GitHub App can also be installed in a User account
		if installation.GetAccount().GetType() != "Organization" {
			continue
		}

		admin, err := g.isAdmin(r.Context(), installation, login)
		if err != nil {
			ctxlog.Get(r.Context()).Errorf(err, "failed to check user admin role")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if admin {
			orgs = append(orgs, orgsListItem{
				ID:   installation.GetAccount().GetID(),
				Name: installation.GetAccount().GetLogin(),
			})
		}
	}

	successJSON(w, r, orgs)
}

// TODO (@carlosms) check if this makes more sense as a middleware
// orgInstallation returns the GitHub App Installation corresponding to the URL
// parameter "orgName". The logged-in user must be an administrator of the
// organization. If there is any error the proper HTTP headers are set in w.
func (g *GitHub) orgInstallation(w http.ResponseWriter, r *http.Request) (*github.Installation, error) {
	login, err := GetUserLogin(r.Context())
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to get user login from context")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return nil, fmt.Errorf(http.StatusText(http.StatusUnauthorized))
	}

	orgName := chi.URLParam(r, "orgName")
	if orgName == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil, fmt.Errorf(http.StatusText(http.StatusBadRequest))
	}

	installation, err := g.installation(r.Context(), orgName)
	if err != nil {
		if err == errNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return nil, fmt.Errorf(http.StatusText(http.StatusNotFound))
		}

		ctxlog.Get(r.Context()).Errorf(err, "failed to get installation with name %v", orgName)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, fmt.Errorf(http.StatusText(http.StatusInternalServerError))
	}

	// a GitHub App can also be installed in a User account
	if installation.GetAccount().GetType() != "Organization" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil, fmt.Errorf(http.StatusText(http.StatusNotFound))
	}

	admin, err := g.isAdmin(r.Context(), installation, login)
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to check user admin role")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, fmt.Errorf(http.StatusText(http.StatusInternalServerError))
	}

	if !admin {
		// 404 to avoid leaking to a non admin if the org exists or is installed
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil, fmt.Errorf(http.StatusText(http.StatusNotFound))
	}

	return installation, nil
}

// orgResponse is the response type used by the individual organization handler
type orgResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Config string `json:"config"`
}

// Org writes in the response the individual organization requested by the URL
// parameter "orgName", only if the user is an admin
func (g *GitHub) Org(w http.ResponseWriter, r *http.Request) {
	installation, err := g.orgInstallation(w, r)
	if err != nil {
		return
	}

	idStr := strconv.FormatInt(installation.GetAccount().GetID(), 10)
	config, err := g.OrganizationOp.Config(r.Context(), github_provider.Provider, idStr)
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed read the organization config from the DB")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	successJSON(w, r, orgResponse{
		ID:     installation.GetAccount().GetID(),
		Name:   installation.GetAccount().GetLogin(),
		Config: config,
	})
}

type updateOrgReq struct {
	Config string `json:"config,omitempty"`
}

// UpdateOrg is a hander that updates the organization settings, and returns
// the updated organization information with the same response as Org
func (g *GitHub) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	var configRequest updateOrgReq
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to read the request body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &configRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request. Body is not a valid JSON: %s", err), http.StatusBadRequest)
		return
	}

	var empty struct{}
	if err := yaml.Unmarshal([]byte(configRequest.Config), &empty); err != nil {
		http.Error(w,
			fmt.Sprintf("Bad Request. The configuration is not valid YAML: %s", err),
			http.StatusBadRequest)
		return
	}

	installation, err := g.orgInstallation(w, r)
	if err != nil {
		return
	}

	idStr := strconv.FormatInt(installation.GetAccount().GetID(), 10)
	err = g.OrganizationOp.Save(r.Context(), github_provider.Provider, idStr, configRequest.Config)
	if err != nil {
		ctxlog.Get(r.Context()).Errorf(err, "failed to save the organization config")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	successJSON(w, r, orgResponse{
		ID:     installation.GetAccount().GetID(),
		Name:   installation.GetAccount().GetLogin(),
		Config: configRequest.Config,
	})
}
