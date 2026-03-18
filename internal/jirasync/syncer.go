package jirasync

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-pipelines/skipjira/internal/github"
	"github.com/openshift-pipelines/skipjira/internal/jira"
	"github.com/openshift-pipelines/skipjira/internal/slack"
	gogithub "github.com/google/go-github/v81/github"
	"github.com/openshift-pipelines/skipjira/internal/gemini"
	"github.com/openshift-pipelines/skipjira/internal/releasenotes"
)

// Syncer coordinates PR to Jira ticket synchronization
type Syncer struct {
	githubToken           string
	jiraURL               string
	jiraEmail             string
	jiraToken             string
	jiraPRField           string
	jiraReleaseNotesField string
	jiraClient            *jira.Client
	geminiClient          *gemini.Client
	sinceTime             time.Time
}

// SyncResult contains the results of syncing a repository
type SyncResult struct {
	Repository          Repository
	PRsProcessed        int
	TicketsTransitioned int
	Errors              []error
}

// SyncSummary contains overall sync results including Slack notification data
type SyncSummary struct {
	Results                []SyncResult
	Notifications          []slack.TransitionNotification
	TicketsInCorrectStatus int
	UnlinkedPRs            int
}

// NewSyncer creates a new syncer instance
func NewSyncer(githubToken, jiraURL, jiraEmail, jiraToken, jiraPRField, jiraReleaseNotesField, geminiAPIKey, geminiModel string, sinceTime time.Time) (*Syncer, error) {
	jiraClient, err := jira.NewClient(jiraURL, jiraEmail, jiraToken, jiraPRField, jiraReleaseNotesField)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	var geminiClient *gemini.Client
	if geminiAPIKey != "" {
		geminiClient, err = gemini.NewClient(geminiAPIKey, geminiModel)
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
	}

	return &Syncer{
		githubToken:           githubToken,
		jiraURL:               jiraURL,
		jiraEmail:             jiraEmail,
		jiraToken:             jiraToken,
		jiraPRField:           jiraPRField,
		jiraReleaseNotesField: jiraReleaseNotesField,
		jiraClient:            jiraClient,
		geminiClient:          geminiClient,
		sinceTime:             sinceTime,
	}, nil
}

