package jira

import (
	"encoding/json"
	"strings"
)

// adfNode represents a node in Atlassian Document Format.
// Field order matches Jira's expected ADF serialization (version before type).
type adfNode struct {
	Version int                    `json:"version,omitempty"`
	Type    string                 `json:"type"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Content []adfNode              `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []map[string]interface{} `json:"marks,omitempty"`
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
