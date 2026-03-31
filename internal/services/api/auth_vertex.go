// Package api provides API client functionality for the claude-code CLI.
// This file contains Google Vertex AI authentication utilities.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/utils"
)

// GoogleCredentials represents Google Cloud credentials.
type GoogleCredentials struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	Expiry      time.Time `json:"-"`
}

// GoogleAuthManager manages Google Cloud authentication state.
type GoogleAuthManager struct {
	mu          sync.RWMutex
	credentials *GoogleCredentials
	lastRefresh time.Time
}

var (
	googleAuthManager     *GoogleAuthManager
	googleAuthManagerOnce sync.Once
)

// GetGoogleAuthManager returns the singleton Google auth manager.
func GetGoogleAuthManager() *GoogleAuthManager {
	googleAuthManagerOnce.Do(func() {
		googleAuthManager = &GoogleAuthManager{}
	})
	return googleAuthManager
}

// Default TTL for Google credentials (1 hour)
const defaultGoogleCredentialTTL = 60 * time.Minute

// Short timeout for GCP credentials check to avoid metadata server delays
const gcpCredentialsCheckTimeout = 5 * time.Second

// RefreshGCPCredentialsIfNeeded refreshes Google Cloud credentials if needed.
func (g *GoogleAuthManager) RefreshGCPCredentialsIfNeeded(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if credentials are still valid
	if g.credentials != nil && time.Now().Before(g.credentials.Expiry) {
		return nil
	}

	// Try to get credentials from various sources
	creds, err := g.getCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Google credentials: %w", err)
	}

	g.credentials = creds
	g.lastRefresh = time.Now()
	return nil
}

