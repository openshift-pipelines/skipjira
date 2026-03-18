package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Issue represents a Jira issue response
type Issue struct {
	Key    string                 `json:"key"`
	Fields map[string]interface{} `json:"fields"`
}

// GetIssueStatus fetches just the status of an issue (lightweight call)
func (c *Client) GetIssueStatus(issueKey string) (string, error) {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s?fields=status", issueKey))

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusOK); err != nil {
		return "", err
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract status name
	if statusField, ok := issue.Fields["status"].(map[string]interface{}); ok {
		if name, ok := statusField["name"].(string); ok {
			return name, nil
		}
	}

	return "", fmt.Errorf("status field not found in response")
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
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s?fields=%s", issueKey, c.prField))

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// GetIssueWithFields fetches an issue with specific fields
func (c *Client) GetIssueWithFields(issueKey string, fields []string) (*Issue, error) {
	fieldsParam := "assignee,summary"
	if len(fields) > 0 {
		fieldsParam = ""
		for i, f := range fields {
			if i > 0 {
				fieldsParam += ","
			}
			fieldsParam += f
		}
	}

	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s?fields=%s", issueKey, fieldsParam))

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// UpdateReleaseNotesField updates the release notes field on a Jira ticket
func (c *Client) UpdateReleaseNotesField(issueKey, releaseNotes string) error {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s", issueKey))

	// Build the update payload using configured field ID
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			c.releaseNotesField: releaseNotes,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := c.newRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return c.checkResponse(resp, http.StatusNoContent, http.StatusOK)
}

// updatePRField updates the PR field using direct REST API call
// The field expects an Atlassian Document Format (ADF) document
func (c *Client) updatePRField(issueKey string, prs []string) error {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s", issueKey))

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

	req, err := c.newRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return c.checkResponse(resp, http.StatusNoContent, http.StatusOK)
}
