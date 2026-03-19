package main

import (
	"context"
	"fmt"

	"github.com/openshift-pipelines/skipjira/internal/audit"
	"github.com/openshift-pipelines/skipjira/internal/config"
	"github.com/openshift-pipelines/skipjira/internal/github"
	"github.com/openshift-pipelines/skipjira/internal/jira"
	"github.com/openshift-pipelines/skipjira/internal/queue"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Process queue and update Jira tickets",
	Long: `Process all pending entries in the queue.

For each entry:
1. Find the PR URL from GitHub (by branch name)
2. Update each Jira ticket with the PR URL
3. Mark the entry as processed

This command is typically run via cron.`,
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	// Read all entries (all entries in queue are pending or failed)
	allEntries, err := queue.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read queue: %w", err)
	}

	if len(allEntries) == 0 {
		fmt.Println("No entries in queue")
		audit.Info(audit.ActionSyncStart, "Sync started with no entries", nil)
		return nil
	}

	fmt.Printf("Processing %d entries...\n\n", len(allEntries))
	audit.Info(audit.ActionSyncStart, fmt.Sprintf("Sync started with %d entries", len(allEntries)), nil)

	// Track entries to keep (failed or still pending)
	var entriesToKeep []queue.Entry

	// Process each entry
	for i := range allEntries {
		entry := &allEntries[i]

		fmt.Printf("Processing: %s/%s branch %s\n", entry.RepoOwner, entry.RepoName, entry.Branch)

		// Load config for this repo using the stored git root path
		cfg, err := config.Load(entry.GitRoot)
		if err != nil {
			entry.Error = fmt.Sprintf("failed to load config: %v", err)
			fmt.Printf("  ✗ Error: %s\n", entry.Error)
			audit.Error(audit.ActionSyncEnd, "Failed to load config, removing entry", map[string]string{
				"branch": entry.Branch,
				"error":  err.Error(),
			})
			// Don't keep config errors - these are permanent failures
			continue
		}

		// Get PR URL from GitHub
		ghClient := github.NewClient(cfg.GithubToken, entry.RepoOwner, entry.RepoName)
		ctx := context.Background()

		headOwner := entry.ForkOwner
		if headOwner == "" {
			headOwner = entry.RepoOwner
		}
		prURL, err := ghClient.GetPRForBranch(ctx, headOwner, entry.Branch)
		if err != nil {
			entry.Error = fmt.Sprintf("failed to get PR: %v", err)
			// Keep in queue - PR might be created later
			fmt.Printf("  ✗ Error: %s (will retry)\n", entry.Error)
			audit.Error(audit.ActionPRFetch, "Failed to fetch PR URL, keeping in queue", map[string]string{
				"branch":     entry.Branch,
				"repo_owner": entry.RepoOwner,
				"repo_name":  entry.RepoName,
				"error":      err.Error(),
			})
			entriesToKeep = append(entriesToKeep, *entry)
			continue
		}

		entry.PRURL = prURL
		fmt.Printf("  Found PR: %s\n", prURL)
		audit.Info(audit.ActionPRFetch, "PR URL fetched successfully", map[string]string{
			"branch":     entry.Branch,
			"repo_owner": entry.RepoOwner,
			"repo_name":  entry.RepoName,
			"pr_url":     prURL,
		})

		// Update Jira tickets
		jiraClient, err := jira.NewClient(cfg.JiraURL, cfg.JiraEmail, cfg.JiraToken, cfg.JiraPRField, "", "", "")
		if err != nil {
			entry.Error = fmt.Sprintf("failed to create Jira client: %v", err)
			fmt.Printf("  ✗ Error: %s\n", entry.Error)
			audit.Error(audit.ActionSyncEnd, "Failed to create Jira client, removing entry", map[string]string{
				"branch": entry.Branch,
				"error":  err.Error(),
			})
			// Don't keep - this is a permanent configuration error
			continue
		}

		successCount := 0
		for _, jiraID := range entry.JiraIDs {
			// Try to update the PR field first
			err := jiraClient.AppendPRToTicket(jiraID, prURL)
			if err != nil {
				fmt.Printf("  ✗ Failed to link %s: %v\n", jiraID, err)
				audit.Error(audit.ActionJiraUpdate, "Failed to update Jira ticket", map[string]string{
					"jira_id": jiraID,
					"pr_url":  prURL,
					"branch":  entry.Branch,
					"error":   err.Error(),
				})
			} else {
				fmt.Printf("  ✓ Updated field on %s\n", jiraID)
				audit.Info(audit.ActionJiraUpdate, "Jira ticket updated successfully", map[string]string{
					"jira_id": jiraID,
					"pr_url":  prURL,
					"branch":  entry.Branch,
				})
				successCount++
			}
		}

		if successCount == len(entry.JiraIDs) {
			// Complete success - remove from queue (logged in audit)
			fmt.Printf("  ✓ All tickets updated successfully, removing from queue\n\n")
			audit.Info(audit.ActionEntryUpdate, "Entry fully processed, removed from queue", map[string]string{
				"branch":    entry.Branch,
				"jira_ids":  fmt.Sprintf("%v", entry.JiraIDs),
				"pr_url":    entry.PRURL,
				"processed": fmt.Sprintf("%d/%d", successCount, len(entry.JiraIDs)),
			})
		} else if successCount > 0 {
			// Partial success - keep in queue for retry
			fmt.Printf("  ⚠ Partial success: %d/%d tickets updated, keeping in queue\n\n", successCount, len(entry.JiraIDs))
			audit.Warn(audit.ActionEntryUpdate, "Entry partially processed, keeping in queue", map[string]string{
				"branch":    entry.Branch,
				"jira_ids":  fmt.Sprintf("%v", entry.JiraIDs),
				"pr_url":    entry.PRURL,
				"processed": fmt.Sprintf("%d/%d", successCount, len(entry.JiraIDs)),
			})
			entriesToKeep = append(entriesToKeep, *entry)
		} else {
			// Complete failure - keep in queue for retry
			fmt.Printf("  ✗ Failed to update any tickets, keeping in queue\n\n")
			audit.Error(audit.ActionEntryUpdate, "Entry failed to process any tickets, keeping in queue", map[string]string{
				"branch":   entry.Branch,
				"jira_ids": fmt.Sprintf("%v", entry.JiraIDs),
				"pr_url":   entry.PRURL,
			})
			entriesToKeep = append(entriesToKeep, *entry)
		}
	}

	// Write updated queue (only failed/pending entries)
	if err := queue.WriteAll(entriesToKeep); err != nil {
		audit.Error(audit.ActionSyncEnd, "Failed to write updated queue", map[string]string{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to write queue: %w", err)
	}

	successfulCount := len(allEntries) - len(entriesToKeep)
	fmt.Printf("Sync complete! Successfully processed: %d, Remaining in queue: %d\n", successfulCount, len(entriesToKeep))
	audit.Info(audit.ActionSyncEnd, fmt.Sprintf("Sync completed: %d successful, %d remaining", successfulCount, len(entriesToKeep)), map[string]string{
		"successful": fmt.Sprintf("%d", successfulCount),
		"remaining":  fmt.Sprintf("%d", len(entriesToKeep)),
	})
	return nil
}
