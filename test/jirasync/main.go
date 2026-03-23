package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openshift-pipelines/skipjira/internal/github"
	"github.com/openshift-pipelines/skipjira/internal/jira"
	"github.com/openshift-pipelines/skipjira/internal/jirasync"
	"github.com/openshift-pipelines/skipjira/internal/slack"
)

func main() {
	// Get tokens from environment
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable is required")
		fmt.Println("Usage: export GITHUB_TOKEN=your_token_here")
		os.Exit(1)
	}

	jiraToken := os.Getenv("JIRA_TOKEN")
	if jiraToken == "" {
		fmt.Println("Error: JIRA_TOKEN environment variable is required")
		fmt.Println("Usage: export JIRA_TOKEN=your_token_here")
		os.Exit(1)
	}

	jiraURL := os.Getenv("JIRA_URL")
	if jiraURL == "" {
		fmt.Println("Error: JIRA_URL environment variable is required")
		fmt.Println("Usage: export JIRA_URL=https://your-jira.com")
		os.Exit(1)
	}

	jiraEmail := os.Getenv("JIRA_EMAIL")
	if jiraEmail == "" {
		fmt.Println("Error: JIRA_EMAIL environment variable is required")
		fmt.Println("Usage: export JIRA_EMAIL=user@company.com")
		os.Exit(1)
	}

	jiraPRField := os.Getenv("JIRA_PR_FIELD")
	if jiraPRField == "" {
		fmt.Println("Error: JIRA_PR_FIELD environment variable is required")
		fmt.Println("Usage: export JIRA_PR_FIELD=customfield_12345")
		os.Exit(1)
	}

	// Check for Slack webhook URL (optional)
	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if slackWebhookURL != "" {
		fmt.Println("📢 Slack notifications enabled")
	}

	// Parse arguments
	var repos []jirasync.Repository

	if len(os.Args) == 1 {
		// No args - use default
		repos = []jirasync.Repository{
			{Owner: "openshift-pipelines", Name: "skipjira"},
		}
	} else if os.Args[1] == "--config" {
		// Config file mode
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./test/jirasync --config <config-file>")
			fmt.Println("   or: go run ./test/jirasync <owner> <repo>")
			os.Exit(1)
		}
		cfg, err := jirasync.LoadConfig(os.Args[2])
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		repos = cfg.Repositories
	} else if len(os.Args) >= 3 {
		// Single repo mode
		repos = []jirasync.Repository{
			{Owner: os.Args[1], Name: os.Args[2]},
		}
	} else {
		fmt.Println("Usage: go run ./test/jirasync                         # test openshift-pipelines/skipjira")
		fmt.Println("   or: go run ./test/jirasync <owner> <repo>          # test single repo")
		fmt.Println("   or: go run ./test/jirasync --config <config-file>  # test multiple repos")
		os.Exit(1)
	}

	// Test with PRs from last 7 days
	sinceTime := time.Now().AddDate(0, 0, -7)

	fmt.Printf("Testing jirasync flow for %d repository(ies)\n", len(repos))
	fmt.Printf("Fetching PRs updated since %s\n\n", sinceTime.Format("2006-01-02"))

	ctx := context.Background()

	// Step 3: Connect to Jira
	fmt.Println("=== Step 1: Connecting to Jira ===")
	jiraClient, err := jira.NewClient(jiraURL, jiraEmail, jiraToken, jiraPRField)
	if err != nil {
		fmt.Printf("Error creating Jira client: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Jira client created successfully\n")

	// Phase 1: Collect all PRs from all repositories
	fmt.Println("=== Step 2: Collecting PRs from All Repositories ===")

	type repoPRInfo struct {
		repo   jirasync.Repository
		number int
		title  string
		url    string
		state  github.PRState
	}

	// Global mapping: ticketKey → list of PRs from any repo
	globalTicketPRs := make(map[string][]repoPRInfo)
	globalTicketInfo := make(map[string]struct {
		Status  string
		Summary string
	})

	totalPRs := 0
	unlinkedPRs := 0
	for _, repo := range repos {
		fmt.Printf("\nRepository: %s/%s\n", repo.Owner, repo.Name)
		ghClient := github.NewClient(ghToken, repo.Owner, repo.Name)

		prs, err := ghClient.ListPRsSince(ctx, "all", sinceTime)
		if err != nil {
			fmt.Printf("  ✗ Error fetching PRs: %v\n", err)
			continue
		}

		fmt.Printf("  Found %d PRs\n", len(prs))
		totalPRs += len(prs)

		// Get PR states and link to tickets
		for _, pr := range prs {
			state, err := ghClient.GetPRState(ctx, pr)
			if err != nil {
				fmt.Printf("  ⚠ Warning: Failed to get state for PR #%d\n", pr.GetNumber())
				continue
			}

			targetStatus := jirasync.PRStateToJiraStatus(state)
			if targetStatus == "" {
				continue
			}

			// Search for Jira tickets
			jql := fmt.Sprintf(`"%s" ~ "%s"`, jiraPRField, pr.GetHTMLURL())
			issues, err := jiraClient.SearchIssuesWithStatusByJQL(jql)
			if err != nil {
				fmt.Printf("  ⚠ PR #%d: JQL search failed\n", pr.GetNumber())
				continue
			}

			// Track unlinked PRs
			if len(issues) == 0 {
				unlinkedPRs++
			}

			// Add PR to global ticket mapping
			for _, issue := range issues {
				globalTicketPRs[issue.Key] = append(globalTicketPRs[issue.Key], repoPRInfo{
					repo:   repo,
					number: pr.GetNumber(),
					title:  pr.GetTitle(),
					url:    pr.GetHTMLURL(),
					state:  state,
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
						if len(summaryField) > 60 {
							summary = summaryField[:60] + "..."
						} else {
							summary = summaryField
						}
					}

					globalTicketInfo[issue.Key] = struct {
						Status  string
						Summary string
					}{Status: status, Summary: summary}
				}
			}
		}
	}

	fmt.Printf("\n=== Collection Summary ===\n")
	fmt.Printf("Total repositories: %d\n", len(repos))
	fmt.Printf("Total PRs collected: %d\n", totalPRs)
	fmt.Printf("Unique Jira tickets: %d\n", len(globalTicketPRs))

	// Count cross-repo tickets
	crossRepoTickets := 0
	for _, prs := range globalTicketPRs {
		repoSet := make(map[string]bool)
		for _, pr := range prs {
			repoSet[fmt.Sprintf("%s/%s", pr.repo.Owner, pr.repo.Name)] = true
		}
		if len(repoSet) > 1 {
			crossRepoTickets++
		}
	}
	fmt.Printf("Tickets with PRs across multiple repos: %d\n", crossRepoTickets)

	if len(globalTicketPRs) == 0 {
		fmt.Println("\nNo Jira tickets found. Test complete.")
		return
	}

	// Phase 2: Process each ticket globally
	transitionsFound := 0
	ticketsInCorrectStatus := 0
	var slackNotifications []slack.TransitionNotification

	for issueKey, prs := range globalTicketPRs {
		info := globalTicketInfo[issueKey]

		fmt.Printf("[%s] %s\n", issueKey, info.Summary)
		fmt.Printf("  Current Status: %s\n", info.Status)

		// Skip tickets that are already closed
		if info.Status == "Closed" || info.Status == "Done" {
			fmt.Printf("  ⊗ Ticket is in terminal state - skipping\n\n")
			continue
		}

		fmt.Printf("  Linked PRs: %d\n", len(prs))

		// Find the most behind PR across all repos
		mostBehind := prs[0]
		for _, pr := range prs[1:] {
			if pr.state.Priority() < mostBehind.state.Priority() {
				mostBehind = pr
			}
		}

		// Show all linked PRs with repo info
		repoSet := make(map[string]bool)
		for _, pr := range prs {
			repoName := fmt.Sprintf("%s/%s", pr.repo.Owner, pr.repo.Name)
			repoSet[repoName] = true

			marker := "  "
			if pr.number == mostBehind.number {
				marker = "→ " // Mark the one we'll use
			}
			fmt.Printf("  %s [%s] PR #%d (%s) - %s\n", marker, repoName, pr.number, pr.state, pr.title)
		}

		targetStatus := jirasync.PRStateToJiraStatus(mostBehind.state)
		if len(repoSet) > 1 {
			fmt.Printf("  ⚠ PRs span %d repositories\n", len(repoSet))
		}
		if len(prs) > 1 {
			fmt.Printf("  Using most behind state: %s (from %s/%s PR #%d)\n",
				mostBehind.state, mostBehind.repo.Owner, mostBehind.repo.Name, mostBehind.number)
		}
		fmt.Printf("  Target Jira Status: %s\n", targetStatus)

		// Get available transitions
		transitions, err := jiraClient.GetTransitions(issueKey)
		if err != nil {
			fmt.Printf("  ✗ Failed to get transitions: %v\n\n", err)
			continue
		}

		if len(transitions) == 0 {
			fmt.Printf("  ⚠ No transitions available\n")
			fmt.Printf("  → Ticket would stay in '%s'\n\n", info.Status)
			continue
		}

		fmt.Printf("  Available transitions (%d): ", len(transitions))
		for i, t := range transitions {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s → %s", t.Name, t.To.Name)
		}
		fmt.Println()

		// Check if already in target state
		if info.Status == targetStatus {
			fmt.Printf("  ✓ Already in target status '%s' (no action needed)\n\n", info.Status)
			ticketsInCorrectStatus++
			continue
		}

		// Try direct transition first
		transition := jirasync.FindTransitionByName(transitions, targetStatus)
		var transitionPath string
		var canTransition bool

		if transition != nil {
			// Direct transition available
			fmt.Printf("  ✓ Would execute direct transition: '%s' → '%s'\n", info.Status, transition.To.Name)
			fmt.Printf("  → Using transition: '%s' (ID: %s)\n\n", transition.Name, transition.ID)
			transitionsFound++
			transitionPath = "direct"
			canTransition = true
		} else {
			// No direct transition - check for multi-step possibility
			fmt.Printf("  ⚠ No direct transition to '%s'\n", targetStatus)

			// Check if we can go through "In Progress" as intermediate
			inProgressTransition := jirasync.FindTransitionByName(transitions, "In Progress")
			if inProgressTransition != nil && info.Status != "In Progress" {
				fmt.Printf("  → Multi-step possible: '%s' → 'In Progress' → '%s'\n", info.Status, targetStatus)
				fmt.Printf("  ✓ Would attempt multi-step transition (max 3 steps)\n\n")
				transitionsFound++
				transitionPath = fmt.Sprintf("%s → In Progress → %s", info.Status, targetStatus)
				canTransition = true
			} else {
				// Check for any other intermediate state
				hasOtherTransitions := false
				for _, t := range transitions {
					if t.To.Name != info.Status { // Not a self-transition
						hasOtherTransitions = true
						break
					}
				}

				if hasOtherTransitions {
					fmt.Printf("  → Multi-step might be possible through intermediate states\n")
					fmt.Printf("  ⚠ Would attempt multi-step transition (uncertain path)\n\n")
					transitionsFound++
					// Don't set canTransition = true - too uncertain for Slack notification
				} else {
					fmt.Printf("  → No transitions available from '%s'\n", info.Status)
					fmt.Printf("  ✗ Ticket would stay in '%s'\n\n", info.Status)
					continue // Skip this ticket
				}
			}
		}

		// Collect Slack notification only for tickets with clear transition path
		if slackWebhookURL != "" && canTransition {
			// Build PR list for Slack
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

			slackNotifications = append(slackNotifications, slack.TransitionNotification{
				IssueKey:       issueKey,
				IssueSummary:   info.Summary,
				IssueURL:       fmt.Sprintf("%s/browse/%s", jiraURL, issueKey),
				CurrentStatus:  info.Status,
				TargetStatus:   targetStatus,
				TransitionPath: transitionPath,
				PRs:            slackPRs,
			})
		}
	}

	// Final summary
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("Overall Summary\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("Total repositories: %d\n", len(repos))
	fmt.Printf("Total PRs collected: %d\n", totalPRs)
	fmt.Printf("Unique Jira tickets: %d\n", len(globalTicketPRs))
	fmt.Printf("Tickets with cross-repo PRs: %d\n", crossRepoTickets)
	fmt.Printf("Transitions that would be executed: %d\n", transitionsFound)

	// Send Slack notification
	if slackWebhookURL != "" && len(slackNotifications) > 0 {
		fmt.Printf("\n📢 Sending Slack notification for %d tickets...\n", len(slackNotifications))
		slackClient := slack.NewClient(slackWebhookURL)

		stats := slack.SummaryStats{
			TicketsInCorrectStatus: ticketsInCorrectStatus,
			UnlinkedPRs:            unlinkedPRs,
		}

		if err := slackClient.SendTransitionSummary(slackNotifications, true, stats); err != nil {
			fmt.Printf("  ⚠ Failed to send Slack notification: %v\n", err)
		} else {
			fmt.Printf("  ✓ Slack notification sent successfully\n")
		}
	}

	fmt.Println("\nTest complete! No actual transitions were executed.")
	fmt.Println("To execute transitions, use: jirasync --config repos.yaml ...")
	if slackWebhookURL == "" {
		fmt.Println("\nTip: Set SLACK_WEBHOOK_URL env var to enable Slack notifications")
	}
}
