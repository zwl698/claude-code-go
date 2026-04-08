// Package utils provides utility functions for the claude-code CLI.
// This file contains environment variable utilities.
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ========================================
// Claude Code Environment Variables
// ========================================

// GetClaudeConfigHome returns the Claude config home directory.
// Priority: CLAUDE_CONFIG_HOME > XDG_CONFIG_HOME/claude > ~/.claude
func GetClaudeConfigHome() string {
	// Check CLAUDE_CONFIG_HOME
	if dir := os.Getenv("CLAUDE_CONFIG_HOME"); dir != "" {
		return dir
	}

	// Check XDG_CONFIG_HOME
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "claude")
	}

	// Default to ~/.claude
	home := GetHomeDir()
	if home == "" {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// GetClaudeDataHome returns the Claude data home directory.
// Priority: CLAUDE_DATA_HOME > XDG_DATA_HOME/claude > ~/.claude/data
func GetClaudeDataHome() string {
	// Check CLAUDE_DATA_HOME
	if dir := os.Getenv("CLAUDE_DATA_HOME"); dir != "" {
		return dir
	}

	// Check XDG_DATA_HOME
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "claude")
	}

	// Default to ~/.claude/data
	configHome := GetClaudeConfigHome()
	return filepath.Join(configHome, "data")
}

// GetClaudeCacheHome returns the Claude cache home directory.
// Priority: CLAUDE_CACHE_HOME > XDG_CACHE_HOME/claude > ~/.claude/cache
func GetClaudeCacheHome() string {
	// Check CLAUDE_CACHE_HOME
	if dir := os.Getenv("CLAUDE_CACHE_HOME"); dir != "" {
		return dir
	}

	// Check XDG_CACHE_HOME
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "claude")
	}

	// Default to ~/.claude/cache
	configHome := GetClaudeConfigHome()
	return filepath.Join(configHome, "cache")
}

// ========================================
// User Type Detection
// ========================================

// UserType represents the type of user.
type UserType string

const (
	UserTypeInternal  UserType = "ant"      // Anthropic internal user
	UserTypeExternal  UserType = "external" // External user
	UserTypeDeveloper UserType = "developer"
)

// GetUserType returns the user type from environment.
func GetUserType() UserType {
	userType := os.Getenv("USER_TYPE")
	if userType == "" {
		return UserTypeExternal
	}
	return UserType(userType)
}

// IsInternalUser returns true if the user is an Anthropic internal user.
func IsInternalUser() bool {
	return GetUserType() == UserTypeInternal
}

// ========================================
// Feature Flags
// ========================================

// IsFeatureEnabled checks if a feature flag is enabled.
func IsFeatureEnabled(feature string) bool {
	envVar := "CLAUDE_CODE_FEATURE_" + strings.ToUpper(feature)
	return IsEnvTruthy(os.Getenv(envVar))
}

// IsProactiveModeEnabled checks if proactive mode is enabled.
func IsProactiveModeEnabled() bool {
	return IsFeatureEnabled("PROACTIVE") || IsFeatureEnabled("KAIROS")
}

// IsExperimentalModeEnabled checks if experimental features are enabled.
func IsExperimentalModeEnabled() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_EXPERIMENTAL"))
}

// ========================================
// Debug and Logging
// ========================================

// IsDebugEnabled checks if debug mode is enabled.
func IsDebugEnabled() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_DEBUG")) ||
		IsEnvTruthy(os.Getenv("DEBUG"))
}

// IsVerboseEnabled checks if verbose mode is enabled.
func IsVerboseEnabled() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_VERBOSE")) ||
		IsEnvTruthy(os.Getenv("VERBOSE"))
}

// ========================================
// Network and Proxy
// ========================================

// GetHTTPProxy returns the HTTP proxy URL.
func GetHTTPProxy() string {
	// Check multiple proxy environment variables
	for _, env := range []string{
		"CLAUDE_CODE_HTTP_PROXY",
		"HTTPS_PROXY",
		"https_proxy",
		"HTTP_PROXY",
		"http_proxy",
	} {
		if proxy := os.Getenv(env); proxy != "" {
			return proxy
		}
	}
	return ""
}

// GetNoProxy returns the no-proxy list.
func GetNoProxy() string {
	for _, env := range []string{"NO_PROXY", "no_proxy"} {
		if noProxy := os.Getenv(env); noProxy != "" {
			return noProxy
		}
	}
	return ""
}

// ========================================
// Entrypoint Detection
// ========================================

// Entrypoint represents where Claude Code was invoked from.
type Entrypoint string

const (
	EntrypointCLI      Entrypoint = "cli"
	EntrypointVSCode   Entrypoint = "vscode"
	EntrypointIntelliJ Entrypoint = "intellij"
	EntrypointNeovim   Entrypoint = "neovim"
	EntrypointEmacs    Entrypoint = "emacs"
	EntrypointUnknown  Entrypoint = "unknown"
)

