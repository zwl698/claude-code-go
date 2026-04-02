package constants

import "strings"

// Product URLs
const (
	ProductURL = "https://claude.com/claude-code"
)

// Claude Code Remote session URLs
const (
	ClaudeAiBaseURL        = "https://claude.ai"
	ClaudeAiStagingBaseURL = "https://claude-ai.staging.ant.dev"
	ClaudeAiLocalBaseURL   = "http://localhost:4000"
)

// IsRemoteSessionStaging determines if we're in a staging environment for remote sessions.
// Checks session ID format and ingress URL.
func IsRemoteSessionStaging(sessionID, ingressURL string) bool {
	return strings.Contains(sessionID, "_staging_") || strings.Contains(ingressURL, "staging")
}

// IsRemoteSessionLocal determines if we're in a local-dev environment for remote sessions.
// Checks session ID format (e.g. `session_local_...`) and ingress URL.
func IsRemoteSessionLocal(sessionID, ingressURL string) bool {
	return strings.Contains(sessionID, "_local_") || strings.Contains(ingressURL, "localhost")
}

// GetClaudeAiBaseUrl gets the base URL for Claude AI based on environment.
func GetClaudeAiBaseUrl(sessionID, ingressURL string) string {
	if IsRemoteSessionLocal(sessionID, ingressURL) {
		return ClaudeAiLocalBaseURL
	}
	if IsRemoteSessionStaging(sessionID, ingressURL) {
		return ClaudeAiStagingBaseURL
	}
	return ClaudeAiBaseURL
}

// GetRemoteSessionUrl gets the full session URL for a remote session.
func GetRemoteSessionUrl(sessionID, ingressURL string) string {
	// In Go version, we don't have the cse_ shim, just use the session ID directly
	baseUrl := GetClaudeAiBaseUrl(sessionID, ingressURL)
	return baseUrl + "/code/" + sessionID
}