// GetAccessToken returns the current access token.
func (g *GoogleAuthManager) GetAccessToken(ctx context.Context) (string, error) {
	if err := g.RefreshGCPCredentialsIfNeeded(ctx); err != nil {
		return "", err
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.credentials == nil {
		return "", fmt.Errorf("no Google credentials available")
	}

	return g.credentials.AccessToken, nil
}

// getCredentials attempts to get Google credentials from multiple sources.
func (g *GoogleAuthManager) getCredentials(ctx context.Context) (*GoogleCredentials, error) {
	// 1. Check for explicit credentials file
	if credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsFile != "" {
		creds, err := g.getCredsFromFile(credsFile)
		if err == nil {
			return creds, nil
		}
	}

	// 2. Try gcloud CLI credentials
	creds, err := g.getCredsFromGcloud(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	// 3. Try Application Default Credentials (ADC)
	creds, err = g.getADC(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	return nil, fmt.Errorf("no Google credentials found")
}

// getCredsFromFile loads credentials from a service account JSON file.
func (g *GoogleAuthManager) getCredsFromFile(filePath string) (*GoogleCredentials, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Check if this is a service account key
	var serviceAccount struct {
		Type          string `json:"type"`
		ProjectID     string `json:"project_id"`
		PrivateKeyID  string `json:"private_key_id"`
		PrivateKey    string `json:"private_key"`
		ClientEmail   string `json:"client_email"`
		ClientID      string `json:"client_id"`
		AuthURI       string `json:"auth_uri"`
		TokenURI      string `json:"token_uri"`
		AuthProvider  string `json:"auth_provider_x509_cert_url"`
		ClientCertURL string `json:"client_x509_cert_url"`
	}

	if err := json.Unmarshal(data, &serviceAccount); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if serviceAccount.Type == "service_account" {
		// For service accounts, we need to use OAuth2 to get an access token
		// This is a simplified implementation - in production, we'd use golang.org/x/oauth2
		return g.getServiceAccountToken(serviceAccount)
	}

	// Check if this is an ADC file
	var adc struct {
		Type         string `json:"type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.Unmarshal(data, &adc); err != nil {
		return nil, fmt.Errorf("failed to parse ADC file: %w", err)
	}

	if adc.Type == "authorized_user" {
		return g.getADCToken(adc)
	}

	return nil, fmt.Errorf("unknown credentials format")
}

// getServiceAccountToken gets an access token using service account credentials.
// Note: This is a simplified implementation. In production, use golang.org/x/oauth2/google.
func (g *GoogleAuthManager) getServiceAccountToken(sa struct {
	Type          string `json:"type"`
	ProjectID     string `json:"project_id"`
	PrivateKeyID  string `json:"private_key_id"`
	PrivateKey    string `json:"private_key"`
	ClientEmail   string `json:"client_email"`
	ClientID      string `json:"client_id"`
	AuthURI       string `json:"auth_uri"`
	TokenURI      string `json:"token_uri"`
	AuthProvider  string `json:"auth_provider_x509_cert_url"`
	ClientCertURL string `json:"client_x509_cert_url"`
}) (*GoogleCredentials, error) {
	// This would require JWT signing and OAuth2 flow
	// For now, we'll fall back to gcloud CLI
	return nil, fmt.Errorf("service account token generation not implemented, use gcloud auth")
}

// getADCToken refreshes an ADC token.
func (g *GoogleAuthManager) getADCToken(adc struct {
	Type         string `json:"type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}) (*GoogleCredentials, error) {
	// Use Google's OAuth2 endpoint to refresh the token
	tokenURL := "https://oauth2.googleapis.com/token"

	reqBody := fmt.Sprintf(
		"client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token",
		adc.ClientID, adc.ClientSecret, adc.RefreshToken,
	)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh ADC token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ADC token refresh failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ADC response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse ADC response: %w", err)
	}

	return &GoogleCredentials{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		Expiry:      time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// getCredsFromGcloud gets credentials from gcloud CLI.
func (g *GoogleAuthManager) getCredsFromGcloud(ctx context.Context) (*GoogleCredentials, error) {
	ctx, cancel := context.WithTimeout(ctx, gcpCredentialsCheckTimeout)
	defer cancel()

	// Try gcloud auth print-access-token
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud auth failed: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return nil, fmt.Errorf("empty access token from gcloud")
	}

	// Get the token expiry from gcloud
	expiryCmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token", "--format=json")
	expiryOutput, err := expiryCmd.Output()
	if err == nil {
		var tokenInfo struct {
			AccessToken struct {
				ExpireTime string `json:"expireTime"`
			} `json:"accessToken"`
		}
		if err := json.Unmarshal(expiryOutput, &tokenInfo); err == nil {
			if expiry, err := time.Parse(time.RFC3339, tokenInfo.AccessToken.ExpireTime); err == nil {
				return &GoogleCredentials{
					AccessToken: token,
					TokenType:   "Bearer",
					Expiry:      expiry,
				}, nil
			}
		}
	}

	// Default expiry if we can't determine it
	return &GoogleCredentials{
		AccessToken: token,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(defaultGoogleCredentialTTL),
	}, nil
}

// getADC gets Application Default Credentials.
func (g *GoogleAuthManager) getADC(ctx context.Context) (*GoogleCredentials, error) {
	// Check for ADC file in standard location
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	if _, err := os.Stat(adcPath); err == nil {
		return g.getCredsFromFile(adcPath)
	}

	// Try to check if running on GCP (metadata server)
	if g.isRunningOnGCP(ctx) {
		return g.getMetadataToken(ctx)
	}

	return nil, fmt.Errorf("no ADC found")
}

// isRunningOnGCP checks if running on Google Cloud Platform.
func (g *GoogleAuthManager) isRunningOnGCP(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://metadata.google.internal/computeMetadata/v1/", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// getMetadataToken gets an access token from the GCP metadata server.
func (g *GoogleAuthManager) getMetadataToken(ctx context.Context) (*GoogleCredentials, error) {
	ctx, cancel := context.WithTimeout(ctx, gcpCredentialsCheckTimeout)
	defer cancel()

	url := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse metadata response: %w", err)
	}

	return &GoogleCredentials{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		Expiry:      time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// ClearCache clears the cached credentials.
func (g *GoogleAuthManager) ClearCache() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.credentials = nil
	g.lastRefresh = time.Time{}
}

// CheckGCPCredentialsValid checks if GCP credentials are currently valid.
func (g *GoogleAuthManager) CheckGCPCredentialsValid(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, gcpCredentialsCheckTimeout)
	defer cancel()

	token, err := g.GetAccessToken(ctx)
	if err != nil {
		return false
	}

	// Verify the token by making a simple API call
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v1/tokeninfo?access_token="+token, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// NewVertexClient creates a client for Google Vertex AI with proper authentication.
func NewVertexClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	client, err := NewClient(opts)
	if err != nil {
		return nil, err
	}

	// Get Vertex-specific configuration
	region := utils.GetVertexRegionForModel(opts.Model)
	projectID := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")

	// Vertex uses a different endpoint pattern
	client.options.BaseURL = fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic",
		region, projectID, region,
	)

	// Check if we should skip auth (for testing/proxy scenarios)
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_SKIP_VERTEX_AUTH")) {
		return client, nil
	}

	// Initialize Google auth manager and get access token
	authManager := GetGoogleAuthManager()

	// Try to get access token
	token, err := authManager.GetAccessToken(ctx)
	if err != nil {
		// Log warning but don't fail - might be handled elsewhere
		fmt.Fprintf(os.Stderr, "Warning: failed to get Google credentials: %v\n", err)
	} else {
		client.googleAccessToken = token
		client.mu.Lock()
		client.defaultHeaders["Authorization"] = "Bearer " + token
		client.mu.Unlock()
	}

	return client, nil
}
