package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
	"golang.org/x/oauth2"
)

// Client wraps GitHub API client
type Client struct {
	client *github.Client
	owner  string
	repo   string
}

// NewClient creates a new GitHub client
func NewClient(token, owner, repo string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		owner:  owner,
		repo:   repo,
	}
}

// GetPRForBranch finds the PR URL for a given branch
func (c *Client) GetPRForBranch(ctx context.Context, branchName string) (string, error) {
	// List PRs for the branch
	opts := &github.PullRequestListOptions{
		State: "all",
		Head:  fmt.Sprintf("%s:%s", c.owner, branchName),
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	prs, _, err := c.client.PullRequests.List(ctx, c.owner, c.repo, opts)
	if err != nil {
		return "", fmt.Errorf("failed to list PRs: %w", err)
	}

	if len(prs) == 0 {
		return "", fmt.Errorf("no PR found for branch %s", branchName)
	}

	// Return the first (most recent) PR
	return prs[0].GetHTMLURL(), nil
}
