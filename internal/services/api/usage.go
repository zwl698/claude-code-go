package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/constants"
	"claude-code-go/internal/utils"
)

// =============================================================================
// Types
// =============================================================================

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

// UsageInfo represents combined usage information for display
type UsageInfo struct {
	Utilization      *Utilization
	IsClaudeAISub    bool
	HasProfileScope  bool
	FetchedAt        time.Time
	CachedUntil      time.Time
	SubscriptionType string
	IsInternalUser   bool
}

// =============================================================================
// Usage Service
// =============================================================================

// UsageService manages API usage queries with caching.
type UsageService struct {
	mu            sync.RWMutex
	cachedUsage   *UsageInfo
	cacheDuration time.Duration
	httpClient    *http.Client
}

// NewUsageService creates a new usage service.
func NewUsageService() *UsageService {
	return &UsageService{
		cacheDuration: 5 * time.Minute,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// =============================================================================
// Subscription and Scope Detection
// =============================================================================

// SubscriptionType represents the type of Claude subscription.
type SubscriptionType string

const (
	SubscriptionTypeNone       SubscriptionType = "none"
	SubscriptionTypeFree       SubscriptionType = "free"
	SubscriptionTypePro        SubscriptionType = "pro"
	SubscriptionTypeTeam       SubscriptionType = "team"
	SubscriptionTypeEnterprise SubscriptionType = "enterprise"
)

// DetectSubscriptionType detects the user's subscription type from the token.
// This is typically done by decoding the JWT token or checking API endpoints.
func DetectSubscriptionType(oauthToken string) SubscriptionType {
	if oauthToken == "" {
		return SubscriptionTypeNone
	}

	// Try to decode JWT token to check subscription type
	// JWT tokens have 3 parts separated by dots
	parts := strings.Split(oauthToken, ".")
	if len(parts) != 3 {
		return SubscriptionTypeFree
	}

	// For now, we'll check via API call or assume pro
	// In production, this would decode the JWT claims
	return SubscriptionTypePro
}

// HasProfileScope checks if the OAuth token has the profile scope.
// The profile scope is required to access usage endpoints.
func HasProfileScope(oauthToken string) bool {
	if oauthToken == "" {
		return false
	}

	// Profile scope is typically included in the token
	// This checks if the token was obtained with the profile scope
	// In production, this would decode the JWT and check scopes
	return true // Assume profile scope is present for OAuth tokens
}

// IsClaudeAISubscriber checks if the user is a Claude AI subscriber.
// This requires checking the user's subscription status via API.
func IsClaudeAISubscriber(oauthToken string) bool {
	if oauthToken == "" {
		return false
	}

	// Check subscription type
	subType := DetectSubscriptionType(oauthToken)
	return subType == SubscriptionTypePro ||
		subType == SubscriptionTypeTeam ||
		subType == SubscriptionTypeEnterprise
}

// =============================================================================
// API Methods
// =============================================================================

// FetchUtilization fetches utilization data from the API.
func FetchUtilization(apiKey, oauthToken string, isSubscriber bool) (*Utilization, error) {
	// Check if user is a Claude AI subscriber with profile scope
	if oauthToken != "" && !HasProfileScope(oauthToken) {
		return &Utilization{}, nil
	}

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

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		// Return empty utilization for non-subscribers
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			return &Utilization{}, nil
		}
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var utilization Utilization
	if err := json.Unmarshal(body, &utilization); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &utilization, nil
}

// FetchUtilizationWithContext fetches utilization data with context support.
func FetchUtilizationWithContext(ctx context.Context, apiKey, oauthToken string, isSubscriber bool) (*Utilization, error) {
	// Check if user is a Claude AI subscriber with profile scope
	if oauthToken != "" && !HasProfileScope(oauthToken) {
		return &Utilization{}, nil
	}

	// Get auth headers
	authResult := utils.GetAuthHeaders(apiKey, oauthToken, isSubscriber)
	if authResult.Error != "" {
		return nil, fmt.Errorf("auth error: %s", authResult.Error)
	}

	// Build request
	oauthConfig := constants.GetOAuthConfig()
	url := fmt.Sprintf("%s/api/oauth/usage", oauthConfig.BaseAPIURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		Timeout: 10 * time.Second,
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

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			return &Utilization{}, nil
		}
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var utilization Utilization
	if err := json.Unmarshal(body, &utilization); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &utilization, nil
}

// GetUsageWithCache returns cached usage or fetches new data.
func (s *UsageService) GetUsageWithCache(ctx context.Context, apiKey, oauthToken string) (*UsageInfo, error) {
	s.mu.RLock()
	if s.cachedUsage != nil && time.Now().Before(s.cachedUsage.CachedUntil) {
		defer s.mu.RUnlock()
		return s.cachedUsage, nil
	}
	s.mu.RUnlock()

	// Fetch new data
	return s.RefreshCache(ctx, apiKey, oauthToken)
}

// RefreshCache forces a refresh of the usage cache.
func (s *UsageService) RefreshCache(ctx context.Context, apiKey, oauthToken string) (*UsageInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine subscription status
	isSubscriber := IsClaudeAISubscriber(oauthToken)
	hasProfileScope := HasProfileScope(oauthToken)
	subType := DetectSubscriptionType(oauthToken)

	// Fetch utilization
	utilization, err := FetchUtilizationWithContext(ctx, apiKey, oauthToken, isSubscriber)
	if err != nil {
		return nil, err
	}

	// Create usage info
	info := &UsageInfo{
		Utilization:      utilization,
		IsClaudeAISub:    isSubscriber,
		HasProfileScope:  hasProfileScope,
		FetchedAt:        time.Now(),
		CachedUntil:      time.Now().Add(s.cacheDuration),
		SubscriptionType: string(subType),
		IsInternalUser:   utils.IsInternalUser(),
	}

	s.cachedUsage = info
	return info, nil
}

// ClearCache clears the usage cache.
func (s *UsageService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedUsage = nil
}

// =============================================================================
// Usage Formatting
// =============================================================================

// FormatUsage returns a human-readable string of usage information.
func FormatUsage(info *UsageInfo) string {
	if info == nil || info.Utilization == nil {
		return "Usage information not available"
	}

	var sb strings.Builder

	// Check subscription status
	if !info.IsClaudeAISub {
		sb.WriteString("Not a Claude AI subscriber\n")
		sb.WriteString("Upgrade to Pro for usage tracking\n")
		return sb.String()
	}

	// Show rate limits
	util := info.Utilization

	if util.SevenDay != nil {
		sb.WriteString(fmt.Sprintf("7-Day Limit: %d%%", *util.SevenDay.Utilization))
		if util.SevenDay.ResetsAt != nil {
			resetsAt, err := time.Parse(time.RFC3339, *util.SevenDay.ResetsAt)
			if err == nil {
				sb.WriteString(fmt.Sprintf(" (resets %s)", formatTimeUntil(resetsAt)))
			}
		}
		sb.WriteString("\n")
	}

	if util.SevenDayOpus != nil {
		sb.WriteString(fmt.Sprintf("Opus Limit: %d%%", *util.SevenDayOpus.Utilization))
		if util.SevenDayOpus.ResetsAt != nil {
			resetsAt, err := time.Parse(time.RFC3339, *util.SevenDayOpus.ResetsAt)
			if err == nil {
				sb.WriteString(fmt.Sprintf(" (resets %s)", formatTimeUntil(resetsAt)))
			}
		}
		sb.WriteString("\n")
	}

	if util.SevenDaySonnet != nil {
		sb.WriteString(fmt.Sprintf("Sonnet Limit: %d%%", *util.SevenDaySonnet.Utilization))
		if util.SevenDaySonnet.ResetsAt != nil {
			resetsAt, err := time.Parse(time.RFC3339, *util.SevenDaySonnet.ResetsAt)
			if err == nil {
				sb.WriteString(fmt.Sprintf(" (resets %s)", formatTimeUntil(resetsAt)))
			}
		}
		sb.WriteString("\n")
	}

	// Show extra usage if enabled
	if util.ExtraUsage != nil && util.ExtraUsage.IsEnabled {
		sb.WriteString(fmt.Sprintf("Extra Usage: %d/%d credits\n",
			*util.ExtraUsage.UsedCredits, *util.ExtraUsage.MonthlyLimit))
	}

	return sb.String()
}

// formatTimeUntil formats a time as a relative duration string.
func formatTimeUntil(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		return "already reset"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("in %d days", days)
	}
	if hours > 0 {
		return fmt.Sprintf("in %d hours", hours)
	}
	return fmt.Sprintf("in %d minutes", minutes)
}

