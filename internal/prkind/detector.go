package prkind

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-pipelines/skipjira/internal/gemini"
)

// DetermineWithAI uses Gemini AI to determine PR kind from context
func DetermineWithAI(ctx context.Context, prContext PRKindContext, geminiClient *gemini.Client) (PRKind, error) {
	if geminiClient == nil {
		return "", fmt.Errorf("Gemini client is nil")
	}

	// Build prompt
	prompt := buildPRKindPrompt(prContext)

	// Call Gemini API with raw prompt
	response, err := geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}

	// Parse response - expect exactly "Bug Fix", "Feature", or "Enhancement"
	kind := strings.TrimSpace(response)

	// Validate and normalize response
	switch kind {
	case "Bug Fix", "bug fix", "BUG FIX":
		return KindBugFix, nil
	case "Feature", "feature", "FEATURE":
		return KindFeature, nil
	case "Enhancement", "enhancement", "ENHANCEMENT":
		return KindEnhancement, nil
	default:
		// If unexpected response, default to Enhancement
		return KindEnhancement, fmt.Errorf("unexpected Gemini response: %s, defaulting to Enhancement", kind)
	}
}
