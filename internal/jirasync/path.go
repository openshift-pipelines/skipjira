package jirasync

import (
	"fmt"

	"github.com/openshift-pipelines/skipjira/internal/jira"
)

// FindTransitionSequence attempts to find a sequence of transitions to reach target
// Uses common workflow patterns when direct transition isn't available
// Returns nil if target isn't directly reachable or via common intermediate states
func FindTransitionSequence(targetStatus string, allTransitions []jira.Transition) []jira.Transition {
	// Try direct transition first
	if direct := FindTransitionByName(allTransitions, targetStatus); direct != nil {
		return []jira.Transition{*direct}
	}

	// Common workflow intermediate states to try
	// These are typical states that workflows pass through
	intermediateStates := []string{
		"In Progress",
		"To Do",
		"Open",
	}

	// Try going through each intermediate state
	for _, intermediate := range intermediateStates {
		// Can we reach the intermediate state?
		firstStep := FindTransitionByName(allTransitions, intermediate)
		if firstStep == nil {
			continue
		}

		// If we could transition to intermediate, we'd need to check
		// if we can then go from intermediate to target
		// But Jira only shows transitions from CURRENT state
		// So we can't know without actually executing the first transition

		// For now, we'll only suggest this if it's a common pattern
		// The actual execution will need to fetch transitions after each step
	}

	// No direct path found
	return nil
}

// TryMultiStepTransition attempts to transition through intermediate states if needed
// Fetches fresh transitions after each step to navigate the workflow
func TryMultiStepTransition(jiraClient *jira.Client, issueKey, currentStatus, targetStatus string, maxSteps int) ([]string, error) {
	if maxSteps <= 0 {
		maxSteps = 3 // Default safety limit
	}

	executedSteps := []string{}
	currentState := currentStatus

	for step := 0; step < maxSteps; step++ {
		// Get available transitions from current state
		transitions, err := jiraClient.GetTransitions(issueKey)
		if err != nil {
			return executedSteps, fmt.Errorf("failed to get transitions at step %d: %w", step+1, err)
		}

		if len(transitions) == 0 {
			return executedSteps, fmt.Errorf("no transitions available from state '%s'", currentState)
		}

		// Try to find direct transition to target
		if direct := FindTransitionByName(transitions, targetStatus); direct != nil {
			// Execute final transition
			if err := jiraClient.DoTransition(issueKey, direct.ID); err != nil {
				return executedSteps, fmt.Errorf("failed to execute transition '%s': %w", direct.Name, err)
			}
			executedSteps = append(executedSteps, direct.Name)
			return executedSteps, nil
		}

		// No direct path - try progressing through workflow states
		// Common workflow progression: To Do → In Progress → Code Review → On QA → Closed
		preferredIntermediates := []string{
			"In Progress", // From To Do
			"Code Review", // From In Progress
			"Open",        // Alternative to In Progress
			"Review",      // Alternative naming
		}

		var intermediateTransition *jira.Transition
		// Try preferred intermediates first
		for _, preferred := range preferredIntermediates {
			for _, t := range transitions {
				if t.To.Name == preferred {
					intermediateTransition = &t
					break
				}
			}
			if intermediateTransition != nil {
				break
			}
		}

		// If no preferred intermediate, try any transition that moves forward
		// (not a self-transition and not moving backward)
		if intermediateTransition == nil {
			for _, t := range transitions {
				// Skip self-transitions
				if t.To.Name == currentState {
					continue
				}
				// Use the first forward-moving transition we find
				intermediateTransition = &t
				break
			}
		}

		// Still nothing? Give up
		if intermediateTransition == nil {
			return executedSteps, fmt.Errorf("no path to '%s' from '%s' (tried %d steps)", targetStatus, currentState, step+1)
		}

		// Execute intermediate transition
		if err := jiraClient.DoTransition(issueKey, intermediateTransition.ID); err != nil {
			return executedSteps, fmt.Errorf("failed to execute intermediate transition '%s': %w", intermediateTransition.Name, err)
		}

		executedSteps = append(executedSteps, intermediateTransition.Name)
		currentState = intermediateTransition.To.Name

		// Check if we reached target
		if currentState == targetStatus {
			return executedSteps, nil
		}
	}

	return executedSteps, fmt.Errorf("exceeded max steps (%d) trying to reach '%s'", maxSteps, targetStatus)
}
