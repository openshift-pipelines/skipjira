package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skipjira",
	Short: "Link GitHub PRs to Jira tickets",
	Long: `SkipJira automatically links GitHub Pull Requests to Jira tickets.

It installs a git pre-push hook that prompts for Jira ticket IDs when pushing
new branches, then batch-updates Jira tickets via a cron job.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(queueCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
}
