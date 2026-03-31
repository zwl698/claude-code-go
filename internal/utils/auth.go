// Package utils provides utility functions for the claude-code CLI.
// This file contains authentication utilities.
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// APIKeySource represents the source of an API key.
type APIKeySource string

const (
	APIKeySourceEnvVar          APIKeySource = "ANTHROPIC_API_KEY"
	APIKeySourceAPIKeyHelper    APIKeySource = "apiKeyHelper"
	APIKeySourceLoginManagedKey APIKeySource = "/login managed key"
	APIKeySourceNone            APIKeySource = "none"
)

// OAuthTokens represents OAuth authentication tokens.
type OAuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
	Scopes       string `json:"scopes"`
}

// AuthManager manages authentication state.
type AuthManager struct {
	mu              sync.RWMutex
	apiKey          string
	apiKeySource    APIKeySource
	oauthTokens     *OAuthTokens
	isSubscriber    bool
	apiKeyHelperTTL time.Duration
	apiKeyCache     *apiKeyCache
}

type apiKeyCache struct {
	value     string
	timestamp time.Time
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		apiKeySource:    APIKeySourceNone,
		apiKeyHelperTTL: 5 * time.Minute,
	}
}

// GetAPIKey returns the API key if available.
func (a *AuthManager) GetAPIKey() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.apiKey
}

// GetAPIKeyWithSource returns the API key and its source.
func (a *AuthManager) GetAPIKeyWithSource() (string, APIKeySource) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.apiKey, a.apiKeySource
}

// SetAPIKey sets the API key.
func (a *AuthManager) SetAPIKey(key string, source APIKeySource) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.apiKey = key
	a.apiKeySource = source
}

// GetOAuthTokens returns the OAuth tokens.
func (a *AuthManager) GetOAuthTokens() *OAuthTokens {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.oauthTokens
}

// SetOAuthTokens sets the OAuth tokens.
func (a *AuthManager) SetOAuthTokens(tokens *OAuthTokens) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.oauthTokens = tokens
}

// IsClaudeAISubscriber returns true if the user is a Claude.ai subscriber.
func (a *AuthManager) IsClaudeAISubscriber() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isSubscriber
}

// SetIsSubscriber sets the subscriber status.
func (a *AuthManager) SetIsSubscriber(isSubscriber bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isSubscriber = isSubscriber
}

// IsAnthropicAuthEnabled checks if Anthropic auth is enabled.
func (a *AuthManager) IsAnthropicAuthEnabled() bool {
	// Check if using 3rd party services
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_BEDROCK")) ||
		IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_VERTEX")) ||
		IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_FOUNDRY")) {
		return false
	}

	// Check for external auth
	if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" && !a.isManagedOAuthContext() {
		return false
	}

	return true
}

// isManagedOAuthContext checks if this is a managed OAuth context.
func (a *AuthManager) isManagedOAuthContext() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_REMOTE")) ||
		os.Getenv("CLAUDE_CODE_ENTRYPOINT") == "claude-desktop"
}

// GetAPIKeyFromAPIKeyHelper executes the API key helper command to get an API key.
func (a *AuthManager) GetAPIKeyFromAPIKeyHelper(ctx context.Context) (string, error) {
	helperCmd := os.Getenv("ANTHROPIC_API_KEY_HELPER")
	if helperCmd == "" {
		return "", nil
	}

	// Check cache
	a.mu.RLock()
	if a.apiKeyCache != nil && time.Since(a.apiKeyCache.timestamp) < a.apiKeyHelperTTL {
		cached := a.apiKeyCache.value
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Execute helper command
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", helperCmd)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("API key helper failed: %w", err)
	}

	key := strings.TrimSpace(string(output))

	// Cache the result
	a.mu.Lock()
	a.apiKeyCache = &apiKeyCache{
		value:     key,
		timestamp: time.Now(),
	}
	a.mu.Unlock()

	return key, nil
}

// LoadOAuthTokens loads OAuth tokens from the config file.
func (a *AuthManager) LoadOAuthTokens() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, "oauth-tokens.json")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No tokens file is not an error
		}
		return err
	}

	var tokens OAuthTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return err
	}

	a.SetOAuthTokens(&tokens)
	return nil
}

// SaveOAuthTokens saves OAuth tokens to the config file.
func (a *AuthManager) SaveOAuthTokens(tokens *OAuthTokens) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, "oauth-tokens.json")
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

// CheckAndRefreshOAuthTokenIfNeeded checks if the OAuth token needs refresh.
func (a *AuthManager) CheckAndRefreshOAuthTokenIfNeeded(ctx context.Context) error {
	tokens := a.GetOAuthTokens()
	if tokens == nil {
		return nil
	}

	// Check if token is expired or about to expire (5 minute buffer)
	if tokens.ExpiresAt == 0 || time.Now().Add(5*time.Minute).Unix() > tokens.ExpiresAt {
		// Token needs refresh
		return a.refreshOAuthToken(ctx)
	}

	return nil
}

// refreshOAuthToken refreshes the OAuth token.
func (a *AuthManager) refreshOAuthToken(ctx context.Context) error {
	// This would normally call the OAuth refresh endpoint
	// For now, we just log that a refresh is needed
	return fmt.Errorf("OAuth token refresh not implemented")
}

// GetConfigDir returns the config directory.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// IsOAuthTokenExpired checks if the OAuth token is expired.
func (tokens *OAuthTokens) IsOAuthTokenExpired() bool {
	if tokens == nil || tokens.ExpiresAt == 0 {
		return true
	}
	return time.Now().Unix() >= tokens.ExpiresAt
}

// GetAuthMethod returns the current authentication method.
func (a *AuthManager) GetAuthMethod() string {
	if a.IsClaudeAISubscriber() && a.GetOAuthTokens() != nil {
		return "oauth"
	}
	if a.GetAPIKey() != "" {
		return "api_key"
	}
	return "none"
}

// InitializeAuth initializes authentication from environment and config.
func (a *AuthManager) InitializeAuth(ctx context.Context) error {
	// Load OAuth tokens from file
	if err := a.LoadOAuthTokens(); err != nil {
		// Non-fatal error
		fmt.Fprintf(os.Stderr, "Warning: failed to load OAuth tokens: %v\n", err)
	}

	// Check for API key from environment
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		a.SetAPIKey(apiKey, APIKeySourceEnvVar)
	}

	// Check for API key from helper
	if apiKey := os.Getenv("ANTHROPIC_API_KEY_HELPER"); apiKey != "" {
		key, err := a.GetAPIKeyFromAPIKeyHelper(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: API key helper failed: %v\n", err)
		} else if key != "" {
			a.SetAPIKey(key, APIKeySourceAPIKeyHelper)
		}
	}

	// Check for auth token
	if token := os.Getenv("ANTHROPIC_AUTH_TOKEN"); token != "" {
		// Auth token is used as bearer token, not API key
		a.SetOAuthTokens(&OAuthTokens{
			AccessToken: token,
		})
	}

	return nil
}
