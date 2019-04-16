package github

// github returns errors in different format for review API
// this file "fixes" it by transforming response on transport level
// it allows handle such errors correctly in client code and log them
// issue in go-github: https://github.com/google/go-github/issues/540

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/google/go-github/v24/github"
)

var reviewPathRegexp = regexp.MustCompile("/repos/.+/.+/pulls/[0-9]+/review")

// fixReviewTransport converts error response from review api endpoint
// to the format known by github library
type fixReviewTransport struct {
	Transport http.RoundTripper
}

// reviewErrorResp defines structure of review error response
type reviewErrorResp struct {
	Message          string
	Errors           []string
	DocumentationURL string
}

// ToGithubErrorResponse converts review errors response to github.ErrorResponse
func (r *reviewErrorResp) ToGithubErrorResponse() github.ErrorResponse {
	errors := make([]github.Error, len(r.Errors))
	for i, errorStr := range r.Errors {
		code := "custom"

		if errorStr == "User can only have one pending review per pull request" {
			code = "already_exists"
		}

		errors[i] = github.Error{
			Resource: "Review",
			Code:     code,
			Message:  errorStr,
		}
	}

	return github.ErrorResponse{
		Message:          r.Message,
		Errors:           errors,
		DocumentationURL: r.DocumentationURL,
	}
}

// RoundTrip implemements http.RoundTripper interface
func (t *fixReviewTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	baseT := t.Transport
	if baseT == nil {
		baseT = http.DefaultTransport
	}

	resp, err := baseT.RoundTrip(req)

	// match only errors for review path
	if err == nil && resp.StatusCode == http.StatusUnprocessableEntity && reviewPathRegexp.MatchString(req.URL.Path) {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("can't read body of review error response: %s", err)
		}

		var errorResp reviewErrorResp
		err = json.Unmarshal(b, &errorResp)
		// if we aren't able to parse response - pass original one
		// in case github fixed the wrong format or started returned something new
		if err != nil || errorResp.Message == "" || len(errorResp.Errors) == 0 {
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			return resp, nil
		}

		newB, err := json.Marshal(errorResp.ToGithubErrorResponse())
		if err != nil {
			// we marshal known struct so it should never happen in theory
			return nil, err
		}

		resp.Body = ioutil.NopCloser(bytes.NewBuffer(newB))
	}

	return resp, err
}
