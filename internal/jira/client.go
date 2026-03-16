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
	jiraEmail  string
	jiraToken  string
	httpClient *http.Client
}

// NewClient creates a new Jira client
func NewClient(jiraURL, email, token, prField string) (*Client, error) {
	return &Client{
		prField:    prField,
		jiraURL:    jiraURL,
		jiraEmail:  email,
		jiraToken:  token,
		httpClient: &http.Client{},
	}, nil
}

// Issue represents a Jira issue response
type Issue struct {
	Fields map[string]interface{} `json:"fields"`
}

// adfNode represents a node in Atlassian Document Format.
// Field order matches Jira's expected ADF serialization (version before type).
type adfNode struct {
	Version int       `json:"version,omitempty"`
	Type    string    `json:"type"`
	Content []adfNode `json:"content,omitempty"`
	Text    string    `json:"text,omitempty"`
}

// adfDoc builds an ADF document with all URLs as a single comma-separated text node.
// Version is always 1 — it represents the ADF spec version, not a revision counter.
func adfDoc(urls []string) adfNode {
	return adfNode{
		Version: 1,
		Type:    "doc",
		Content: []adfNode{
			{
				Type:    "paragraph",
				Content: []adfNode{{Type: "text", Text: strings.Join(urls, ", ")}},
			},
		},
	}
}

// extractURLsFromADF pulls the comma-separated URL list out of an ADF document returned by Jira
func extractURLsFromADF(val interface{}) []string {
	data, err := json.Marshal(val)
	if err != nil {
		return nil
	}
	var doc adfNode
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	var urls []string
	for _, para := range doc.Content {
		for _, inline := range para.Content {
			if inline.Type != "text" {
				continue
			}
			for _, part := range strings.Split(inline.Text, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					urls = append(urls, trimmed)
				}
			}
		}
	}
	return urls
}

// AppendPRToTicket adds a PR URL to the ticket's ADF PR field.
// Deduplicates to avoid adding the same PR URL twice.
func (c *Client) AppendPRToTicket(issueKey, prURL string) error {
	// Get current issue
	issue, err := c.getIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue %s: %w", issueKey, err)
	}

	// Get current value of PR field (ADF document format)
	var prs []string
	if issue.Fields != nil {
		if val, ok := issue.Fields[c.prField]; ok && val != nil {
			prs = extractURLsFromADF(val)
		}
	}
	if prs == nil {
		prs = []string{}
	}

	// Check if PR URL already exists
	for _, url := range prs {
		if url == prURL {
			return nil
		}
	}

	prs = append(prs, prURL)
	return c.updatePRField(issueKey, prs)
}

// getIssue fetches an issue from Jira
func (c *Client) getIssue(issueKey string) (*Issue, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=%s", strings.TrimRight(c.jiraURL, "/"), issueKey, c.prField)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
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
// The field expects an Atlassian Document Format (ADF) document
func (c *Client) updatePRField(issueKey string, prs []string) error {
	// Construct the API endpoint (use v3)
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", strings.TrimRight(c.jiraURL, "/"), issueKey)

	// Build the update payload using ADF
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			c.prField: adfDoc(prs),
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
	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := c.httpClient.Do(req)
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
