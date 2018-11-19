package github

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"
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

// createReview creates pull request review
// it handles all no successful cases like:
// - limit on number of comments in review
// - merges comments that are made on the same line to reduce number of comments
// - checks that comments weren't posted already
func createReview(
	ctx context.Context,
	client *Client,
	owner, repo string, number int,
	req *github.PullRequestReviewRequest,
	onlyNewComments bool,
) error {
	if onlyNewComments {
		postedComments, err := getPostedComment(ctx, client, owner, repo, number)
		if err != nil {
			return err
		}
		req.Comments = filterPostedComments(req.Comments, postedComments)
	}

	req.Comments = mergeComments(req.Comments)

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

			// posted comment may consist merged comments
			for _, body := range strings.Split(pc.GetBody(), commentsSeparator) {
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
			comments, resp, err := client.PullRequests.ListReviewComments(ctx, owner, repo, int64(number), review.GetID(), listCommentsOpts)
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
