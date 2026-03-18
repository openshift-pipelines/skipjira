package prkind

import (
	"fmt"
	"strings"
)

// buildPRKindPrompt builds the prompt for Gemini to determine PR kind
func buildPRKindPrompt(ctx PRKindContext) string {
	parts := []string{
		"You are analyzing a pull request to determine its type.",
		"",
		"Classify this PR as one of:",
		"- \"Bug Fix\": Fixes a defect, error, or incorrect behavior",
		"- \"Feature\": Introduces new functionality or capability",
		"- \"Enhancement\": Improves existing functionality (performance, UX, refactoring)",
		"",
		"Guidelines:",
		"- Analyze the PR title, description, and commit messages",
		"- If commits mention \"fix\", \"bug\", \"error\", \"crash\" → likely Bug Fix",
		"- If commits mention \"add\", \"new\", \"implement\", \"introduce\" → likely Feature",
		"- If commits mention \"improve\", \"optimize\", \"refactor\", \"update\" → likely Enhancement",
		"- When unclear, default to \"Enhancement\"",
		"- Return ONLY one of: \"Bug Fix\", \"Feature\", or \"Enhancement\"",
		"",
	}

	// Add PR title
	parts = append(parts, fmt.Sprintf("PR Title: %s", ctx.Title))
	parts = append(parts, "")

	// Add PR description (truncated to 1500 chars)
	if ctx.Description != "" {
		description := ctx.Description
		if len(description) > 1500 {
			description = description[:1500] + "\n... (truncated)"
		}
		parts = append(parts, "PR Description:")
		parts = append(parts, description)
		parts = append(parts, "")
	}

	// Add commit messages (truncated to 1000 chars total)
	if len(ctx.CommitMessages) > 0 {
		commitText := strings.Join(ctx.CommitMessages, "\n")
		if len(commitText) > 1000 {
			commitText = commitText[:1000] + "\n... (truncated)"
		}
		parts = append(parts, "Commit Messages:")
		parts = append(parts, commitText)
		parts = append(parts, "")
	}

	// Final instruction
	parts = append(parts, "Classify this PR as (respond with exactly one of: \"Bug Fix\", \"Feature\", or \"Enhancement\"):")

	return strings.Join(parts, "\n")
}
