package jira

// Transition represents a Jira workflow transition
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

// TransitionsResponse is the response from GET /issue/{key}/transitions
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// SearchIssue represents a Jira issue from search results (includes key)
type SearchIssue struct {
	Key    string                 `json:"key"`
	Fields map[string]interface{} `json:"fields"`
}

// SearchResponse is the response from POST /search (JQL)
type SearchResponse struct {
	Issues []SearchIssue `json:"issues"`
	Total  int           `json:"total"`
}
