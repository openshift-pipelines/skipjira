package prkind

import (
	"strings"

	"github.com/google/go-github/v81/github"
)

// DetermineKindFromLabels determines PR kind from GitHub labels
// Returns (kind, found) where found indicates if a matching label was detected
// Priority order: Bug > Feature > Enhancement
func DetermineKindFromLabels(labels []*github.Label) (PRKind, bool) {
	if labels == nil || len(labels) == 0 {
		return "", false
	}

	labelNames := extractLabelNames(labels)

	// Check for bug labels (highest priority)
	bugPatterns := []string{"bug", "bugfix", "fix", "hotfix", "bug-fix", "hot-fix"}
	if matchesAny(labelNames, bugPatterns) {
		return KindBugFix, true
	}

	// Check for feature labels
	featurePatterns := []string{"feature", "new-feature", "new feature"}
	if matchesAny(labelNames, featurePatterns) {
		return KindFeature, true
	}

	// Check for enhancement labels
	enhancementPatterns := []string{"enhancement", "improvement"}
	if matchesAny(labelNames, enhancementPatterns) {
		return KindEnhancement, true
	}

	return "", false
}

// extractLabelNames converts label objects to lowercase names
func extractLabelNames(labels []*github.Label) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != nil && label.Name != nil {
			names = append(names, strings.ToLower(*label.Name))
		}
	}
	return names
}

// matchesAny checks if any label name matches any pattern
func matchesAny(labelNames []string, patterns []string) bool {
	for _, labelName := range labelNames {
		for _, pattern := range patterns {
			if strings.Contains(labelName, pattern) {
				return true
			}
		}
	}
	return false
}
