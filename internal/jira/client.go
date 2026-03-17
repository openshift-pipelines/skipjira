package jira

import (
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
		prField:   prField,
		jiraURL:   strings.TrimRight(jiraURL, "/"), // Normalize URL once
		jiraEmail: email,
		jiraToken: token,
		httpClient: &http.Client{
			Transport: &http.Transport{ForceAttemptHTTP2: true},
		},
	}, nil
}

// apiURL constructs a full API URL from a path
func (c *Client) apiURL(path string) string {
	return c.jiraURL + path
}

// newRequest creates an HTTP request with authentication headers
func (c *Client) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// checkResponse verifies the HTTP response status code
func (c *Client) checkResponse(resp *http.Response, expectedCodes ...int) error {
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			return nil
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
}
