package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/theakshaypant/skipjira/internal/config"
	"github.com/theakshaypant/skipjira/internal/git"
	"github.com/theakshaypant/skipjira/internal/queue"
)

var (
	queueBranch  string
	queueJiraIDs string
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Add PR-to-Jira link to queue",
	Long: `Add a new entry to the queue for later processing.

This command is typically called by the pre-push hook, but can be run manually.`,
	RunE: runQueue,
}

func init() {
	queueCmd.Flags().StringVar(&queueBranch, "branch", "", "Branch name (required)")
	queueCmd.Flags().StringVar(&queueJiraIDs, "jira-ids", "", "Comma-separated Jira IDs (required)")
	queueCmd.MarkFlagRequired("branch")
	queueCmd.MarkFlagRequired("jira-ids")
}

func runQueue(cmd *cobra.Command, args []string) error {
	// Get git root and config
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	cfg, err := config.Load(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nRun 'skipjira install' first", err)
	}

	// Parse Jira IDs
	jiraIDList := strings.Split(queueJiraIDs, ",")
	for i, id := range jiraIDList {
		jiraIDList[i] = strings.TrimSpace(id)
	}

	// Create queue entry
	entry := &queue.Entry{
		Branch:    queueBranch,
		JiraIDs:   jiraIDList,
		Timestamp: time.Now(),
		RepoOwner: cfg.RepoOwner,
		RepoName:  cfg.RepoName,
		GitRoot:   gitRoot,
	}

	// Add to queue
	if err := queue.Add(entry); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	fmt.Printf("✓ Queued %s for Jira tickets: %s\n", queueBranch, queueJiraIDs)
	return nil
}
