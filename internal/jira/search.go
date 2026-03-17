package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SearchIssuesByJQL executes a JQL query and returns matching issue keys
func (c *Client) SearchIssuesByJQL(jql string) ([]string, error) {
	url := c.apiURL("/rest/api/3/search/jql")

	payload := map[string]interface{}{
		"jql":        jql,
		"fields":     []string{"key", "status", "summary"},
		"maxResults": 100,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JQL payload: %w", err)
	}

	req, err := c.newRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute JQL: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("JQL search failed: %w", err)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	keys := make([]string, len(result.Issues))
	for i, issue := range result.Issues {
		keys[i] = issue.Key
	}

	return keys, nil
}

// SearchIssuesWithStatusByJQL executes a JQL query and returns issues with their status
func (c *Client) SearchIssuesWithStatusByJQL(jql string) ([]SearchIssue, error) {
	url := c.apiURL("/rest/api/3/search/jql")

	payload := map[string]interface{}{
		"jql":        jql,
		"fields":     []string{"key", "status", "summary"},
		"maxResults": 100,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JQL payload: %w", err)
	}

	req, err := c.newRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute JQL: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("JQL search failed: %w", err)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return result.Issues, nil
}
