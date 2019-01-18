package github

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
	log "gopkg.in/src-d/go-log.v1"
)

// TODO(@smacker) maybe we need to take into account commitID somewhere
// but on the first glace looks like we can just ignore it

// Github doesn't allow to post more than 32 comments in 1 review
// returning "was submitted too quickly"
// with 32 comments they got posted by GH return 502 Server Error
// issue: https://github.com/src-d/lookout/issues/264
// issue in go-github: https://github.com/google/go-github/issues/540
const batchReviewComments = 30

// comments posted on the same line will be merged using this separator
const commentsSeparator = "\n<!-- lookout comment separator -->\n---\n"

// comment can contain footer with link to the analyzer
const footnoteSeparator = "\n<!-- lookout footnote separator -->\n"

// createReview creates pull request review on github using multiple http calls
// in case of too many comments
func createReview(
	ctx context.Context,
	client *Client,
	owner, repo string, number int,
	req *github.PullRequestReviewRequest,
) error {
	requests := splitReviewRequest(req, batchReviewComments)
	for i, req := range requests {
		_, resp, err := client.PullRequests.CreateReview(ctx, owner, repo, number, req)
		if err = handleAPIError(resp, err); err != nil {
			return err
		}

		// need to wait between requests to avoid "was submitted too quickly" error
		if i < len(requests) {
			time.Sleep(time.Second)
		}
	}

	return nil
}

func filterPostedComments(comments []*github.DraftReviewComment, posted []*github.PullRequestComment) []*github.DraftReviewComment {
	var filtered []*github.DraftReviewComment

	for _, comment := range comments {
		var filterOut bool
		for _, pc := range posted {
			if comment.GetPath() != pc.GetPath() {
				continue
			}

			if comment.GetPosition() != pc.GetPosition() {
				continue
			}

			postedBody := removeFootnote(pc.GetBody())

			// posted comment may consist merged comments
			for _, body := range strings.Split(postedBody, commentsSeparator) {
				if comment.GetBody() == body {
					filterOut = true
					break
				}
			}

			if filterOut {
				break
			}
		}

		if !filterOut {
			filtered = append(filtered, comment)
		}
	}

	return filtered
}

func getPostedComment(ctx context.Context, client *Client, owner, repo string, number int) ([]*github.PullRequestComment, error) {
	var result []*github.PullRequestComment

	listReviewsOpts := &github.ListOptions{
		PerPage: 100,
	}
	var reviews []*github.PullRequestReview
	for {
		rs, resp, err := client.PullRequests.ListReviews(ctx, owner, repo, number, listReviewsOpts)
		if handleAPIError(resp, err) != nil {
			return nil, err
		}

		reviews = append(reviews, rs...)
		if resp.NextPage == 0 {
			break
		}

		listReviewsOpts.Page = resp.NextPage
	}

	for _, review := range reviews {
		listCommentsOpts := &github.ListOptions{
			PerPage: 100,
		}

		for {
			comments, resp, err := client.PullRequests.ListReviewComments(ctx, owner, repo, number, review.GetID(), listCommentsOpts)
			if handleAPIError(resp, err) != nil {
				return nil, err
			}

			result = append(result, comments...)
			if resp.NextPage == 0 {
				break
			}

			listCommentsOpts.Page = resp.NextPage
		}
	}

	return result, nil
}

func mergeComments(comments []*github.DraftReviewComment) []*github.DraftReviewComment {
	var mergedComments []*github.DraftReviewComment

	// sort by path and position
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].GetPath() < comments[j].GetPath() {
			return true
		}
		if comments[i].GetPath() > comments[j].GetPath() {
			return false
		}
		if comments[i].GetPosition() < comments[j].GetPosition() {
			return true
		}

		return false
	})

	var lastComment *github.DraftReviewComment
	for _, comment := range comments {
		if lastComment != nil &&
			lastComment.GetPath() == comment.GetPath() &&
			lastComment.GetPosition() == comment.GetPosition() {

			mergedBody := lastComment.GetBody() + commentsSeparator + comment.GetBody()
			lastComment.Body = &mergedBody
			continue
		}

		mergedComments = append(mergedComments, comment)
		lastComment = comment
	}

	return mergedComments
}

// splitReviewRequest transforms a review into a list of reviews with not more than N comments in each
func splitReviewRequest(review *github.PullRequestReviewRequest, n int) []*github.PullRequestReviewRequest {
	if len(review.Comments) <= n {
		return []*github.PullRequestReviewRequest{review}
	}

	var result []*github.PullRequestReviewRequest
	comments := review.Comments
	// set body only to the last review
	emptyBody := ""

	for len(comments) > n {
		result = append(result, &github.PullRequestReviewRequest{
			CommitID: review.CommitID,
			Event:    review.Event,
			Body:     &emptyBody,
			Comments: comments[:n],
		})

		comments = comments[n:]
	}

	if len(comments) > 0 {
		result = append(result, &github.PullRequestReviewRequest{
			CommitID: review.CommitID,
			Event:    review.Event,
			Body:     &emptyBody,
			Comments: comments,
		})
	}

	result[len(result)-1].Body = review.Body

	return result
}

// addFootnote adds footnote link to text of a comment
func addFootnote(text, tmpl, url string) string {
	if text == "" || tmpl == "" || url == "" {
		return text
	}

	return text + footnoteSeparator + fmt.Sprintf(tmpl, url)
}

// removeFootnote removes footnote and returns only text of a comment
func removeFootnote(text string) string {
	return strings.SplitN(text, footnoteSeparator, 2)[0]
}

// convertComments transforms []*lookout.Comment to []*github.DraftReviewComment and list of string for body
func convertComments(ctx context.Context, cs []*lookout.Comment, dl *diffLines) ([]string, []*github.DraftReviewComment) {
	var bodyComments []string
	var comments []*github.DraftReviewComment

	for _, c := range cs {
		if c.File == "" {
			bodyComments = append(bodyComments, c.Text)
			continue
		}

		if c.Line < 1 {
			line := 1
			comment := &github.DraftReviewComment{
				Path:     &c.File,
				Position: &line,
				Body:     &c.Text,
			}
			comments = append(comments, comment)
			continue
		}

		logger := convertLineLogger(ctx, c)
		line, err := dl.ConvertLine(c.File, int(c.Line), true)
		if ErrLineOutOfDiff.Is(err) {
			logger.Debugf("skipping comment out the diff range")
			continue
		}

		if ErrLineNotAddition.Is(err) {
			logger.Debugf("skipping comment not on an added line (+ in diff)")
			continue
		}

		if ErrFileNotFound.Is(err) {
			logger.Warningf("skipping comment on a file not part of the diff")
			continue
		}

		if ErrBadPatch.Is(err) {
			patch, _ := dl.filePatch(c.File)
			ctxlog.Get(ctx).With(log.Fields{
				"file":  c.File,
				"patch": patch,
			}).Errorf(err, "skipping comment because the diff could not be parsed")
			continue
		}

		if err != nil {
			convertLineLogger(ctx, c).Errorf(err, "skipping comment because of unknown error")
			continue
		}

		comment := &github.DraftReviewComment{
			Path:     &c.File,
			Position: &line,
			Body:     &c.Text,
		}

		comments = append(comments, comment)
	}

	return bodyComments, comments
}

func convertLineLogger(ctx context.Context, c *lookout.Comment) log.Logger {
	return ctxlog.Get(ctx).With(log.Fields{
		"file": c.File,
		"line": c.Line,
	})
}
