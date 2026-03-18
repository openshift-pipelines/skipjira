package releasenotes

import (
	"regexp"
	"strings"
)

// ExtractFromDescription attempts to extract release notes from a PR description.
// It looks for "Release Notes" as either:
//   - Markdown heading: "### Release Notes" or "## Release Notes"
//   - Text with colon: "Release Notes:" or "**Release Notes:**"
//
// Extracts content until the next markdown heading or end of string.
// Handles code blocks (```) and removes them if the content is inside code fences.
// Returns the extracted notes and a boolean indicating if notes were found.
func ExtractFromDescription(description string) (string, bool) {
	if description == "" {
		return "", false
	}

	// Case-insensitive regex to find "Release Notes" section
	// Matches both:
	// - Markdown headings: "### Release Notes" or "## Release Notes"
	// - Text with colon: "Release Notes:" or "**Release Notes:**"
	// Captures everything after until the next markdown heading or end of string
	pattern := `(?i)(?:^|\n)\s*(?:#{2,}\s*release\s*notes|(?:\*\*)?release\s*notes(?:\*\*)?\s*:)\s*\n((?:.*\n?)*?)(?:\n#{2,}|$)`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(description)
	if len(matches) < 2 {
		return "", false
	}

	// Extract the content
	notes := strings.TrimSpace(matches[1])
	if notes == "" {
		return "", false
	}

	// Remove code fences if the entire content is wrapped in them
	notes = removeCodeFences(notes)

	return notes, true
}

// removeCodeFences removes surrounding code fences (```) from the text
func removeCodeFences(text string) string {
	// Pattern to match content inside code fences
	codeFencePattern := regexp.MustCompile(`(?s)^\s*` + "```" + `[^\n]*\n(.*?)\n` + "```" + `\s*$`)
	if matches := codeFencePattern.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return text
}
