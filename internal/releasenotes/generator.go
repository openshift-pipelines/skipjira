package releasenotes

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v81/github"
	"github.com/openshift-pipelines/skipjira/internal/gemini"
	ghclient "github.com/openshift-pipelines/skipjira/internal/github"
	"github.com/openshift-pipelines/skipjira/internal/prkind"
)

// Result contains release notes and metadata about how they were obtained
type Result struct {
	Notes       string
	IsGenerated bool   // true if AI-generated, false if extracted from PR
	Kind        string // "Bug Fix", "Feature", or "Enhancement"
}

// GetOrGenerate attempts to extract release notes from a PR description,
// and if not found, generates them using AI from PR changes.
// Also determines the PR kind (Bug Fix / Feature / Enhancement) from labels or AI.
// Returns the release notes result and an error if the process fails.
func GetOrGenerate(ctx context.Context, pr *github.PullRequest, ghClient *ghclient.Client, geminiClient *gemini.Client) (*Result, error) {
	if pr == nil {
		return nil, fmt.Errorf("PR is nil")
	}

	prDescription := pr.GetBody()

	// Determine PR kind from labels first
	labels := ghclient.GetPRLabels(pr)
	kind, found := prkind.DetermineKindFromLabels(labels)
	if !found {
		// Labels didn't provide kind, try AI if available
		if geminiClient != nil {
			// Build context for AI determination
			commits, err := ghClient.GetPRCommits(ctx, pr.GetNumber())
			if err == nil {
				commitMessages := make([]string, 0, len(commits))
				for _, commit := range commits {
					if commit.Commit != nil && commit.Commit.Message != nil {
						commitMessages = append(commitMessages, *commit.Commit.Message)
					}
				}

				prContext := prkind.PRKindContext{
					Title:          pr.GetTitle(),
					Description:    pr.GetBody(),
					CommitMessages: commitMessages,
				}

				aiKind, err := prkind.DetermineWithAI(ctx, prContext, geminiClient)
				if err != nil {
					// AI failed, use default
					fmt.Printf("  ⚠ Failed to determine PR kind via AI: %v, using default\\n", err)
					kind = prkind.KindEnhancement
				} else {
					kind = aiKind
				}
			} else {
				// Failed to get commits, use default
				kind = prkind.KindEnhancement
			}
		} else {
			// No Gemini client, use default
			kind = prkind.KindEnhancement
		}
	}

	// First, try to extract from description
	if notes, found := ExtractFromDescription(prDescription); found {
		return &Result{
			Notes:       notes,
			IsGenerated: false,
			Kind:        string(kind),
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
		Kind:        string(kind),
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
