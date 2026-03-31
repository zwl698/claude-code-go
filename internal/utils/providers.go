// Package utils provides utility functions for the claude-code CLI.
// This file contains API provider detection and configuration.
package utils

import (
	"net/url"
	"os"
	"strings"
)

// APIProvider represents the API provider type.
type APIProvider string

const (
	APIProviderFirstParty APIProvider = "firstParty"
	APIProviderBedrock    APIProvider = "bedrock"
	APIProviderVertex     APIProvider = "vertex"
	APIProviderFoundry    APIProvider = "foundry"
)

// GetAPIProvider returns the current API provider based on environment variables.
func GetAPIProvider() APIProvider {
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_BEDROCK")) {
		return APIProviderBedrock
	}
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_VERTEX")) {
		return APIProviderVertex
	}
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_FOUNDRY")) {
		return APIProviderFoundry
	}
	return APIProviderFirstParty
}

// IsFirstPartyAnthropicBaseURL checks if ANTHROPIC_BASE_URL is a first-party Anthropic API URL.
// Returns true if not set (default API) or points to api.anthropic.com
// (or api-staging.anthropic.com for ant users).
func IsFirstPartyAnthropicBaseURL() bool {
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		return true
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	allowedHosts := []string{"api.anthropic.com"}
	if os.Getenv("USER_TYPE") == "ant" {
		allowedHosts = append(allowedHosts, "api-staging.anthropic.com")
	}

	for _, host := range allowedHosts {
		if parsedURL.Host == host {
			return true
		}
	}
	return false
}

// GetAWSRegion returns the AWS region for Bedrock.
func GetAWSRegion() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}
	return "us-east-1"
}

// GetVertexRegionForModel returns the Vertex AI region for a specific model.
func GetVertexRegionForModel(model string) string {
	// Model-specific region overrides
	switch {
	case strings.Contains(model, "haiku"):
		if region := os.Getenv("VERTEX_REGION_CLAUDE_3_5_HAIKU"); region != "" {
			return region
		}
		if region := os.Getenv("VERTEX_REGION_CLAUDE_HAIKU_4_5"); region != "" {
			return region
		}
	case strings.Contains(model, "sonnet"):
		if region := os.Getenv("VERTEX_REGION_CLAUDE_3_5_SONNET"); region != "" {
			return region
		}
		if region := os.Getenv("VERTEX_REGION_CLAUDE_3_7_SONNET"); region != "" {
			return region
		}
	}

	// Global region fallback
	if region := os.Getenv("CLOUD_ML_REGION"); region != "" {
		return region
	}

	// Default region
	return "us-east5"
}

// IsEnvTruthy checks if an environment variable value is truthy.
func IsEnvTruthy(value string) bool {
	lower := strings.ToLower(value)
	return lower == "true" || lower == "1" || lower == "yes"
}
