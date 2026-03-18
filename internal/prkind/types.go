package prkind

// PRKind represents the type of pull request
type PRKind string

const (
	// KindBugFix represents a bug fix PR
	KindBugFix PRKind = "Bug Fix"
	// KindFeature represents a new feature PR
	KindFeature PRKind = "Feature"
	// KindEnhancement represents an enhancement to existing functionality
	KindEnhancement PRKind = "Enhancement"
)

// PRKindContext contains the context needed for AI-based PR kind determination
type PRKindContext struct {
	Title          string
	Description    string
	CommitMessages []string
}
