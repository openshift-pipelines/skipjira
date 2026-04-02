package jirasync

import (
	"strings"

	"github.com/openshift-pipelines/skipjira/internal/github"
	"github.com/openshift-pipelines/skipjira/internal/jira"
)

// PRStateToJiraStatus maps PR states to desired Jira status
// Note: We never automatically close tickets - "On QA" is the furthest we go
func PRStateToJiraStatus(prState github.PRState) string {
	switch prState {
	case github.PRStateDraft, github.PRStateChangesRequested:
		return "In Progress"
	case github.PRStateOpen, github.PRStateReviewRequested:
		return "Code Review"
	case github.PRStateApproved:
		return "Code Review"
	case github.PRStateMerged:
		return "On QA"
	case github.PRStateClosed:
		// Closed without merge goes back to In Progress (work abandoned/needs redo)
		return "In Progress"
	default:
		return ""
	}
}

// FindTransitionByName finds a transition that matches the target status name
// Uses case-insensitive matching to handle variations in transition names
func FindTransitionByName(transitions []jira.Transition, targetStatus string) *jira.Transition {
	if targetStatus == "" {
		return nil
	}

	// First try exact match (case-insensitive)
	for i := range transitions {
		if strings.EqualFold(transitions[i].To.Name, targetStatus) {
			return &transitions[i]
		}
	}

	// Try matching the transition name itself (some workflows use "Move to X" format)
	targetLower := strings.ToLower(targetStatus)
	for i := range transitions {
		transitionNameLower := strings.ToLower(transitions[i].Name)
		// Match if transition name contains the target status
		// e.g., "Move to Code Review" contains "code review"
		if strings.Contains(transitionNameLower, targetLower) {
			return &transitions[i]
		}
	}

	return nil
}
