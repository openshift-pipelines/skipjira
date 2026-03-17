package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetTransitions fetches available transitions for an issue
func (c *Client) GetTransitions(issueKey string) ([]Transition, error) {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey))

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transitions: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}

	// Read the full response for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result TransitionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode transitions (response: %s): %w", string(body), err)
	}

	return result.Transitions, nil
}

// DoTransition executes a transition on an issue
func (c *Client) DoTransition(issueKey, transitionID string) error {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey))

	payload := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal transition payload: %w", err)
	}

	req, err := c.newRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute transition: %w", err)
	}
	defer resp.Body.Close()

	return c.checkResponse(resp, http.StatusNoContent, http.StatusOK)
}
