package constants

import (
	"os"
	"strings"
)

// System prompt prefix types
type CLISyspromptPrefix string

const (
	DefaultPrefix                  CLISyspromptPrefix = "You are Claude Code, Anthropic's official CLI for Claude."
	AgentSDKClaudeCodePresetPrefix CLISyspromptPrefix = "You are Claude Code, Anthropic's official CLI for Claude, running within the Claude Agent SDK."
	AgentSDKPrefix                 CLISyspromptPrefix = "You are a Claude agent, built on Anthropic's Claude Agent SDK."
)

// CLISyspromptPrefixes contains all possible CLI sysprompt prefix values
var CLISyspromptPrefixes = map[CLISyspromptPrefix]bool{
	DefaultPrefix:                  true,
	AgentSDKClaudeCodePresetPrefix: true,
	AgentSDKPrefix:                 true,
}

// GetCLISyspromptPrefix returns the appropriate CLI sysprompt prefix based on context
func GetCLISyspromptPrefix(isNonInteractive bool, hasAppendSystemPrompt bool) CLISyspromptPrefix {
	// For now, return default prefix
	// TODO: Add API provider check when vertex support is added
	if isNonInteractive {
		if hasAppendSystemPrompt {
			return AgentSDKClaudeCodePresetPrefix
		}
		return AgentSDKPrefix
	}
	return DefaultPrefix
}

// SystemPromptDynamicBoundary is a marker separating static from dynamic content
const SystemPromptDynamicBoundary = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__"

// GetAttributionHeader returns the attribution header for API requests
func GetAttributionHeader(fingerprint string) string {
	// Check if attribution header is enabled
	if isAttributionHeaderEnabled() {
		return ""
	}

	version := "1.0.0." + fingerprint // TODO: Use actual version
	entrypoint := getEnvOrDefault("CLAUDE_CODE_ENTRYPOINT", "unknown")

	// TODO: Add native client attestation when supported
	// TODO: Add workload context when supported
	header := "x-anthropic-billing-header: cc_version=" + version + "; cc_entrypoint=" + entrypoint + ";"

	return header
}

func isAttributionHeaderEnabled() bool {
	// Check environment variable
	val := os.Getenv("CLAUDE_CODE_ATTRIBUTION_HEADER")
	if val == "false" || val == "0" || val == "no" {
		return false
	}
	// TODO: Add GrowthBook feature flag check when implemented
	return true
}

// ClaudeCodeDocsMapURL is the URL for the Claude Code docs map
const ClaudeCodeDocsMapURL = "https://code.claude.com/docs/en/claude_code_docs_map.md"

// GetUnameSR returns OS type and release (similar to uname -sr)
func GetUnameSR() string {
	// TODO: Implement actual OS detection
	// For now, return a generic value
	return "Unknown OS"
}

// Helper function
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}
	return defaultValue
}
