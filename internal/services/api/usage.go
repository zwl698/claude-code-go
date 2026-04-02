package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"claude-code-go/internal/constants"
	"claude-code-go/internal/utils"
)

// RateLimit represents rate limit information
type RateLimit struct {
	Utilization *int    `json:"utilization"` // a percentage from 0 to 100
	ResetsAt    *string `json:"resets_at"`   // ISO 8601 timestamp
}

// ExtraUsage represents extra usage information
type ExtraUsage struct {
	IsEnabled    bool `json:"is_enabled"`
	MonthlyLimit *int `json:"monthly_limit"`
	UsedCredits  *int `json:"used_credits"`
	Utilization  *int `json:"utilization"`
}

// Utilization represents API utilization data
type Utilization struct {
	FiveHour          *RateLimit  `json:"five_hour,omitempty"`
	SevenDay          *RateLimit  `json:"seven_day,omitempty"`
	SevenDayOAuthApps *RateLimit  `json:"seven_day_oauth_apps,omitempty"`
	SevenDayOpus      *RateLimit  `json:"seven_day_opus,omitempty"`
	SevenDaySonnet    *RateLimit  `json:"seven_day_sonnet,omitempty"`
	ExtraUsage        *ExtraUsage `json:"extra_usage,omitempty"`
}

// FetchUtilization fetches utilization data from the API
// TODO: This is a simplified version. Full implementation requires:
// - isClaudeAISubscriber() check
// - hasProfileScope() check
// - Global auth manager integration
func FetchUtilization(apiKey, oauthToken string, isSubscriber bool) (*Utilization, error) {
	// Check if user is a Claude AI subscriber with profile scope
	// if !isSubscriber {
	// 	return &Utilization{}, nil
	// }

	// Get auth headers
	authResult := utils.GetAuthHeaders(apiKey, oauthToken, isSubscriber)
	if authResult.Error != "" {
		return nil, fmt.Errorf("auth error: %s", authResult.Error)
	}

	// Build request
	oauthConfig := constants.GetOAuthConfig()
	url := fmt.Sprintf("%s/api/oauth/usage", oauthConfig.BaseAPIURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", utils.GetClaudeCodeUserAgent())
	for key, value := range authResult.Headers {
		req.Header.Set(key, value)
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var utilization Utilization
	if err := json.Unmarshal(body, &utilization); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &utilization, nil
}
