package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-pipelines/skipjira/internal/audit"
	"github.com/spf13/cobra"
)

var (
	auditLines  int
	auditLevel  string
	auditAction string
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit logs",
	Long: `View audit logs for skipjira operations.

The audit log tracks all queue additions, sync operations, PR fetches, and Jira updates.
It's stored in ~/.config/skipjira/audit.jsonl`,
	RunE: runAudit,
}

func init() {
	auditCmd.Flags().IntVarP(&auditLines, "lines", "n", 50, "Number of recent log entries to show")
	auditCmd.Flags().StringVar(&auditLevel, "level", "", "Filter by log level (INFO, WARN, ERROR)")
	auditCmd.Flags().StringVar(&auditAction, "action", "", "Filter by action type")
}

func runAudit(cmd *cobra.Command, args []string) error {
	auditPath, err := audit.GetAuditLogPath()
	if err != nil {
		return fmt.Errorf("failed to get audit log path: %w", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No audit log found")
			return nil
		}
		return fmt.Errorf("failed to read audit log: %w", err)
	}

	if len(data) == 0 {
		fmt.Println("Audit log is empty")
		return nil
	}

	// Parse all entries
	var entries []audit.Entry
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var entry audit.Entry
		if err := decoder.Decode(&entry); err != nil {
			return fmt.Errorf("failed to decode entry: %w", err)
		}
		entries = append(entries, entry)
	}

	// Filter by level if specified
	if auditLevel != "" {
		filtered := []audit.Entry{}
		for _, entry := range entries {
			if strings.EqualFold(string(entry.Level), auditLevel) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	// Filter by action if specified
	if auditAction != "" {
		filtered := []audit.Entry{}
		for _, entry := range entries {
			if strings.EqualFold(string(entry.Action), auditAction) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	// Get last N entries
	start := 0
	if len(entries) > auditLines {
		start = len(entries) - auditLines
	}
	entries = entries[start:]

	// Display entries
	fmt.Printf("Showing last %d audit log entries:\n\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("[%s] %s - %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), entry.Level, entry.Action)
		fmt.Printf("  %s\n", entry.Message)
		if len(entry.Details) > 0 {
			fmt.Println("  Details:")
			for k, v := range entry.Details {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		fmt.Println()
	}

	return nil
}
