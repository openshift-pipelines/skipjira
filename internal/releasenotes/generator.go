package releasenotes

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v81/github"
	"github.com/theakshaypant/skipjira/internal/gemini"
	ghclient "github.com/theakshaypant/skipjira/internal/github"
)

// Result contains release notes and metadata about how they were obtained
type Result struct {
	Notes       string
	IsGenerated bool // true if AI-generated, false if extracted from PR
}

// GetOrGenerate attempts to extract release notes from a PR description,
// and if not found, generates them using AI from PR changes.
// Returns the release notes result and an error if the process fails.
func GetOrGenerate(ctx context.Context, pr *github.PullRequest, ghClient *ghclient.Client, geminiClient *gemini.Client) (*Result, error) {
	if pr == nil {
		return nil, fmt.Errorf("PR is nil")
	}

	prDescription := pr.GetBody()

	// First, try to extract from description
	if notes, found := ExtractFromDescription(prDescription); found {
		return &Result{
			Notes:       notes,
			IsGenerated: false,
		}, nil
	}

	// If not found, generate using AI
	generatedNotes, err := generateWithAI(ctx, pr, ghClient, geminiClient)
	if err != nil {
		return nil, err
	}

	return &Result{
		Notes:       generatedNotes,
		IsGenerated: true,
	}, nil
}

// generateWithAI generates release notes using Gemini AI
func generateWithAI(ctx context.Context, pr *github.PullRequest, ghClient *ghclient.Client, geminiClient *gemini.Client) (string, error) {
	prNumber := pr.GetNumber()

	// Fetch PR diff
	diff, err := ghClient.GetPRDiff(ctx, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	// Fetch PR commits
	commits, err := ghClient.GetPRCommits(ctx, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PR commits: %w", err)
	}

	// Extract commit messages
	commitMessages := make([]string, 0, len(commits))
	for _, commit := range commits {
		if commit.Commit != nil && commit.Commit.Message != nil {
			commitMessages = append(commitMessages, *commit.Commit.Message)
		}
	}

	// Build PR context
	prContext := gemini.PRContext{
		Title:          pr.GetTitle(),
		Description:    pr.GetBody(),
		Diff:           diff,
		CommitMessages: commitMessages,
	}

	// Generate release note using Gemini
	releaseNote, err := geminiClient.GenerateReleaseNote(ctx, prContext)
	if err != nil {
		return "", fmt.Errorf("failed to generate release note: %w", err)
	}

	return strings.TrimSpace(releaseNote), nil
}
