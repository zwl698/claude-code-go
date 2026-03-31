package constants

import (
	"os"
	"time"
)

// GetLocalISODate returns the LOCAL date in ISO format (YYYY-MM-DD)
func GetLocalISODate() string {
	// Check for date override
	if override := os.Getenv("CLAUDE_CODE_OVERRIDE_DATE"); override != "" {
		return override
	}

	now := time.Now()
	return now.Format("2006-01-02")
}

// GetLocalMonthYear returns "Month YYYY" (e.g. "February 2026") in the user's local timezone.
func GetLocalMonthYear() string {
	date := time.Now()
	if override := os.Getenv("CLAUDE_CODE_OVERRIDE_DATE"); override != "" {
		if parsed, err := time.Parse("2006-01-02", override); err == nil {
			date = parsed
		}
	}
	return date.Format("January 2006")
}

// GetSessionStartDate returns the cached session start date
var sessionStartDate string
var sessionStartDateCached bool

func GetSessionStartDate() string {
	if !sessionStartDateCached {
		sessionStartDate = GetLocalISODate()
		sessionStartDateCached = true
	}
	return sessionStartDate
}

// =============================================================================
// Application Constants
// =============================================================================

const (
	// Application name
	AppName = "claude-code-go"

	// Version
	Version = "1.0.0"

	// Default model
	DefaultModel = "claude-sonnet-4-20250514"

	// Default temperature
	DefaultTemperature = 1.0

	// Config directory name
	ConfigDirName = ".claude"

	// History file name
	HistoryFileName = "history.json"

	// Config file name
	ConfigFileName = "config.json"
)

// =============================================================================
// Environment Variables
// =============================================================================

const (
	// Environment variable for API key
	EnvAPIKey = "ANTHROPIC_API_KEY"

	// Environment variable for base URL
	EnvBaseURL = "ANTHROPIC_BASE_URL"

	// Environment variable for model
	EnvModel = "CLAUDE_MODEL"

	// Environment variable for permission mode
	EnvPermissionMode = "CLAUDE_PERMISSION_MODE"

	// Environment variable for verbose mode
	EnvVerbose = "CLAUDE_VERBOSE"

	// Environment variable for debug mode
	EnvDebug = "CLAUDE_DEBUG"
)

// =============================================================================
// Permission Modes
// =============================================================================

const (
	PermissionModeDefault = "default"
	PermissionModeAccept  = "accept"
	PermissionModeAuto    = "auto"
)

// =============================================================================
// MIME Types
// =============================================================================

const (
	MimeTypeJSON = "application/json"
	MimeTypeText = "text/plain"
	MimeTypePDF  = "application/pdf"
)

// =============================================================================
// Line Endings
// =============================================================================

const (
	LineEndingLF   = "LF"
	LineEndingCRLF = "CRLF"
)
