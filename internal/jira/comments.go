package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AddComment adds a simple comment to a Jira issue using ADF (Atlassian Document Format)
func (c *Client) AddComment(issueKey string, comment string) error {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey))

	// Build simple ADF document for the comment
	adfComment := adfNode{
		Version: 1,
		Type:    "doc",
		Content: []adfNode{
			{
				Type: "paragraph",
				Content: []adfNode{
					{
						Type: "text",
						Text: comment,
					},
				},
			},
		},
	}

	// Create request payload
	payload := map[string]interface{}{
		"body": adfComment,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal comment payload: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusCreated); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// AddReleaseNotesComment adds a highlighted AI-generated release notes comment with assignee mention
func (c *Client) AddReleaseNotesComment(issueKey, releaseNotes, releaseNotesType, releaseNotesStatus, assigneeAccountId string) error {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey))

	// Build ADF document for the comment with panel and mention
	adfComment := buildCommentADF(releaseNotes, releaseNotesType, releaseNotesStatus, assigneeAccountId)

	// Create request payload
	payload := map[string]interface{}{
		"body": adfComment,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal comment payload: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, http.StatusCreated); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// buildCommentADF creates an ADF document for a release notes comment
// If assigneeAccountId is provided, creates an attention-grabbing panel with mention (AI-generated)
// Otherwise, creates a simple panel for extracted release notes
func buildCommentADF(comment string, releaseNotesType string, releaseNotesStatus string, assigneeAccountId string) adfNode {
	if assigneeAccountId == "" {
		// Simple info panel for extracted release notes (no assignee mention)
		attrs := map[string]interface{}{
			"panelType": "info", // Blue info panel
		}

		releaseNotesParagraph := adfNode{
			Type: "paragraph",
			Content: []adfNode{
				{
					Type: "text",
					Text: comment,
				},
			},
		}

		metadataParagraph := adfNode{
			Type: "paragraph",
			Content: []adfNode{
				{
					Type: "text",
					Text: "Type: ",
					Marks: []map[string]interface{}{
						{"type": "strong"},
					},
				},
				{
					Type: "text",
					Text: releaseNotesType + " | ",
				},
				{
					Type: "text",
					Text: "Status: ",
					Marks: []map[string]interface{}{
						{"type": "strong"},
					},
				},
				{
					Type: "text",
					Text: releaseNotesStatus,
				},
			},
		}

		footerParagraph := adfNode{
			Type: "paragraph",
			Content: []adfNode{
				{
					Type: "text",
					Text: "Extracted from PR description",
					Marks: []map[string]interface{}{
						{"type": "em"}, // Italic
					},
				},
			},
		}

		return adfNode{
			Version: 1,
			Type:    "doc",
			Content: []adfNode{
				{
					Type:  "panel",
					Attrs: attrs,
					Content: []adfNode{
						{
							Type: "paragraph",
							Content: []adfNode{
								{
									Type: "text",
									Text: "📝 Release Notes",
									Marks: []map[string]interface{}{
										{"type": "strong"},
									},
								},
							},
						},
						releaseNotesParagraph,
						metadataParagraph,
						footerParagraph,
					},
				},
			},
		}
	}

	// Create attention-grabbing panel with mention for AI-generated notes
	attrs := map[string]interface{}{
		"panelType": "warning", // Orange warning panel for visibility
	}

	mentionParagraph := adfNode{
		Type: "paragraph",
		Content: []adfNode{
			{
				Type: "mention",
				Attrs: map[string]interface{}{
					"id": assigneeAccountId,
				},
			},
			{
				Type: "text",
				Text: " Please review these AI-generated release notes and add them to the Release Notes section if appropriate.",
			},
		},
	}

	releaseNotesParagraph := adfNode{
		Type: "paragraph",
		Content: []adfNode{
			{
				Type: "text",
				Text: comment,
			},
		},
	}

	// Add type and status metadata
	metadataParagraph := adfNode{
		Type: "paragraph",
		Content: []adfNode{
			{
				Type: "text",
				Text: "Type: ",
				Marks: []map[string]interface{}{
					{"type": "strong"},
				},
			},
			{
				Type: "text",
				Text: releaseNotesType + " | ",
			},
			{
				Type: "text",
				Text: "Status: ",
				Marks: []map[string]interface{}{
					{"type": "strong"},
				},
			},
			{
				Type: "text",
				Text: releaseNotesStatus,
			},
		},
	}

	footerParagraph := adfNode{
		Type: "paragraph",
		Content: []adfNode{
			{
				Type: "text",
				Text: "Generated by skipjira tool using AI from PR changes",
				Marks: []map[string]interface{}{
					{"type": "em"}, // Italic for footer
				},
			},
		},
	}

	return adfNode{
		Version: 1,
		Type:    "doc",
		Content: []adfNode{
			{
				Type:  "panel",
				Attrs: attrs,
				Content: []adfNode{
					{
						Type: "paragraph",
						Content: []adfNode{
							{
								Type: "text",
								Text: "🤖 AI-Generated Release Notes",
								Marks: []map[string]interface{}{
									{"type": "strong"},
								},
							},
						},
					},
					releaseNotesParagraph,
					metadataParagraph,
					footerParagraph,
					mentionParagraph,
				},
			},
		},
	}
}
