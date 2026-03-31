// Package api provides API client functionality for the claude-code CLI.
// This file contains Azure Foundry authentication utilities.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/utils"
)

// AzureCredentials represents Azure AD credentials.
type AzureCredentials struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresOn   int64     `json:"expires_on"`
	Expiry      time.Time `json:"-"`
	Resource    string    `json:"resource"`
}

// AzureAuthManager manages Azure authentication state.
type AzureAuthManager struct {
	mu          sync.RWMutex
	credentials *AzureCredentials
	lastRefresh time.Time
}

var (
	azureAuthManager     *AzureAuthManager
	azureAuthManagerOnce sync.Once
)

// GetAzureAuthManager returns the singleton Azure auth manager.
func GetAzureAuthManager() *AzureAuthManager {
	azureAuthManagerOnce.Do(func() {
		azureAuthManager = &AzureAuthManager{}
	})
	return azureAuthManager
}

// Default TTL for Azure credentials (1 hour)
const defaultAzureCredentialTTL = 60 * time.Minute

// Azure resource for Cognitive Services
const azureCognitiveServicesResource = "https://cognitiveservices.azure.com/.default"

// RefreshAzureCredentialsIfNeeded refreshes Azure credentials if needed.
func (a *AzureAuthManager) RefreshAzureCredentialsIfNeeded(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if credentials are still valid
	if a.credentials != nil && time.Now().Before(a.credentials.Expiry) {
		return nil
	}

	// Try to get credentials from various sources
	creds, err := a.getCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Azure credentials: %w", err)
	}

	a.credentials = creds
	a.lastRefresh = time.Now()
	return nil
}

// GetAccessToken returns the current access token.
func (a *AzureAuthManager) GetAccessToken(ctx context.Context) (string, error) {
	if err := a.RefreshAzureCredentialsIfNeeded(ctx); err != nil {
		return "", err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.credentials == nil {
		return "", fmt.Errorf("no Azure credentials available")
	}

	return a.credentials.AccessToken, nil
}

// getCredentials attempts to get Azure credentials from multiple sources.
func (a *AzureAuthManager) getCredentials(ctx context.Context) (*AzureCredentials, error) {
	// 1. Check for explicit API key (Foundry API Key)
	if apiKey := os.Getenv("ANTHROPIC_FOUNDRY_API_KEY"); apiKey != "" {
		// API key authentication - no need for AD token
		// Return empty credentials, the API key will be used directly
		return &AzureCredentials{}, nil
	}

	// 2. Check for service principal credentials
	if tenantID := os.Getenv("AZURE_TENANT_ID"); tenantID != "" {
		creds, err := a.getServicePrincipalToken(ctx)
		if err == nil && creds != nil {
			return creds, nil
		}
	}

	// 3. Try Azure CLI credentials
	creds, err := a.getAzureCLIToken(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	// 4. Try managed identity (if running on Azure)
	creds, err = a.getManagedIdentityToken(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	return nil, fmt.Errorf("no Azure credentials found")
}

// getServicePrincipalToken gets a token using service principal credentials.
func (a *AzureAuthManager) getServicePrincipalToken(ctx context.Context) (*AzureCredentials, error) {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing service principal configuration")
	}

	// OAuth2 token endpoint for Azure AD
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", azureCognitiveServicesResource)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get service principal token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		ExpiresOn   int64  `json:"expires_on"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &AzureCredentials{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		ExpiresOn:   tokenResp.ExpiresOn,
		Expiry:      time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// getAzureCLIToken gets a token using Azure CLI.
func (a *AzureAuthManager) getAzureCLIToken(ctx context.Context) (*AzureCredentials, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try az account get-access-token
	cmd := exec.CommandContext(ctx, "az", "account", "get-access-token", "--resource", "https://cognitiveservices.azure.com", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Azure CLI auth failed: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int    `json:"expiresIn"`
		ExpiresOn   string `json:"expiresOn"`
	}

	if err := json.Unmarshal(output, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse Azure CLI response: %w", err)
	}

	expiry, _ := time.Parse(time.RFC3339, tokenResp.ExpiresOn)

	return &AzureCredentials{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		Expiry:      expiry,
	}, nil
}

// getManagedIdentityToken gets a token using Azure Managed Identity.
func (a *AzureAuthManager) getManagedIdentityToken(ctx context.Context) (*AzureCredentials, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try Azure Instance Metadata Service (IMDS)
	// First, check if we're running on Azure
	imdsURL := "http://169.254.169.254/metadata/identity/oauth2/token"

	params := url.Values{}
	params.Set("api-version", "2018-02-01")
	params.Set("resource", "https://cognitiveservices.azure.com")

	req, err := http.NewRequestWithContext(ctx, "GET", imdsURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Metadata", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("managed identity request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("managed identity returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read managed identity response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   string `json:"expires_in"`
		ExpiresOn   string `json:"expires_on"`
		Resource    string `json:"resource"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse managed identity response: %w", err)
	}

	expiresIn, _ := time.ParseDuration(tokenResp.ExpiresIn + "s")

	return &AzureCredentials{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   int(expiresIn.Seconds()),
		Expiry:      time.Now().Add(expiresIn),
		Resource:    tokenResp.Resource,
	}, nil
}

// ClearCache clears the cached credentials.
func (a *AzureAuthManager) ClearCache() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.credentials = nil
	a.lastRefresh = time.Time{}
}

// NewFoundryClient creates a client for Azure Foundry with proper authentication.
func NewFoundryClient(opts ClientOptions) (*Client, error) {
	client, err := NewClient(opts)
	if err != nil {
		return nil, err
	}

	// Get Foundry-specific configuration
	resource := os.Getenv("ANTHROPIC_FOUNDRY_RESOURCE")
	baseURLOverride := os.Getenv("ANTHROPIC_FOUNDRY_BASE_URL")

	if baseURLOverride != "" {
		client.options.BaseURL = baseURLOverride
	} else if resource != "" {
		client.options.BaseURL = fmt.Sprintf(
			"https://%s.services.ai.azure.com/anthropic/v1",
			resource,
		)
	}

	// Check if we should skip auth (for testing/proxy scenarios)
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_SKIP_FOUNDRY_AUTH")) {
		return client, nil
	}

	// Check for API key authentication
	if apiKey := os.Getenv("ANTHROPIC_FOUNDRY_API_KEY"); apiKey != "" {
		client.mu.Lock()
		client.defaultHeaders["api-key"] = apiKey
		client.mu.Unlock()
		return client, nil
	}

	// Initialize Azure auth manager and get access token
	ctx := context.Background()
	authManager := GetAzureAuthManager()

	// Try to get access token
	token, err := authManager.GetAccessToken(ctx)
	if err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to get Azure credentials: %v\n", err)
	} else if token != "" {
		client.azureAccessToken = token
		client.mu.Lock()
		client.defaultHeaders["Authorization"] = "Bearer " + token
		client.mu.Unlock()
	}

	return client, nil
}
