package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"
)

// ListPRsSince lists pull requests updated since a given time
// Uses server-side sorting to fetch most recently updated PRs first
func (c *Client) ListPRsSince(ctx context.Context, state string, since time.Time) ([]*github.PullRequest, error) {
	var result []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:     state, // "open", "closed", "all"
		Sort:      "updated",
		Direction: "desc", // Most recently updated first
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := c.client.PullRequests.List(ctx, c.owner, c.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list PRs: %w", err)
		}

		// Add PRs that are newer than the cutoff
		for _, pr := range prs {
			if pr.UpdatedAt != nil && pr.UpdatedAt.After(since) {
				result = append(result, pr)
			} else {
				// Since we're sorted by updated desc, once we hit an old PR,
				// all remaining PRs will be older - stop pagination
				return result, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}
