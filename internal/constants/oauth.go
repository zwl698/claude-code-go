package constants

import (
	"os"
	"strings"
)

// OAuth scope constants
const (
	// ClaudeAIInferenceScope is the scope for inference-only tokens
	ClaudeAIInferenceScope = "user:inference"
	// ClaudeAIProfileScope is the scope for profile access
	ClaudeAIProfileScope = "user:profile"
	// ConsoleScope is the scope for API key creation via Console
	ConsoleScope = "org:create_api_key"
	// OAuthBetaHeader is the beta header for OAuth
	OAuthBetaHeader = "oauth-2025-04-20"
)

// ConsoleOAuthScopes are the OAuth scopes for Console authentication
var ConsoleOAuthScopes = []string{
	ConsoleScope,
	ClaudeAIProfileScope,
}

// ClaudeAIOAuthScopes are the OAuth scopes for Claude.ai subscribers
var ClaudeAIOAuthScopes = []string{
	ClaudeAIProfileScope,
	ClaudeAIInferenceScope,
	"user:sessions:claude_code",
	"user:mcp_servers",
	"user:file_upload",
}

// AllOAuthScopes is the union of all OAuth scopes used in Claude CLI
var AllOAuthScopes = uniqueScopes(append(ConsoleOAuthScopes, ClaudeAIOAuthScopes...))

