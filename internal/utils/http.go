// Package utils provides utility functions for the claude-code CLI.
// This file contains HTTP utility functions.
package utils

import (
	"fmt"
	"os"
	"strings"
)

// GetUserAgent returns the User-Agent header for API requests.
func GetUserAgent() string {
	version := "1.0.0" // MACRO.VERSION equivalent
	userType := os.Getenv("USER_TYPE")
	if userType == "" {
		userType = "external"
	}

	entrypoint := os.Getenv("CLAUDE_CODE_ENTRYPOINT")
	if entrypoint == "" {
		entrypoint = "cli"
	}

	var parts []string

	// Agent SDK version
	if sdkVersion := os.Getenv("CLAUDE_AGENT_SDK_VERSION"); sdkVersion != "" {
		parts = append(parts, fmt.Sprintf("agent-sdk/%s", sdkVersion))
	}

	// Client app
	if clientApp := os.Getenv("CLAUDE_AGENT_SDK_CLIENT_APP"); clientApp != "" {
		parts = append(parts, fmt.Sprintf("client-app/%s", clientApp))
	}

	suffix := ""
	if len(parts) > 0 {
		suffix = ", " + strings.Join(parts, ", ")
	}

	return fmt.Sprintf("claude-cli/%s (%s, %s%s)", version, userType, entrypoint, suffix)
}

// GetMCPUserAgent returns the User-Agent for MCP requests.
func GetMCPUserAgent() string {
	version := "1.0.0"
	var parts []string

	if entrypoint := os.Getenv("CLAUDE_CODE_ENTRYPOINT"); entrypoint != "" {
		parts = append(parts, entrypoint)
	}
	if sdkVersion := os.Getenv("CLAUDE_AGENT_SDK_VERSION"); sdkVersion != "" {
		parts = append(parts, fmt.Sprintf("agent-sdk/%s", sdkVersion))
	}
	if clientApp := os.Getenv("CLAUDE_AGENT_SDK_CLIENT_APP"); clientApp != "" {
		parts = append(parts, fmt.Sprintf("client-app/%s", clientApp))
	}

	suffix := ""
	if len(parts) > 0 {
		suffix = " (" + strings.Join(parts, ", ") + ")"
	}

	return fmt.Sprintf("claude-code/%s%s", version, suffix)
}

// GetWebFetchUserAgent returns the User-Agent for web fetch requests.
func GetWebFetchUserAgent() string {
	return fmt.Sprintf("Claude-User (%s; +https://support.anthropic.com/)", GetClaudeCodeUserAgent())
}

// GetClaudeCodeUserAgent returns a simplified claude-code user agent.
func GetClaudeCodeUserAgent() string {
	version := "1.0.0"
	return fmt.Sprintf("claude-code/%s", version)
}

// AuthHeaders represents authentication headers for API requests.
type AuthHeaders struct {
	Headers map[string]string
	Error   string
}

// GetAuthHeaders returns authentication headers for API requests.
// Returns either OAuth headers for Max/Pro users or API key headers for regular users.
func GetAuthHeaders(apiKey string, oauthToken string, isSubscriber bool) AuthHeaders {
	if isSubscriber && oauthToken != "" {
		return AuthHeaders{
			Headers: map[string]string{
				"Authorization":  "Bearer " + oauthToken,
				"anthropic-beta": "oauth-2025-01-08",
			},
		}
	}

	if apiKey == "" {
		return AuthHeaders{
			Headers: map[string]string{},
			Error:   "No API key available",
		}
	}

	return AuthHeaders{
		Headers: map[string]string{
			"x-api-key": apiKey,
		},
	}
}

// GetCustomHeaders parses custom headers from the ANTHROPIC_CUSTOM_HEADERS environment variable.
func GetCustomHeaders() map[string]string {
	customHeaders := make(map[string]string)
	customHeadersEnv := os.Getenv("ANTHROPIC_CUSTOM_HEADERS")

	if customHeadersEnv == "" {
		return customHeaders
	}

	// Split by newlines to support multiple headers
	headerStrings := strings.Split(customHeadersEnv, "\n")
	for _, headerString := range headerStrings {
		headerString = strings.TrimSpace(headerString)
		if headerString == "" {
			continue
		}

		// Parse header in format "Name: Value" (curl style)
		colonIdx := strings.Index(headerString, ":")
		if colonIdx == -1 {
			continue
		}

		name := strings.TrimSpace(headerString[:colonIdx])
		value := strings.TrimSpace(headerString[colonIdx+1:])
		if name != "" {
			customHeaders[name] = value
		}
	}

	return customHeaders
}
