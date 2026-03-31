// Package api provides API client functionality for the claude-code CLI.
// This file contains AWS Bedrock authentication utilities.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/utils"
)

// AWSCredentials represents AWS session credentials.
type AWSCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      time.Time
}

// AWSAuthManager manages AWS authentication state.
type AWSAuthManager struct {
	mu          sync.RWMutex
	credentials *AWSCredentials
	lastRefresh time.Time
}

var (
	awsAuthManager     *AWSAuthManager
	awsAuthManagerOnce sync.Once
)

// GetAWSAuthManager returns the singleton AWS auth manager.
func GetAWSAuthManager() *AWSAuthManager {
	awsAuthManagerOnce.Do(func() {
		awsAuthManager = &AWSAuthManager{}
	})
	return awsAuthManager
}

// Default TTL for AWS STS credentials (1 hour)
const defaultAWSSTSTTL = 60 * time.Minute

// RefreshAndGetAWSCredentials refreshes AWS credentials if needed.
// This implements the credential refresh logic similar to the TypeScript version.
func (a *AWSAuthManager) RefreshAndGetAWSCredentials(ctx context.Context) (*AWSCredentials, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if credentials are still valid
	if a.credentials != nil && time.Since(a.lastRefresh) < defaultAWSSTSTTL {
		return a.credentials, nil
	}

	// Try to get credentials from various sources
	creds, err := a.getCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	a.credentials = creds
	a.lastRefresh = time.Now()
	return creds, nil
}

// getCredentials attempts to get AWS credentials from multiple sources.
func (a *AWSAuthManager) getCredentials(ctx context.Context) (*AWSCredentials, error) {
	// 1. Check for AWS Bearer Token (Bedrock API key authentication)
	if bearerToken := os.Getenv("AWS_BEARER_TOKEN_BEDROCK"); bearerToken != "" {
		// Bearer token authentication - no need for standard credentials
		return &AWSCredentials{
			// Bearer token is handled separately in the request headers
			AccessKeyID:     "",
			SecretAccessKey: "",
			SessionToken:    "",
		}, nil
	}

	// 2. Check for explicit AWS credentials from environment
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		return &AWSCredentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		}, nil
	}

	// 3. Try AWS credential export command (similar to TypeScript's awsCredentialExport)
	creds, err := a.getAWSCredsFromCredentialExport(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	// 4. Try AWS CLI configured credentials
	creds, err = a.getAWSCredsFromCLI(ctx)
	if err == nil && creds != nil {
		return creds, nil
	}

	return nil, fmt.Errorf("no AWS credentials found")
}

// getAWSCredsFromCredentialExport runs the awsCredentialExport command if configured.
func (a *AWSAuthManager) getAWSCredsFromCredentialExport(ctx context.Context) (*AWSCredentials, error) {
	// This would be read from settings, for now we skip this
	// as the Go version doesn't have the full settings system yet
	return nil, fmt.Errorf("awsCredentialExport not configured")
}

// getAWSCredsFromCLI attempts to get credentials from AWS CLI.
func (a *AWSAuthManager) getAWSCredsFromCLI(ctx context.Context) (*AWSCredentials, error) {
	// Try to get credentials using AWS CLI
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("AWS CLI not available or not configured: %w", err)
	}

	// Parse the output to verify we have valid credentials
	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
		UserId  string `json:"UserId"`
	}
	if err := json.Unmarshal(output, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse AWS CLI output: %w", err)
	}

	// AWS CLI handles credential resolution automatically
	// We need to get the actual credentials
	creds, err := a.getAWSCredsFromCLIConfig()
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// getAWSCredsFromCLIConfig reads credentials from AWS CLI config files.
func (a *AWSAuthManager) getAWSCredsFromCLIConfig() (*AWSCredentials, error) {
	// Get the active profile
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	// Read credentials file
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	credsPath := home + "/.aws/credentials"
	credsData, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS credentials file: %w", err)
	}

	// Parse the credentials file (simple INI format)
	creds, err := parseAWSCreds(string(credsData), profile)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// parseAWSCreds parses AWS credentials file in INI format.
func parseAWSCreds(data string, profile string) (*AWSCredentials, error) {
	lines := strings.Split(data, "\n")
	currentProfile := ""
	creds := &AWSCredentials{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for profile header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentProfile = strings.Trim(line, "[]")
			// Handle [profile name] format
			if strings.HasPrefix(currentProfile, "profile ") {
				currentProfile = strings.TrimPrefix(currentProfile, "profile ")
			}
			continue
		}

		// Parse key=value
		if currentProfile == profile {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "aws_access_key_id":
				creds.AccessKeyID = value
			case "aws_secret_access_key":
				creds.SecretAccessKey = value
			case "aws_session_token":
				creds.SessionToken = value
			}
		}
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("no credentials found for profile %s", profile)
	}

	return creds, nil
}

// CheckSTSCallerIdentity verifies AWS credentials by calling STS GetCallerIdentity.
func (a *AWSAuthManager) CheckSTSCallerIdentity(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials: %w", err)
	}

	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
		UserId  string `json:"UserId"`
	}
	if err := json.Unmarshal(output, &identity); err != nil {
		return fmt.Errorf("failed to parse STS response: %w", err)
	}

	return nil
}

// ClearCache clears the cached credentials.
func (a *AWSAuthManager) ClearCache() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.credentials = nil
	a.lastRefresh = time.Time{}
}

// NewBedrockClient creates a client for AWS Bedrock with proper authentication.
func NewBedrockClient(opts ClientOptions) (*Client, error) {
	client, err := NewClient(opts)
	if err != nil {
		return nil, err
	}

	// Set Bedrock-specific base URL
	region := utils.GetAWSRegion()
	if opts.Model != "" && os.Getenv("ANTHROPIC_SMALL_FAST_MODEL_AWS_REGION") != "" {
		// Use region override for small fast model
		region = os.Getenv("ANTHROPIC_SMALL_FAST_MODEL_AWS_REGION")
	}

	// Bedrock uses a different endpoint pattern
	client.options.BaseURL = fmt.Sprintf(
		"https://bedrock-runtime.%s.amazonaws.com",
		region,
	)

	// Check if we should skip auth (for testing/proxy scenarios)
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_SKIP_BEDROCK_AUTH")) {
		return client, nil
	}

	// Handle Bearer token authentication (Bedrock API Key)
	if bearerToken := os.Getenv("AWS_BEARER_TOKEN_BEDROCK"); bearerToken != "" {
		client.defaultHeaders["Authorization"] = "Bearer " + bearerToken
		return client, nil
	}

	// Initialize AWS auth manager and get credentials
	ctx := context.Background()
	authManager := GetAWSAuthManager()

	// Try to get credentials (this will refresh if needed)
	creds, err := authManager.RefreshAndGetAWSCredentials(ctx)
	if err != nil {
		// Log warning but don't fail - AWS SDK might handle it
		// This allows the SDK to use its default credential chain
		fmt.Fprintf(os.Stderr, "Warning: failed to get AWS credentials: %v\n", err)
	}

	// Store credentials for request signing
	// The actual signing will be done per-request in the transport
	client.awsCredentials = creds
	client.awsRegion = region

	return client, nil
}
