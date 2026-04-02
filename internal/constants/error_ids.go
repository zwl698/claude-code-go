package constants

// Error IDs for tracking error sources in production.
// These IDs are obfuscated identifiers that help us trace
// which logError() call generated an error.
//
// ADDING A NEW ERROR TYPE:
// 1. Add a const based on Next ID.
// 2. Increment Next ID.
// Next ID: 346

// Error ID constants
const (
	// EToolUseSummaryGenerationFailed is the error ID for tool use summary generation failures
	EToolUseSummaryGenerationFailed = 344
)