// SyncAll syncs all repositories
// Batches PRs across all repos to avoid multiple transitions for tickets with cross-repo PRs
func (s *Syncer) SyncAll(ctx context.Context, repositories []Repository) (*SyncSummary, error) {
	fmt.Printf("\nStarting jirasync for %d repositories\n\n", len(repositories))

	// Phase 1: Collect all PRs from all repositories
	type repoPRInfo struct {
		repo   Repository
		number int
		title  string
		url    string
		state  github.PRState
	}

	// Global mapping: ticketKey → list of PRs from any repo
	globalTicketPRs := make(map[string][]repoPRInfo)
	// Track tickets with full metadata for Slack notifications
	globalTicketInfo := make(map[string]struct {
		Status  string
		Summary string
	})
	results := make([]SyncResult, len(repositories))
	unlinkedPRs := 0

	for repoIdx, repo := range repositories {
		results[repoIdx].Repository = repo
		fmt.Printf("=== Collecting PRs from %s/%s ===\n", repo.Owner, repo.Name)

		ghClient := github.NewClient(s.githubToken, repo.Owner, repo.Name)

		// List PRs for this repo
		prs, err := ghClient.ListPRsSince(ctx, "all", s.sinceTime)
		if err != nil {
			fmt.Printf("  ✗ Failed to list PRs: %v\n", err)
			results[repoIdx].Errors = append(results[repoIdx].Errors, fmt.Errorf("failed to list PRs: %w", err))
			continue
		}

		fmt.Printf("  Found %d PRs updated since %s\n", len(prs), s.sinceTime.Format("2006-01-02"))

		// Process each PR
		for _, pr := range prs {
			prURL := pr.GetHTMLURL()

			// Get PR state
			prState, err := ghClient.GetPRState(ctx, pr)
			if err != nil {
				results[repoIdx].Errors = append(results[repoIdx].Errors, fmt.Errorf("PR #%d: failed to get state: %w", pr.GetNumber(), err))
				continue
			}

			// Check if this maps to a Jira status
			targetStatus := PRStateToJiraStatus(prState)
			if targetStatus == "" {
				results[repoIdx].PRsProcessed++
				continue
			}

			// Find Jira tickets linked to this PR
			jql := fmt.Sprintf(`"%s" ~ "%s"`, s.jiraPRField, prURL)
			issues, err := s.jiraClient.SearchIssuesWithStatusByJQL(jql)
			if err != nil {
				results[repoIdx].Errors = append(results[repoIdx].Errors, fmt.Errorf("PR #%d: JQL search failed: %w", pr.GetNumber(), err))
				continue
			}

			if len(issues) == 0 {
				unlinkedPRs++
				results[repoIdx].PRsProcessed++
				continue
			}

			// Add release notes to Jira tickets (if Gemini client is configured)
			if s.geminiClient != nil {
				s.addReleaseNotes(ctx, pr, issues, ghClient)
			}

			// Add this PR to global ticket mapping and store ticket metadata
			for _, issue := range issues {
				globalTicketPRs[issue.Key] = append(globalTicketPRs[issue.Key], repoPRInfo{
					repo:   repo,
					number: pr.GetNumber(),
					title:  pr.GetTitle(),
					url:    prURL,
					state:  prState,
				})

				// Store ticket metadata (only once)
				if _, exists := globalTicketInfo[issue.Key]; !exists {
					status := "Unknown"
					if statusField, ok := issue.Fields["status"].(map[string]interface{}); ok {
						if name, ok := statusField["name"].(string); ok {
							status = name
						}
					}

					summary := ""
					if summaryField, ok := issue.Fields["summary"].(string); ok {
						summary = summaryField
					}

					globalTicketInfo[issue.Key] = struct {
						Status  string
						Summary string
					}{Status: status, Summary: summary}
				}
			}

			results[repoIdx].PRsProcessed++
		}
	}

	fmt.Printf("\n=== Processing Tickets (Global Batching) ===\n")
	fmt.Printf("Found %d unique tickets across all repositories\n\n", len(globalTicketPRs))

	// Phase 2: Process each ticket once, using the most behind PR from any repo
	totalTransitioned := 0
	ticketsInCorrectStatus := 0
	var notifications []slack.TransitionNotification

	for issueKey, prs := range globalTicketPRs {
		info := globalTicketInfo[issueKey]

		// Skip tickets in terminal states
		if info.Status == "Closed" || info.Status == "Done" {
			fmt.Printf("  ⊗ %s: Already in terminal state '%s' - skipping\n", issueKey, info.Status)
			continue
		}
		// Find the most behind PR across all repos
		mostBehind := prs[0]
		for _, pr := range prs[1:] {
			if pr.state.Priority() < mostBehind.state.Priority() {
				mostBehind = pr
			}
		}

		targetStatus := PRStateToJiraStatus(mostBehind.state)

		// Log which repos have PRs for this ticket
		if len(prs) > 1 {
			repoSet := make(map[string]bool)
			for _, pr := range prs {
				repoSet[fmt.Sprintf("%s/%s", pr.repo.Owner, pr.repo.Name)] = true
			}
			fmt.Printf("  %s: %d PRs across %d repo(s), using %s/%s PR #%d (%s)\n",
				issueKey, len(prs), len(repoSet),
				mostBehind.repo.Owner, mostBehind.repo.Name,
				mostBehind.number, mostBehind.state)
		}

		// Transition the ticket
		transitioned, transitionPath, err := s.syncTicket(ctx, issueKey, targetStatus, mostBehind.url, string(mostBehind.state))
		if err != nil {
			// Add error to the repo that "owns" this PR
			for i := range results {
				if results[i].Repository.Owner == mostBehind.repo.Owner &&
					results[i].Repository.Name == mostBehind.repo.Name {
					results[i].Errors = append(results[i].Errors, fmt.Errorf("ticket %s: %w", issueKey, err))
					break
				}
			}
			continue
		}

		// Track if already in correct status
		if !transitioned && transitionPath == "" {
			// Check if it's because we're already in target status
			currentStatus, _ := s.jiraClient.GetIssueStatus(issueKey)
			if currentStatus == targetStatus {
				ticketsInCorrectStatus++
			}
		}

		if transitioned {
			totalTransitioned++
			// Credit the transition to the repo with the most behind PR
			for i := range results {
				if results[i].Repository.Owner == mostBehind.repo.Owner &&
					results[i].Repository.Name == mostBehind.repo.Name {
					results[i].TicketsTransitioned++
					break
				}
			}

			// Collect Slack notification for transitioned tickets
			slackPRs := make([]slack.PRInfo, len(prs))
			for i, pr := range prs {
				slackPRs[i] = slack.PRInfo{
					Number: pr.number,
					Title:  pr.title,
					URL:    pr.url,
					State:  string(pr.state),
					Repo:   fmt.Sprintf("%s/%s", pr.repo.Owner, pr.repo.Name),
				}
			}

			notifications = append(notifications, slack.TransitionNotification{
				IssueKey:       issueKey,
				IssueSummary:   info.Summary,
				IssueURL:       fmt.Sprintf("%s/browse/%s", s.jiraURL, issueKey),
				CurrentStatus:  info.Status,
				TargetStatus:   targetStatus,
				TransitionPath: transitionPath,
				PRs:            slackPRs,
			})
		}
	}

	fmt.Printf("\nJirasync completed - %d tickets transitioned\n", totalTransitioned)

	return &SyncSummary{
		Results:                results,
		Notifications:          notifications,
		TicketsInCorrectStatus: ticketsInCorrectStatus,
		UnlinkedPRs:            unlinkedPRs,
	}, nil
}

