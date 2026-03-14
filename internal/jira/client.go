package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client wraps Jira API client
type Client struct {
	prField    string
	jiraURL    string
	jiraToken  string
	httpClient *http.Client
}

// NewClient creates a new Jira client
func NewClient(jiraURL, token, prField string) (*Client, error) {
	return &Client{
		prField:    prField,
		jiraURL:    jiraURL,
		jiraToken:  token,
		httpClient: &http.Client{},
	}, nil
}

// Issue represents a Jira issue response
type Issue struct {
	Fields map[string]interface{} `json:"fields"`
}

// AppendPRToTicket adds a PR URL to the ticket's Git Pull Request field
// The field is a comma-separated string of URLs
// Deduplicates to avoid adding the same PR URL twice
func (c *Client) AppendPRToTicket(issueKey, prURL string) error {
	// Get current issue
	issue, err := c.getIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue %s: %w", issueKey, err)
	}

	// Get current value of PR field (it's a comma-separated string)
	var prs []string
	if issue.Fields != nil {
		if val, ok := issue.Fields[c.prField]; ok && val != nil {
			// The field value can be a string or potentially other types
			switch v := val.(type) {
			case string:
				if v != "" {
					// Split by comma and trim spaces
					parts := strings.Split(v, ",")
					for _, part := range parts {
						trimmed := strings.TrimSpace(part)
						if trimmed != "" {
							prs = append(prs, trimmed)
						}
					}
				}
			}
		}
	}

	// Initialize array if nil
	if prs == nil {
		prs = []string{}
	}

	// Check if PR URL already exists
	for _, url := range prs {
		if url == prURL {
			// Already exists, no need to update
			return nil
		}
	}

	// Append new PR URL to the array
	prs = append(prs, prURL)

	// Update via direct REST API call
	return c.updatePRField(issueKey, prs)
}

// getIssue fetches an issue from Jira
func (c *Client) getIssue(issueKey string) (*Issue, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", strings.TrimRight(c.jiraURL, "/"), issueKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jiraToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// updatePRField updates the PR field using direct REST API call
// The field expects a comma-separated string of URLs
func (c *Client) updatePRField(issueKey string, prs []string) error {
	// Construct the API endpoint (use v3)
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", strings.TrimRight(c.jiraURL, "/"), issueKey)

	// Build the update payload
	// Pass as a comma-separated string
	value := strings.Join(prs, ", ")

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			c.prField: value,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jiraToken))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
