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
	configFile                  string
	githubToken                 string
	jiraURL                     string
	jiraEmail                   string
	jiraToken                   string
	jiraPRField                 string
	jiraReleaseNotesTextField   string
	jiraReleaseNotesTypeField   string
	jiraReleaseNotesStatusField string
	geminiAPIKey                string
	geminiModel                 string
	since                       string
	slackWebhook                string
	transitionComment           bool
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
  - merged                  → "Dev Complete"
  - closed (without merge)  → "In Progress"

Note: Tickets are never automatically closed - "Dev Complete" is the furthest state.
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
	rootCmd.Flags().StringVar(&jiraReleaseNotesTextField, "jira-release-notes-text-field", "", "Jira custom field ID for release notes text (e.g., customfield_10783)")
	rootCmd.Flags().StringVar(&jiraReleaseNotesTypeField, "jira-release-notes-type-field", "", "Jira custom field ID for release notes type (e.g., customfield_10785)")
	rootCmd.Flags().StringVar(&jiraReleaseNotesStatusField, "jira-release-notes-status-field", "", "Jira custom field ID for release notes status (e.g., customfield_10807)")
	rootCmd.Flags().StringVar(&geminiAPIKey, "gemini-api-key", "", "Google Gemini API key for release notes generation (optional)")
	rootCmd.Flags().StringVar(&geminiModel, "gemini-model", "", "Gemini model to use (optional, defaults to gemini-3-flash-preview)")
	rootCmd.Flags().StringVar(&since, "since", "", "Only process PRs updated since this date (format: 2006-01-02 or DD/MM/YYYY). Defaults to yesterday if not provided.")
	rootCmd.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL for notifications (optional)")
	rootCmd.Flags().BoolVar(&transitionComment, "transition-comment", false, "Add a Jira comment on each transitioned ticket explaining the reason (optional)")

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
		jiraReleaseNotesTextField,
		jiraReleaseNotesTypeField,
		jiraReleaseNotesStatusField,
		geminiAPIKey,
		geminiModel,
		sinceTime,
		transitionComment,
	)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	// Run sync
	ctx := context.Background()
	summary, err := syncer.SyncAll(ctx, cfg.Repositories, cfg.Users)
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
	totalSkipped := 0
	totalTickets := 0
	totalErrors := 0

	for _, result := range summary.Results {
		fmt.Printf("\n%s/%s:\n", result.Repository.Owner, result.Repository.Name)
		fmt.Printf("  PRs processed: %d\n", result.PRsProcessed)
		if result.PRsSkipped > 0 {
			fmt.Printf("  PRs skipped (not in users list): %d\n", result.PRsSkipped)
		}
		fmt.Printf("  Tickets transitioned: %d\n", result.TicketsTransitioned)
		if len(result.Errors) > 0 {
			fmt.Printf("  Errors: %d\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Printf("    - %v\n", err)
			}
		}

		totalPRs += result.PRsProcessed
		totalSkipped += result.PRsSkipped
		totalTickets += result.TicketsTransitioned
		totalErrors += len(result.Errors)
	}

	fmt.Printf("\nTotal:\n")
	fmt.Printf("  PRs processed: %d\n", totalPRs)
	if totalSkipped > 0 {
		fmt.Printf("  PRs skipped (not in users list): %d\n", totalSkipped)
	}
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
