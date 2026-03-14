package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theakshaypant/skipjira/internal/queue"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show queue status",
	Long:  `Display the current status of the queue (pending and failed entries only).

Successfully processed entries are removed from the queue and logged in the audit log.`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	entries, err := queue.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read queue: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("Queue is empty (all entries processed successfully)")
		fmt.Println("Use 'skipjira audit' to view processing history")
		return nil
	}

	fmt.Printf("Queue Status (pending/failed only):\n")
	fmt.Printf("  Total entries: %d\n\n", len(entries))

	fmt.Println("Entries:")
	for _, entry := range entries {
		status := "pending"
		if entry.Error != "" {
			status = fmt.Sprintf("error: %s", entry.Error)
		}
		fmt.Printf("  - %s/%s: %s → %v\n", entry.RepoOwner, entry.RepoName, entry.Branch, entry.JiraIDs)
		fmt.Printf("    Status: %s\n", status)
		if entry.PRURL != "" {
			fmt.Printf("    PR URL: %s\n", entry.PRURL)
		}
		fmt.Println()
	}

	fmt.Println("Note: Successfully processed entries are removed from the queue.")
	fmt.Println("Use 'skipjira audit' to view complete processing history.")

	return nil
}