// GetEntrypoint returns the entrypoint for this session.
func GetEntrypoint() Entrypoint {
	entrypoint := os.Getenv("CLAUDE_CODE_ENTRYPOINT")
	if entrypoint == "" {
		return EntrypointCLI
	}

	switch strings.ToLower(entrypoint) {
	case "vscode":
		return EntrypointVSCode
	case "intellij":
		return EntrypointIntelliJ
	case "neovim":
		return EntrypointNeovim
	case "emacs":
		return EntrypointEmacs
	default:
		return EntrypointCLI
	}
}

// ========================================
// Session and Process Info
// ========================================

// GetSessionID returns the current session ID.
func GetSessionID() string {
	return os.Getenv("CLAUDE_CODE_SESSION_ID")
}

// GetParentProcessID returns the parent process ID (if set).
func GetParentProcessID() string {
	return os.Getenv("CLAUDE_CODE_PARENT_PID")
}

// ========================================
// Subscription and Billing
// ========================================

// SubscriptionType represents the subscription type.
type SubscriptionType string

const (
	SubscriptionFree       SubscriptionType = "free"
	SubscriptionPro        SubscriptionType = "pro"
	SubscriptionTeam       SubscriptionType = "team"
	SubscriptionEnterprise SubscriptionType = "enterprise"
)

// GetSubscriptionType returns the subscription type from environment.
func GetSubscriptionType() SubscriptionType {
	sub := os.Getenv("CLAUDE_CODE_SUBSCRIPTION_TYPE")
	if sub == "" {
		return SubscriptionFree
	}
	return SubscriptionType(strings.ToLower(sub))
}

// HasBillingAccess checks if the user has billing access.
func HasBillingAccess() bool {
	// Internal users always have billing access
	if IsInternalUser() {
		return true
	}

	// Check subscription type
	sub := GetSubscriptionType()
	return sub == SubscriptionPro || sub == SubscriptionTeam || sub == SubscriptionEnterprise
}

// ========================================
// MCP Configuration
// ========================================

// GetMCPConfigPath returns the MCP configuration file path.
func GetMCPConfigPath() string {
	if path := os.Getenv("CLAUDE_CODE_MCP_CONFIG"); path != "" {
		return path
	}

	configHome := GetClaudeConfigHome()
	return filepath.Join(configHome, "mcp.json")
}

// ========================================
// Privacy and Telemetry
// ========================================

// PrivacyLevel represents the privacy level.
type PrivacyLevel string

const (
	PrivacyLevelDefault PrivacyLevel = "default"
	PrivacyLevelMinimal PrivacyLevel = "minimal"
	PrivacyLevelStrict  PrivacyLevel = "strict"
)

// GetPrivacyLevel returns the privacy level.
func GetPrivacyLevel() PrivacyLevel {
	level := os.Getenv("CLAUDE_CODE_PRIVACY_LEVEL")
	if level == "" {
		return PrivacyLevelDefault
	}
	return PrivacyLevel(strings.ToLower(level))
}

// IsTelemetryEnabled checks if telemetry is enabled.
func IsTelemetryEnabled() bool {
	// Disabled in strict privacy mode
	if GetPrivacyLevel() == PrivacyLevelStrict {
		return false
	}

	// Check explicit setting
	if disabled := IsEnvTruthy(os.Getenv("CLAUDE_CODE_DISABLE_TELEMETRY")); disabled {
		return false
	}

	return true
}

// IsEssentialTrafficOnly checks if only essential network traffic is allowed.
func IsEssentialTrafficOnly() bool {
	privacy := GetPrivacyLevel()
	return privacy == PrivacyLevelMinimal || privacy == PrivacyLevelStrict
}

// ========================================
// Expand Environment Variables
// ========================================

// ExpandEnv expands environment variables in a string.
// Supports ${VAR} and ${VAR:-default} syntax.
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		// Handle ${VAR:-default} syntax
		if strings.Contains(key, ":-") {
			parts := strings.SplitN(key, ":-", 2)
			envVar := parts[0]
			defaultVal := parts[1]
			if val := os.Getenv(envVar); val != "" {
				return val
			}
			return defaultVal
		}

		// Regular ${VAR} syntax
		return os.Getenv(key)
	})
}

// ExpandEnvWithMap expands environment variables with a custom map.
func ExpandEnvWithMap(s string, envMap map[string]string) string {
	return os.Expand(s, func(key string) string {
		// Handle ${VAR:-default} syntax
		if strings.Contains(key, ":-") {
			parts := strings.SplitN(key, ":-", 2)
			envVar := parts[0]
			defaultVal := parts[1]

			// Check custom map first
			if val, ok := envMap[envVar]; ok && val != "" {
				return val
			}
			// Fall back to OS environment
			if val := os.Getenv(envVar); val != "" {
				return val
			}
			return defaultVal
		}

		// Check custom map first
		if val, ok := envMap[key]; ok {
			return val
		}
		// Fall back to OS environment
		return os.Getenv(key)
	})
}
