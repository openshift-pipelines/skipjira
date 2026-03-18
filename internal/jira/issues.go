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
			c.releaseNotesTextField: releaseNotes,
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

// UpdateReleaseNotesFields updates multiple release notes fields atomically
// Only updates fields that are configured (non-empty field IDs) AND currently empty
// Returns (updated bool, error) where updated indicates if any fields were changed
func (c *Client) UpdateReleaseNotesFields(issueKey, notes, kind, status string) (bool, error) {
	// Build list of fields to fetch
	var fieldsToFetch []string
	if c.releaseNotesTextField != "" {
		fieldsToFetch = append(fieldsToFetch, c.releaseNotesTextField)
	}
	if c.releaseNotesTypeField != "" {
		fieldsToFetch = append(fieldsToFetch, c.releaseNotesTypeField)
	}
	if c.releaseNotesStatusField != "" {
		fieldsToFetch = append(fieldsToFetch, c.releaseNotesStatusField)
	}

	if len(fieldsToFetch) == 0 {
		return false, fmt.Errorf("no release notes fields configured")
	}

	// Fetch current field values
	issue, err := c.GetIssueWithFields(issueKey, fieldsToFetch)
	if err != nil {
		return false, fmt.Errorf("failed to fetch current field values: %w", err)
	}

	// Only update fields if they are configured AND currently empty
	fields := make(map[string]interface{})

	if c.releaseNotesTextField != "" && isFieldEmpty(issue.Fields[c.releaseNotesTextField]) {
		// Text field requires ADF (Atlassian Document Format)
		fields[c.releaseNotesTextField] = convertToADF(notes)
	}
	if c.releaseNotesTypeField != "" && isFieldEmpty(issue.Fields[c.releaseNotesTypeField]) {
		// Option fields require object with "value" property
		fields[c.releaseNotesTypeField] = map[string]interface{}{"value": kind}
	}
	if c.releaseNotesStatusField != "" && isFieldEmpty(issue.Fields[c.releaseNotesStatusField]) {
		// Option fields require object with "value" property
		fields[c.releaseNotesStatusField] = map[string]interface{}{"value": status}
	}

	if len(fields) == 0 {
		// All fields already have values - nothing to update
		return false, nil
	}

	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s", issueKey))

	payload := map[string]interface{}{
		"fields": fields,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := c.newRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusNoContent, http.StatusOK); err != nil {
		return false, err
	}

	return true, nil
}

// convertToADF converts plain text to Atlassian Document Format
func convertToADF(text string) map[string]interface{} {
	return map[string]interface{}{
		"version": 1,
		"type":    "doc",
		"content": []map[string]interface{}{
			{
				"type": "paragraph",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": text,
					},
				},
			},
		},
	}
}

// isFieldEmpty checks if a Jira field value is empty or unset
// Returns true if the field should be updated (is empty), false if it has a value
func isFieldEmpty(fieldValue interface{}) bool {
	// Nil or not present
	if fieldValue == nil {
		return true
	}

	// Check if it's an option field (map with "value" key)
	if optionMap, ok := fieldValue.(map[string]interface{}); ok {
		// Check if it has a "value" key
		if value, hasValue := optionMap["value"]; hasValue {
			// Value key exists - check if it's empty
			if value == nil || value == "" {
				return true
			}
			return false
		}

		// Check if it's an ADF document (has "type": "doc")
		if docType, hasType := optionMap["type"]; hasType {
			if docType == "doc" {
				// It's an ADF document - check if it has content
				if content, hasContent := optionMap["content"]; hasContent {
					if contentSlice, ok := content.([]interface{}); ok {
						return len(contentSlice) == 0
					}
				}
				// No content or empty content
				return true
			}
		}
	}

	// For any other type, consider it non-empty
	return false
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
