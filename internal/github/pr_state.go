package github

import (
	"context"

	"github.com/google/go-github/v81/github"
)

// PRState represents the state of a pull request
type PRState string

const (
	PRStateDraft            PRState = "draft"
	PRStateChangesRequested PRState = "changes_requested"
	PRStateOpen             PRState = "ready"
	PRStateReviewRequested  PRState = "review_requested"
	PRStateApproved         PRState = "approved"
	PRStateMerged           PRState = "merged"
	PRStateClosed           PRState = "closed"
)

// Priority returns the workflow priority of a PR state.
// Lower priority = earlier in workflow (more behind).
// Used when multiple PRs link to one Jira ticket to determine which state to use.
func (s PRState) Priority() int {
	switch s {
	case PRStateDraft, PRStateChangesRequested, PRStateClosed:
		return 1 // Most behind - needs work (draft, changes needed, or abandoned)
	case PRStateOpen, PRStateReviewRequested, PRStateApproved:
		return 2 // In review
	case PRStateMerged:
		return 3 // Merged - ready for QA
	default:
		return 999 // Unknown
	}
}

// GetPRState determines the state of a pull request
// Following the logic from ospctl/test/main.go
func (c *Client) GetPRState(ctx context.Context, pr *github.PullRequest) (PRState, error) {
	// Check if merged first - MergedAt is more reliable than Merged field
	// When listing PRs, the Merged boolean field is sometimes not populated,
	// but MergedAt timestamp is always set for merged PRs
	if pr.MergedAt != nil && !pr.MergedAt.IsZero() {
		return PRStateMerged, nil
	}

	// Check if closed without merge
	// If we reach here, MergedAt is nil/zero, so this is truly a closed (not merged) PR
	if pr.GetState() == "closed" {
		return PRStateClosed, nil
	}

	// Check if draft
	if pr.GetDraft() {
		return PRStateDraft, nil
	}

	// Default: ready for review
	return PRStateOpen, nil
}