func uniqueScopes(scopes []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range scopes {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// OAuthConfig contains OAuth configuration
type OAuthConfig struct {
	BaseAPIURL           string
	ConsoleAuthorizeURL  string
	ClaudeAIAuthorizeURL string
	ClaudeAIOrigin       string
	TokenURL             string
	APIKeyURL            string
	RolesURL             string
	ConsoleSuccessURL    string
	ClaudeAISuccessURL   string
	ManualRedirectURL    string
	ClientID             string
	OAuthFileSuffix      string
	MCPProxyURL          string
	MCPProxyPath         string
}

// oauthConfigType represents the type of OAuth config
type oauthConfigType string

const (
	oauthConfigProd    oauthConfigType = "prod"
	oauthConfigStaging oauthConfigType = "staging"
	oauthConfigLocal   oauthConfigType = "local"
)

// getOAuthConfigType returns the OAuth config type based on environment
func getOAuthConfigType() oauthConfigType {
	if os.Getenv("USER_TYPE") == "ant" {
		if IsEnvTruthy(os.Getenv("USE_LOCAL_OAUTH")) {
			return oauthConfigLocal
		}
		if IsEnvTruthy(os.Getenv("USE_STAGING_OAUTH")) {
			return oauthConfigStaging
		}
	}
	return oauthConfigProd
}

// IsEnvTruthy checks if an environment variable is truthy
func IsEnvTruthy(val string) bool {
	val = strings.ToLower(val)
	return val == "true" || val == "1" || val == "yes"
}

// prodOAuthConfig is the production OAuth configuration
var prodOAuthConfig = OAuthConfig{
	BaseAPIURL:           "https://api.anthropic.com",
	ConsoleAuthorizeURL:  "https://platform.claude.com/oauth/authorize",
	ClaudeAIAuthorizeURL: "https://claude.com/cai/oauth/authorize",
	ClaudeAIOrigin:       "https://claude.ai",
	TokenURL:             "https://platform.claude.com/v1/oauth/token",
	APIKeyURL:            "https://api.anthropic.com/api/oauth/claude_cli/create_api_key",
	RolesURL:             "https://api.anthropic.com/api/oauth/claude_cli/roles",
	ConsoleSuccessURL:    "https://platform.claude.com/buy_credits?returnUrl=/oauth/code/success%3Fapp%3Dclaude-code",
	ClaudeAISuccessURL:   "https://platform.claude.com/oauth/code/success?app=claude-code",
	ManualRedirectURL:    "https://platform.claude.com/oauth/code/callback",
	ClientID:             "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
	OAuthFileSuffix:      "",
	MCPProxyURL:          "https://mcp-proxy.anthropic.com",
	MCPProxyPath:         "/v1/mcp/{server_id}",
}

// MCPClientMetadataURL is the Client ID Metadata Document URL for MCP OAuth
const MCPClientMetadataURL = "https://claude.ai/oauth/claude-code-client-metadata"

// AllowedOAuthBaseURLs are the allowed base URLs for CLAUDE_CODE_CUSTOM_OAUTH_URL override
var AllowedOAuthBaseURLs = []string{
	"https://beacon.claude-ai.staging.ant.dev",
	"https://claude.fedstart.com",
	"https://claude-staging.fedstart.com",
}

// GetOAuthConfig returns the OAuth configuration based on environment
func GetOAuthConfig() OAuthConfig {
	var config OAuthConfig

	switch getOAuthConfigType() {
	case oauthConfigLocal:
		config = getLocalOAuthConfig()
	case oauthConfigStaging:
		config = getStagingOAuthConfig()
	default:
		config = prodOAuthConfig
	}

	// Allow overriding all OAuth URLs to point to an approved FedStart deployment
	oauthBaseUrl := os.Getenv("CLAUDE_CODE_CUSTOM_OAUTH_URL")
	if oauthBaseUrl != "" {
		base := strings.TrimSuffix(oauthBaseUrl, "/")
		if !isAllowedOAuthBaseURL(base) {
			panic("CLAUDE_CODE_CUSTOM_OAUTH_URL is not an approved endpoint")
		}
		config = OAuthConfig{
			BaseAPIURL:           base,
			ConsoleAuthorizeURL:  base + "/oauth/authorize",
			ClaudeAIAuthorizeURL: base + "/oauth/authorize",
			ClaudeAIOrigin:       base,
			TokenURL:             base + "/v1/oauth/token",
			APIKeyURL:            base + "/api/oauth/claude_cli/create_api_key",
			RolesURL:             base + "/api/oauth/claude_cli/roles",
			ConsoleSuccessURL:    base + "/oauth/code/success?app=claude-code",
			ClaudeAISuccessURL:   base + "/oauth/code/success?app=claude-code",
			ManualRedirectURL:    base + "/oauth/code/callback",
			ClientID:             config.ClientID,
			OAuthFileSuffix:      "-custom-oauth",
			MCPProxyURL:          config.MCPProxyURL,
			MCPProxyPath:         config.MCPProxyPath,
		}
	}

	// Allow CLIENT_ID override via environment variable
	if clientID := os.Getenv("CLAUDE_CODE_OAUTH_CLIENT_ID"); clientID != "" {
		config.ClientID = clientID
	}

	return config
}

func isAllowedOAuthBaseURL(base string) bool {
	for _, allowed := range AllowedOAuthBaseURLs {
		if allowed == base {
			return true
		}
	}
	return false
}

func getStagingOAuthConfig() OAuthConfig {
	return OAuthConfig{
		BaseAPIURL:           "https://api-staging.anthropic.com",
		ConsoleAuthorizeURL:  "https://platform.staging.ant.dev/oauth/authorize",
		ClaudeAIAuthorizeURL: "https://claude-ai.staging.ant.dev/oauth/authorize",
		ClaudeAIOrigin:       "https://claude-ai.staging.ant.dev",
		TokenURL:             "https://platform.staging.ant.dev/v1/oauth/token",
		APIKeyURL:            "https://api-staging.anthropic.com/api/oauth/claude_cli/create_api_key",
		RolesURL:             "https://api-staging.anthropic.com/api/oauth/claude_cli/roles",
		ConsoleSuccessURL:    "https://platform.staging.ant.dev/buy_credits?returnUrl=/oauth/code/success%3Fapp%3Dclaude-code",
		ClaudeAISuccessURL:   "https://platform.staging.ant.dev/oauth/code/success?app=claude-code",
		ManualRedirectURL:    "https://platform.staging.ant.dev/oauth/code/callback",
		ClientID:             "22422756-60c9-4084-8eb7-27705fd5cf9a",
		OAuthFileSuffix:      "-staging-oauth",
		MCPProxyURL:          "https://mcp-proxy-staging.anthropic.com",
		MCPProxyPath:         "/v1/mcp/{server_id}",
	}
}

func getLocalOAuthConfig() OAuthConfig {
	api := getEnvOrDefault("CLAUDE_LOCAL_OAUTH_API_BASE", "http://localhost:8000")
	apps := getEnvOrDefault("CLAUDE_LOCAL_OAUTH_APPS_BASE", "http://localhost:4000")
	consoleBase := getEnvOrDefault("CLAUDE_LOCAL_OAUTH_CONSOLE_BASE", "http://localhost:3000")

	return OAuthConfig{
		BaseAPIURL:           strings.TrimSuffix(api, "/"),
		ConsoleAuthorizeURL:  consoleBase + "/oauth/authorize",
		ClaudeAIAuthorizeURL: apps + "/oauth/authorize",
		ClaudeAIOrigin:       strings.TrimSuffix(apps, "/"),
		TokenURL:             api + "/v1/oauth/token",
		APIKeyURL:            api + "/api/oauth/claude_cli/create_api_key",
		RolesURL:             api + "/api/oauth/claude_cli/roles",
		ConsoleSuccessURL:    consoleBase + "/buy_credits?returnUrl=/oauth/code/success%3Fapp%3Dclaude-code",
		ClaudeAISuccessURL:   consoleBase + "/oauth/code/success?app=claude-code",
		ManualRedirectURL:    consoleBase + "/oauth/code/callback",
		ClientID:             "22422756-60c9-4084-8eb7-27705fd5cf9a",
		OAuthFileSuffix:      "-local-oauth",
		MCPProxyURL:          "http://localhost:8205",
		MCPProxyPath:         "/v1/toolbox/shttp/mcp/{server_id}",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSuffix(val, "/")
	}
	return defaultValue
}

// FileSuffixForOAuthConfig returns the file suffix for the current OAuth config
func FileSuffixForOAuthConfig() string {
	if os.Getenv("CLAUDE_CODE_CUSTOM_OAUTH_URL") != "" {
		return "-custom-oauth"
	}
	switch getOAuthConfigType() {
	case oauthConfigLocal:
		return "-local-oauth"
	case oauthConfigStaging:
		return "-staging-oauth"
	default:
		return ""
	}
}
