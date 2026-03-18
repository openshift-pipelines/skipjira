package gemini

// PRContext contains all the information about a PR needed to generate release notes
type PRContext struct {
	Title          string
	Description    string
	Diff           string
	CommitMessages []string
}
