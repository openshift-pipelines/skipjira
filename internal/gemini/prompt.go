package gemini

import (
	"fmt"
	"strings"
)

// buildReleaseNotePrompt constructs the prompt for Gemini to generate a release note
// Based on the reference implementation in gemini.py
func buildReleaseNotePrompt(ctx PRContext) string {
	parts := []string{
		"You are a technical writer creating release notes for software.",
		"Generate a concise, user-facing release note based on the following PR information.",
		"",
		"Guidelines:",
		"- Write 1-3 sentences maximum",
		"- Focus on what changed from the user's perspective, not implementation details",
		"- Use present tense (e.g., 'Adds support for...' not 'Added support for...')",
		"- If it's a bug fix, describe what is now fixed",
		"- If it's a feature, describe the new capability",
		"- Do not include PR numbers, commit hashes, or internal references",
		"- Do not use marketing language or superlatives",
		"",
		fmt.Sprintf("PR Title: %s", ctx.Title),
	}

	if ctx.Description != "" {
		// Limit description to 2000 chars
		desc := ctx.Description
		if len(desc) > 2000 {
			desc = desc[:2000] + "..."
		}
		parts = append(parts, fmt.Sprintf("\nPR Description:\n%s", desc))
	}

	if ctx.Diff != "" {
		// Limit diff to 3000 chars
		diff := ctx.Diff
		if len(diff) > 3000 {
			diff = diff[:3000] + "..."
		}
		parts = append(parts, fmt.Sprintf("\nPR Changes (diff):\n%s", diff))
	}

	if len(ctx.CommitMessages) > 0 {
		commitText := strings.Join(ctx.CommitMessages, "\n")
		parts = append(parts, fmt.Sprintf("\nCommit Messages:\n%s", commitText))
	}

	parts = append(parts, "\nGenerate the release note:")

	return strings.Join(parts, "\n")
}
