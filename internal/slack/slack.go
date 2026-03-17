package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client wraps Slack webhook functionality
type Client struct {
	webhookURL string
	httpClient *http.Client
}

// NewClient creates a new Slack client
func NewClient(webhookURL string) *Client {
	if webhookURL == "" {
		return nil
	}

	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{},
	}
}

// TransitionNotification represents a ticket that will be transitioned
type TransitionNotification struct {
	IssueKey       string
	IssueSummary   string
	IssueURL       string
	CurrentStatus  string
	TargetStatus   string
	TransitionPath string // e.g., "direct" or "To Do → In Progress → Code Review"
	PRs            []PRInfo
}

// PRInfo represents a PR linked to a ticket
type PRInfo struct {
	Number int
	Title  string
	URL    string
	State  string
	Repo   string // e.g., "tektoncd/results"
}

// SummaryStats contains summary statistics for the Slack footer
type SummaryStats struct {
	TicketsInCorrectStatus int
	UnlinkedPRs            int
}

// SendTransitionSummary sends a summary of tickets that will be transitioned
func (c *Client) SendTransitionSummary(notifications []TransitionNotification, dryRun bool, stats SummaryStats) error {
	if c == nil {
		return nil // Slack not configured
	}

	if len(notifications) == 0 {
		return nil // Nothing to notify
	}

	// Build Slack message
	message := c.buildMessage(notifications, dryRun, stats)

	// Send to Slack
	payload := map[string]interface{}{
		"blocks": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// buildMessage constructs the Slack Block Kit message
func (c *Client) buildMessage(notifications []TransitionNotification, dryRun bool, stats SummaryStats) []map[string]interface{} {
	blocks := []map[string]interface{}{}

	// Simple header
	contextText := "The following Jira tickets were processed for workflow transitions."
	blocks = append(blocks, map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": contextText,
		},
	})

	// Build ticket-first list with PRs grouped below
	messageText := ""
	for _, notif := range notifications {
		// Status indicator
		statusEmoji := ":white_check_mark:"
		statusText := "WOULD APPLY"
		if !dryRun {
			statusText = "APPLIED"
		}

		// Format: KEY `status` → `status` ✓ STATUS
		messageText += fmt.Sprintf("<%s|*%s*> `%s` → `%s` %s *%s*\n",
			notif.IssueURL,
			notif.IssueKey,
			notif.CurrentStatus,
			notif.TargetStatus,
			statusEmoji,
			statusText,
		)

		// List all related PRs in org/repo#num format
		for _, pr := range notif.PRs {
			// Truncate title to prevent line wrapping
			// Account for "  • org/repo#num: " prefix (~30-40 chars)
			maxTitleLen := 80
			title := pr.Title
			if len(title) > maxTitleLen {
				title = title[:maxTitleLen] + "..."
			}

			messageText += fmt.Sprintf("  • <%s|%s#%d>: %s\n",
				pr.URL,
				pr.Repo,
				pr.Number,
				title,
			)
		}

		// Add blank line between tickets
		messageText += "\n"
	}

	blocks = append(blocks, map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": messageText,
		},
	})

	// Compact footer with stats
	footerParts := []string{
		fmt.Sprintf("🤖 jirasync | %d ticket%s processed",
			len(notifications),
			pluralize(len(notifications)),
		),
	}

	if stats.TicketsInCorrectStatus > 0 {
		footerParts = append(footerParts,
			fmt.Sprintf("%d already in correct status",
				stats.TicketsInCorrectStatus,
			),
		)
	}

	if stats.UnlinkedPRs > 0 {
		footerParts = append(footerParts,
			fmt.Sprintf("%d unlinked PR%s",
				stats.UnlinkedPRs,
				pluralize(stats.UnlinkedPRs),
			),
		)
	}

	footerText := footerParts[0]
	if len(footerParts) > 1 {
		footerText += " • " + footerParts[1]
	}
	if len(footerParts) > 2 {
		footerText += " • " + footerParts[2]
	}

	blocks = append(blocks, map[string]interface{}{
		"type": "context",
		"elements": []map[string]interface{}{
			{
				"type": "mrkdwn",
				"text": footerText,
			},
		},
	})

	return blocks
}

// pluralize returns "s" if count != 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
