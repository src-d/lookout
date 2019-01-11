package web

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/src-d/lookout/util/ctxlog"
)

// GitHub is an HTTP service to call GitHub endpoints
type GitHub struct {
	AppID      int
	PrivateKey string
}

func (g *GitHub) installations(ctx context.Context) ([]*github.Installation, error) {
	appTr, err := ghinstallation.NewAppsTransportKeyFromFile(
		http.DefaultTransport, g.AppID, g.PrivateKey)
	if err != nil {
		return nil, err
	}

	appClient := github.NewClient(&http.Client{Transport: appTr})

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

// orgResponse is the response type used by Orgs handler
type orgRespose struct {
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
	orgs := []orgRespose{}

	for _, installation := range installations {
		// New transport for each installation
		itr, err := ghinstallation.NewKeyFromFile(
			http.DefaultTransport, g.AppID, int(installation.GetID()), g.PrivateKey)

		if err != nil {
			ctxlog.Get(r.Context()).Errorf(err, "failed to initialize the GitHub App installation client")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Use installation transport in a new client
		client := github.NewClient(&http.Client{Transport: itr})

		if installation.Account == nil || installation.Account.Login == nil {
			ctxlog.Get(r.Context()).Errorf(nil, "failed to get GitHub installation organization name")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// a GitHub App can also be installed in a User account
		if *installation.Account.Type != "Organization" {
			continue
		}

		org := *installation.Account.Login
		mem, _, err := client.Organizations.GetOrgMembership(r.Context(), login, org)
		if err != nil {
			ctxlog.Get(r.Context()).Errorf(err, "failed to get GitHub user %s membership to organization %s: ", login, org)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if *mem.Role == "admin" {
			orgs = append(orgs, orgRespose{Name: org})
		}
	}

	successJSON(w, r, orgs)
}
