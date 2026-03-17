package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openshift-pipelines/skipjira/internal/jirasync"
	"github.com/openshift-pipelines/skipjira/internal/slack"
	"github.com/spf13/cobra"
)

var (
	// CLI flags
	configFile   string
	githubToken  string
	jiraURL      string
	jiraEmail    string
	jiraToken    string
	jiraPRField  string
	since        string
	slackWebhook string
)

var rootCmd = &cobra.Command{
	Use:   "jirasync",
	Short: "Synchronize GitHub PR states to Jira ticket transitions",
	Long: `jirasync monitors GitHub pull requests across multiple repositories
and automatically transitions linked Jira tickets based on PR state.

PR States → Jira Transitions:
  - draft/changes_requested → "In Progress"
  - ready/review_requested  → "Code Review"
  - approved                → "Code Review"
  - merged/closed           → "On QA"

Note: Tickets are never automatically closed - "On QA" is the furthest state.
Closing tickets should be done manually after QA verification.`,
	RunE: runSync,
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "", "Path to repositories YAML config file (required)")
	rootCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token (required)")
	rootCmd.Flags().StringVar(&jiraURL, "jira-url", "", "Jira base URL (required)")
	rootCmd.Flags().StringVar(&jiraEmail, "jira-email", "", "Jira email for authentication (required)")
	rootCmd.Flags().StringVar(&jiraToken, "jira-token", "", "Jira API token (required)")
	rootCmd.Flags().StringVar(&jiraPRField, "jira-pr-field", "", "Jira custom field ID for PR links (required)")
	rootCmd.Flags().StringVar(&since, "since", "", "Only process PRs updated since this date (format: 2006-01-02 or DD/MM/YYYY). Defaults to yesterday if not provided.")
	rootCmd.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL for notifications (optional)")

	rootCmd.MarkFlagRequired("config")
	rootCmd.MarkFlagRequired("github-token")
	rootCmd.MarkFlagRequired("jira-url")
	rootCmd.MarkFlagRequired("jira-email")
	rootCmd.MarkFlagRequired("jira-token")
	rootCmd.MarkFlagRequired("jira-pr-field")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Load repository configuration
	cfg, err := jirasync.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Loaded %d repositories from config\n", len(cfg.Repositories))

	// Parse since date - default to yesterday if not provided
	var sinceTime time.Time
	if since != "" {
		var err error
		sinceTime, err = jirasync.ParseDate(since)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		fmt.Printf("Processing PRs updated since %s\n", sinceTime.Format("2006-01-02"))
	} else {
		// Default to yesterday (24 hours ago)
		sinceTime = time.Now().AddDate(0, 0, -1)
		fmt.Printf("Processing PRs updated since %s (yesterday - default)\n", sinceTime.Format("2006-01-02"))
	}

	// Create syncer
	syncer, err := jirasync.NewSyncer(
		githubToken,
		jiraURL,
		jiraEmail,
		jiraToken,
		jiraPRField,
		sinceTime,
	)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	// Run sync
	ctx := context.Background()
	summary, err := syncer.SyncAll(ctx, cfg.Repositories)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Print summary
	printSummary(summary)

	// Send Slack notification if configured
	if slackWebhook != "" && len(summary.Notifications) > 0 {
		fmt.Printf("\n📢 Sending Slack notification for %d tickets...\n", len(summary.Notifications))
		slackClient := slack.NewClient(slackWebhook)

		stats := slack.SummaryStats{
			TicketsInCorrectStatus: summary.TicketsInCorrectStatus,
			UnlinkedPRs:            summary.UnlinkedPRs,
		}

		if err := slackClient.SendTransitionSummary(summary.Notifications, false, stats); err != nil {
			fmt.Printf("  ⚠ Failed to send Slack notification: %v\n", err)
		} else {
			fmt.Printf("  ✓ Slack notification sent successfully\n")
		}
	}

	return nil
}

func printSummary(summary *jirasync.SyncSummary) {
	fmt.Println("\n=== Sync Summary ===")
	totalPRs := 0
	totalTickets := 0
	totalErrors := 0

	for _, result := range summary.Results {
		fmt.Printf("\n%s/%s:\n", result.Repository.Owner, result.Repository.Name)
		fmt.Printf("  PRs processed: %d\n", result.PRsProcessed)
		fmt.Printf("  Tickets transitioned: %d\n", result.TicketsTransitioned)
		if len(result.Errors) > 0 {
			fmt.Printf("  Errors: %d\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Printf("    - %v\n", err)
			}
		}

		totalPRs += result.PRsProcessed
		totalTickets += result.TicketsTransitioned
		totalErrors += len(result.Errors)
	}

	fmt.Printf("\nTotal:\n")
	fmt.Printf("  PRs: %d\n", totalPRs)
	fmt.Printf("  Tickets transitioned: %d\n", totalTickets)
	fmt.Printf("  Tickets already in correct status: %d\n", summary.TicketsInCorrectStatus)
	fmt.Printf("  Unlinked PRs: %d\n", summary.UnlinkedPRs)
	fmt.Printf("  Errors: %d\n", totalErrors)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