// syncTicket attempts to transition a single Jira ticket
// Supports multi-step transitions when direct transition isn't available
// Returns: (transitioned bool, transitionPath string, error)
func (s *Syncer) syncTicket(_ context.Context, issueKey, targetStatus, _, prState string) (bool, string, error) {
	// Get current status first
	currentStatus, err := s.jiraClient.GetIssueStatus(issueKey)
	if err != nil {
		return false, "", fmt.Errorf("failed to get current status: %w", err)
	}

	// Already in target state?
	if currentStatus == targetStatus {
		return false, "", nil
	}

	// Get available transitions
	transitions, err := s.jiraClient.GetTransitions(issueKey)
	if err != nil {
		return false, "", fmt.Errorf("failed to get transitions: %w", err)
	}

	if len(transitions) == 0 {
		// No transitions available
		return false, "", nil
	}

	// Try direct transition first
	if direct := FindTransitionByName(transitions, targetStatus); direct != nil {
		if err := s.jiraClient.DoTransition(issueKey, direct.ID); err != nil {
			return false, "", fmt.Errorf("failed to execute transition: %w", err)
		}
		fmt.Printf("  ✓ Transitioned %s: '%s' → '%s' (PR state: %s)\n",
			issueKey, currentStatus, targetStatus, prState)
		return true, "direct", nil
	}

	// No direct transition - try multi-step (max 3 steps)
	steps, err := TryMultiStepTransition(s.jiraClient, issueKey, currentStatus, targetStatus, 3)
	if err != nil {
		// Log warning but don't fail - ticket just stays in current state
		fmt.Printf("  ⚠ %s: Could not transition from '%s' to '%s': %v\n",
			issueKey, currentStatus, targetStatus, err)
		return false, "", nil
	}

	// Success!
	fmt.Printf("  ✓ Transitioned %s: '%s' → '%s' in %d steps (PR state: %s)\n",
		issueKey, currentStatus, targetStatus, len(steps), prState)
	if len(steps) > 1 {
		fmt.Printf("    Path: %s\n", formatSteps(steps))
	}

	// Build transition path from steps
	transitionPath := formatSteps(steps)
	return true, transitionPath, nil
}

// formatSteps formats a list of transition names
func formatSteps(steps []string) string {
	result := ""
	for i, step := range steps {
		if i > 0 {
			result += " → "
		}
		result += fmt.Sprintf("'%s'", step)
	}
	return result
}

// addReleaseNotes extracts or generates release notes and adds them to Jira tickets
// If extracted from PR description: updates Jira release notes fields directly
// If AI-generated: adds a highlighted comment with assignee mention
func (s *Syncer) addReleaseNotes(ctx context.Context, pr *gogithub.PullRequest, issues []jira.SearchIssue, ghClient *github.Client) {
	// Extract or generate release notes
	result, err := releasenotes.GetOrGenerate(ctx, pr, ghClient, s.geminiClient)
	if err != nil {
		fmt.Printf("  ⚠ PR #%d: Failed to get release notes: %v\n", pr.GetNumber(), err)
		return
	}

	// Process each linked Jira ticket
	releaseNotesType := "Enhancement"
	releaseNotesStatus := "proposed"

	for _, issue := range issues {
		if result.IsGenerated {
			// AI-generated: Add orange warning panel with assignee mention
			// Get assignee account ID
			fullIssue, err := s.jiraClient.GetIssueWithFields(issue.Key, []string{"assignee"})
			if err != nil {
				fmt.Printf("  ⚠ PR #%d: Failed to get assignee for %s: %v\n", pr.GetNumber(), issue.Key, err)
				// Fall back to comment without mention
				if err := s.jiraClient.AddReleaseNotesComment(issue.Key, result.Notes, releaseNotesType, releaseNotesStatus, ""); err != nil {
					fmt.Printf("  ⚠ PR #%d: Failed to add comment to %s: %v\n", pr.GetNumber(), issue.Key, err)
				} else {
					fmt.Printf("  ✓ PR #%d: Added AI-generated release notes comment to %s\n", pr.GetNumber(), issue.Key)
				}
				continue
			}

			// Extract assignee account ID
			assigneeId := ""
			if assignee, ok := fullIssue.Fields["assignee"].(map[string]interface{}); ok {
				if accountId, ok := assignee["accountId"].(string); ok {
					assigneeId = accountId
				}
			}

			// Add comment with mention, type, and status
			if err := s.jiraClient.AddReleaseNotesComment(issue.Key, result.Notes, releaseNotesType, releaseNotesStatus, assigneeId); err != nil {
				fmt.Printf("  ⚠ PR #%d: Failed to add comment to %s: %v\n", pr.GetNumber(), issue.Key, err)
			} else {
				fmt.Printf("  ✓ PR #%d: Added AI-generated release notes comment to %s\n", pr.GetNumber(), issue.Key)
			}
		} else {
			// Extracted from PR: Add blue info panel (no assignee mention)
			if err := s.jiraClient.AddReleaseNotesComment(issue.Key, result.Notes, releaseNotesType, releaseNotesStatus, ""); err != nil {
				fmt.Printf("  ⚠ PR #%d: Failed to add release notes comment to %s: %v\n", pr.GetNumber(), issue.Key, err)
			} else {
				fmt.Printf("  ✓ PR #%d: Added release notes comment to %s (extracted from PR)\n", pr.GetNumber(), issue.Key)
			}
		}
	}
}
