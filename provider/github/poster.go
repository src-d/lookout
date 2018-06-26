package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/src-d/lookout"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-errors.v1"
)

var (
	// ErrGitHubAPI signals an error while making a request to the GitHub API.
	ErrGitHubAPI = errors.NewKind("github api error")
	// ErrEventNotSupported signals that this provider does not support the
	// given event for a given operation.
	ErrEventNotSupported = errors.NewKind("event not supported")
)

// Poster posts comments as Pull Request Reviews.
type Poster struct {
	rc ReviewCreator
	cc CommitsComparator
}

var _ lookout.Poster = &Poster{}

// NewPoster creates a new poster given a GitHub API client.
func NewPoster(rc ReviewCreator, cc CommitsComparator) *Poster {
	return &Poster{rc: rc, cc: cc}
}

// Post posts comments as a Pull Request Review.
// If the event is not a GitHub Pull Request, ErrEventNotSupported is returned.
// If a GitHub API request fails, ErrGitHubAPI is returned.
func (p *Poster) Post(ctx context.Context, e lookout.Event,
	cs []*lookout.Comment) error {
	switch ev := e.(type) {
	case *lookout.PullRequestEvent:
		if ev.Provider != Provider {
			return ErrEventNotSupported.Wrap(
				fmt.Errorf("unsupported provider: %s", ev.Provider))
		}

		return p.postPR(ctx, ev, cs)
	default:
		return ErrEventNotSupported.Wrap(fmt.Errorf("unsupported event type"))
	}
}

func (p *Poster) postPR(ctx context.Context, e *lookout.PullRequestEvent,
	cs []*lookout.Comment) error {

	owner, repo, pr, err := p.validatePR(e)
	if err != nil {
		return err
	}

	// TODO: make this request lazily, only if there are comments using
	// positions.
	cc, resp, err := p.cc.CompareCommits(ctx, owner, repo,
		e.Base.Hash,
		e.Head.Hash)
	if err = p.handleAPIError(resp, err); err != nil {
		return err
	}

	dl := newDiffLines(cc)
	review, err := createReviewRequest(cs, dl)
	if err != nil {
		return err
	}

	_, resp, err = p.rc.CreateReview(ctx, owner, repo, pr, review)
	if err = p.handleAPIError(resp, err); err != nil {
		return err
	}

	return nil
}

func (p *Poster) validatePR(
	e *lookout.PullRequestEvent) (owner, repo string, pr int, err error) {

	base := e.Base
	owner, err = extractOwner(base)
	if err != nil {
		err = ErrEventNotSupported.Wrap(err)
		return
	}

	repo, err = extractRepo(base)
	if err != nil {
		err = ErrEventNotSupported.Wrap(err)
		return
	}

	pr, err = strconv.Atoi(e.InternalID)
	if err != nil {
		err = ErrEventNotSupported.Wrap(
			fmt.Errorf("bad PR ID: %s", e.InternalID))
		return
	}

	return
}

func (p *Poster) handleAPIError(resp *github.Response, err error) error {
	if err != nil {
		return ErrGitHubAPI.Wrap(err)
	}

	if resp.StatusCode == 200 {
		return nil
	}

	return ErrGitHubAPI.Wrap(fmt.Errorf("bad HTTP status: %d", resp.StatusCode))
}

// ReviewCreator can create code reviews on GitHub. *github.PullRequestsService
// fulfills this interface.
type ReviewCreator interface {
	// CreateReview creates a new code review on a GitHub pull request.
	CreateReview(ctx context.Context, owner, repo string,
		number int, review *github.PullRequestReviewRequest) (
		*github.PullRequestReview, *github.Response, error)
}

var _ ReviewCreator = &github.PullRequestsService{}

// CommitsComparator compares commits on GitHub. *github.RepositoriesService
// fulfills this interface.
type CommitsComparator interface {
	// CompareCommits compare two commits.
	CompareCommits(ctx context.Context, owner, repo string, base, head string) (
		*github.CommitsComparison, *github.Response, error)
}

var _ CommitsComparator = &github.RepositoriesService{}

var approveEvent = "APPROVE"

func createReviewRequest(
	cs []*lookout.Comment,
	dl *diffLines) (*github.PullRequestReviewRequest, error) {
	req := &github.PullRequestReviewRequest{
		Event: &approveEvent,
	}

	var bodyComments []string

	for _, c := range cs {
		if c.File == "" {
			bodyComments = append(bodyComments, c.Text)
		} else if c.Line < 1 {
			line := 1
			comment := &github.DraftReviewComment{
				Path:     &c.File,
				Position: &line,
				Body:     &c.Text,
			}
			req.Comments = append(req.Comments, comment)
		} else {
			line, err := dl.ConvertLine(c.File, int(c.Line))
			if err != nil {
				return nil, err
			}

			comment := &github.DraftReviewComment{
				Path:     &c.File,
				Position: &line,
				Body:     &c.Text,
			}
			req.Comments = append(req.Comments, comment)
		}
	}

	if len(bodyComments) > 0 {
		body := strings.Join(bodyComments, "\n\n")
		req.Body = &body
	}

	return req, nil
}
