package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetRoot returns the git repository root directory
func GetRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the remote URL for origin
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExtractJiraIDs extracts Jira ticket IDs from a branch name
// Matches pattern: [A-Z]+-\d+ (e.g., TEKTON-123, PROJ-456)
func ExtractJiraIDs(branchName string) []string {
	re := regexp.MustCompile(`([A-Z]+-\d+)`)
	matches := re.FindAllString(branchName, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}

	return result
}

// ParseRepoFromURL extracts owner and repo name from GitHub URL
// Supports both HTTPS and SSH formats
func ParseRepoFromURL(remoteURL string) (owner, repo string, err error) {
	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		parts := strings.TrimPrefix(remoteURL, "git@github.com:")
		parts = strings.TrimSuffix(parts, ".git")
		repoParts := strings.Split(parts, "/")
		if len(repoParts) == 2 {
			return repoParts[0], repoParts[1], nil
		}
	}

	// HTTPS format: https://github.com/owner/repo.git
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		parts := strings.TrimPrefix(remoteURL, "https://github.com/")
		parts = strings.TrimSuffix(parts, ".git")
		repoParts := strings.Split(parts, "/")
		if len(repoParts) == 2 {
			return repoParts[0], repoParts[1], nil
		}
	}

	return "", "", fmt.Errorf("failed to parse GitHub repo from URL: %s", remoteURL)
}

// IsNewBranch checks if a branch exists on the remote
func IsNewBranch(branchName string) (bool, error) {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check remote branch: %w", err)
	}

	// If output is empty, branch doesn't exist on remote
	return out.Len() == 0, nil
}