// =============================================================================
// Rate Limit Status Helpers
// =============================================================================

// IsRateLimited checks if any rate limit is at or near capacity.
func IsRateLimited(util *Utilization, threshold int) bool {
	if util == nil {
		return false
	}

	if util.SevenDay != nil && util.SevenDay.Utilization != nil {
		if *util.SevenDay.Utilization >= threshold {
			return true
		}
	}

	if util.SevenDayOpus != nil && util.SevenDayOpus.Utilization != nil {
		if *util.SevenDayOpus.Utilization >= threshold {
			return true
		}
	}

	if util.SevenDaySonnet != nil && util.SevenDaySonnet.Utilization != nil {
		if *util.SevenDaySonnet.Utilization >= threshold {
			return true
		}
	}

	return false
}

// GetHighestUtilization returns the highest utilization percentage across all limits.
func GetHighestUtilization(util *Utilization) int {
	if util == nil {
		return 0
	}

	highest := 0

	if util.SevenDay != nil && util.SevenDay.Utilization != nil {
		if *util.SevenDay.Utilization > highest {
			highest = *util.SevenDay.Utilization
		}
	}

	if util.SevenDayOpus != nil && util.SevenDayOpus.Utilization != nil {
		if *util.SevenDayOpus.Utilization > highest {
			highest = *util.SevenDayOpus.Utilization
		}
	}

	if util.SevenDaySonnet != nil && util.SevenDaySonnet.Utilization != nil {
		if *util.SevenDaySonnet.Utilization > highest {
			highest = *util.SevenDaySonnet.Utilization
		}
	}

	return highest
}

// GetNearestReset returns the nearest reset time across all limits.
func GetNearestReset(util *Utilization) *time.Time {
	if util == nil {
		return nil
	}

	var nearest *time.Time

	checkReset := func(resetStr *string) {
		if resetStr == nil {
			return
		}
		t, err := time.Parse(time.RFC3339, *resetStr)
		if err != nil {
			return
		}
		if nearest == nil || t.Before(*nearest) {
			nearest = &t
		}
	}

	if util.SevenDay != nil {
		checkReset(util.SevenDay.ResetsAt)
	}
	if util.SevenDayOpus != nil {
		checkReset(util.SevenDayOpus.ResetsAt)
	}
	if util.SevenDaySonnet != nil {
		checkReset(util.SevenDaySonnet.ResetsAt)
	}

	return nearest
}
