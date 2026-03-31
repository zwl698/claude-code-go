// Package api provides API client functionality for the claude-code CLI.
// This file contains enhanced retry logic with exponential backoff and error handling.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/services/oauth"
	"claude-code-go/internal/utils"
)

// Retry constants
const (
	DefaultMaxRetries         = 10
	FloorOutputTokens         = 3000
	BaseDelayMs               = 500
	MaxDelayMs                = 32000
	Max529Retries             = 3
	PersistentMaxBackoffMs    = 5 * 60 * 1000      // 5 minutes
	PersistentResetCapMs      = 6 * 60 * 60 * 1000 // 6 hours
	HeartbeatIntervalMs       = 30000
	DefaultFastModeFallbackMs = 30 * 60 * 1000 // 30 minutes
	ShortRetryThresholdMs     = 20 * 1000      // 20 seconds
	MinCooldownMs             = 10 * 60 * 1000 // 10 minutes
)

// RetryContext contains context for retry operations.
type RetryContext struct {
	Model             string
	MaxTokensOverride *int
	ThinkingConfig    interface{}
	FastMode          bool
}

// CannotRetryError indicates that a retry is not possible.
type CannotRetryError struct {
	OriginalError error
	RetryContext  RetryContext
}

func (e *CannotRetryError) Error() string {
	if e.OriginalError != nil {
		return e.OriginalError.Error()
	}
	return "cannot retry"
}

func (e *CannotRetryError) Unwrap() error {
	return e.OriginalError
}

// FallbackTriggeredError indicates that a model fallback was triggered.
type FallbackTriggeredError struct {
	OriginalModel string
	FallbackModel string
}

func (e *FallbackTriggeredError) Error() string {
	return fmt.Sprintf("model fallback triggered: %s -> %s", e.OriginalModel, e.FallbackModel)
}

// RetryOptions contains options for retry operations.
type RetryOptions struct {
	MaxRetries                  int
	Model                       string
	FallbackModel               string
	ThinkingConfig              interface{}
	FastMode                    bool
	QuerySource                 string
	InitialConsecutive529Errors int
}

// RetryManager manages retry logic.
type RetryManager struct {
	mu                   sync.RWMutex
	consecutive529Errors int
	lastError            error
	persistentAttempt    int
}

// NewRetryManager creates a new retry manager.
func NewRetryManager() *RetryManager {
	return &RetryManager{}
}

// Foreground529RetrySources defines query sources where the user IS blocking on the result.
// These retry on 529. Everything else (summaries, titles, suggestions, classifiers)
// bails immediately during capacity cascades.
var Foreground529RetrySources = map[string]bool{
	"repl_main_thread":                         true,
	"repl_main_thread:outputStyle:custom":      true,
	"repl_main_thread:outputStyle:Explanatory": true,
	"repl_main_thread:outputStyle:Learning":    true,
	"sdk":                                      true,
	"agent:custom":                             true,
	"agent:default":                            true,
	"agent:builtin":                            true,
	"compact":                                  true,
	"hook_agent":                               true,
	"hook_prompt":                              true,
	"verification_agent":                       true,
	"side_question":                            true,
	"auto_mode":                                true,
	"bash_classifier":                          true,
}

// IsPersistentRetryEnabled checks if persistent retry mode is enabled for unattended sessions.
func IsPersistentRetryEnabled() bool {
	return utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_UNATTENDED_RETRY"))
}

// ShouldRetry529 determines if 529 errors should be retried based on query source.
func ShouldRetry529(querySource string) bool {
	// Empty source defaults to retry (conservative for untagged call paths)
	if querySource == "" {
		return true
	}
	return Foreground529RetrySources[querySource]
}

// IsTransientCapacityError checks if error is a transient capacity error (529 or 429).
func IsTransientCapacityError(err *APIError) bool {
	return Is529Error(err) || err.Status == 429
}

// GetMaxRetries returns the maximum number of retries.
func GetMaxRetries(opts *RetryOptions) int {
	if opts == nil || opts.MaxRetries == 0 {
		if maxRetries := os.Getenv("CLAUDE_CODE_MAX_RETRIES"); maxRetries != "" {
			if val, err := strconv.Atoi(maxRetries); err == nil {
				return val
			}
		}
		return DefaultMaxRetries
	}
	return opts.MaxRetries
}

// GetRetryDelay calculates the delay before the next retry.
func GetRetryDelay(attempt int, retryAfterHeader string, maxDelayMs int) time.Duration {
	if maxDelayMs == 0 {
		maxDelayMs = MaxDelayMs
	}

	// Check retry-after header first
	if retryAfterHeader != "" {
		if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	// Exponential backoff with jitter
	baseDelay := BaseDelayMs * (1 << uint(attempt-1))
	if baseDelay > maxDelayMs {
		baseDelay = maxDelayMs
	}

	// Add jitter (0-25% of base delay)
	jitter := rand.Intn(baseDelay/4 + 1)
	return time.Duration(baseDelay+jitter) * time.Millisecond
}

// ShouldRetry determines if an error should be retried.
func ShouldRetry(err error, opts *RetryOptions) bool {
	if err == nil {
		return false
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		// Check for network/connection errors
		errStr := err.Error()
		return strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "connection refused")
	}

	// Never retry mock errors
	if IsMockRateLimitError(apiErr) {
		return false
	}

	// Check for overloaded errors (529 or message content)
	if Is529Error(apiErr) {
		return true
	}

	// Check for max tokens context overflow
	if ParseMaxTokensContextOverflowError(apiErr) != nil {
		return true
	}

	// Check x-should-retry header
	shouldRetryHeader := apiErr.Headers["X-Should-Retry"]
	if shouldRetryHeader == "true" {
		// For subscription users, check if enterprise
		// Enterprise users can retry because they typically use PAYG
		return true
	}

	if shouldRetryHeader == "false" {
		// Ant users can ignore for 5xx errors
		if os.Getenv("USER_TYPE") == "ant" && apiErr.Status >= 500 {
			return true
		}
		return false
	}

	// Request timeout
	if apiErr.Status == 408 {
		return true
	}

	// Lock timeout
	if apiErr.Status == 409 {
		return true
	}

	// Rate limit (429)
	if apiErr.Status == 429 {
		return true
	}

	// Authentication errors - clear cache and retry
	if apiErr.Status == 401 {
		return true
	}

	// OAuth token revoked
	if IsOAuthTokenRevokedError(apiErr) {
		return true
	}

	// Server errors (5xx)
	if apiErr.Status >= 500 {
		return true
	}

	return false
}

// Is529Error checks if an error is a 529 (overloaded) error.
func Is529Error(err *APIError) bool {
	if err == nil {
		return false
	}

	// Check status code
	if err.Status == 529 {
		return true
	}

	// Check message content (SDK sometimes fails to pass 529 status during streaming)
	if strings.Contains(err.Message, `"type":"overloaded_error"`) {
		return true
	}

	return false
}

// IsOAuthTokenRevokedError checks if an error is an OAuth token revoked error.
func IsOAuthTokenRevokedError(err *APIError) bool {
	if err == nil {
		return false
	}
	return err.Status == 403 && strings.Contains(err.Message, "OAuth token has been revoked")
}

// IsFastModeNotEnabledError checks if an error indicates fast mode is not enabled.
func IsFastModeNotEnabledError(err *APIError) bool {
	if err == nil {
		return false
	}
	return err.Status == 400 && strings.Contains(err.Message, "Fast mode is not enabled")
}

// IsMockRateLimitError checks if this is a mock rate limit error (for testing).
func IsMockRateLimitError(err *APIError) bool {
	if err == nil {
		return false
	}
	// Check for mock rate limit marker
	return err.Headers["X-Mock-Rate-Limit"] == "true"
}

// MaxTokensOverflow contains parsed max tokens overflow error data.
type MaxTokensOverflow struct {
	InputTokens  int
	MaxTokens    int
	ContextLimit int
}

// ParseMaxTokensContextOverflowError parses max tokens context overflow errors.
func ParseMaxTokensContextOverflowError(err *APIError) *MaxTokensOverflow {
	if err == nil || err.Status != 400 || err.Message == "" {
		return nil
	}

	if !strings.Contains(err.Message, "input length and `max_tokens` exceed context limit") {
		return nil
	}

	// Parse: "input length and `max_tokens` exceed context limit: 188059 + 20000 > 200000"
	parts := strings.Split(err.Message, ":")
	if len(parts) < 2 {
		return nil
	}

	numPart := strings.TrimSpace(parts[1])
	// Parse: "188059 + 20000 > 200000"
	components := strings.Fields(numPart)
	if len(components) < 5 {
		return nil
	}

	inputTokens, err1 := strconv.Atoi(components[0])
	maxTokens, err2 := strconv.Atoi(components[2])
	contextLimit, err3 := strconv.Atoi(components[4])

	if err1 != nil || err2 != nil || err3 != nil {
		return nil
	}

	return &MaxTokensOverflow{
		InputTokens:  inputTokens,
		MaxTokens:    maxTokens,
		ContextLimit: contextLimit,
	}
}

// HandleCloudAuthError handles cloud provider authentication errors.
func HandleCloudAuthError(err error) bool {
	if err == nil {
		return false
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}

	// AWS Bedrock auth errors
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_BEDROCK")) {
		if apiErr.Status == 403 {
			// Clear AWS credentials cache
			GetAWSAuthManager().ClearCache()
			return true
		}
	}

	// Google Vertex auth errors
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_USE_VERTEX")) {
		if apiErr.Status == 401 {
			// Clear GCP credentials cache
			GetGoogleAuthManager().ClearCache()
			return true
		}
	}

	return false
}

// GetRetryAfter extracts the retry-after value from an error.
func GetRetryAfter(err error) string {
	apiErr, ok := err.(*APIError)
	if !ok {
		return ""
	}

	// Check header
	if retryAfter := apiErr.Headers["Retry-After"]; retryAfter != "" {
		return retryAfter
	}

	return ""
}

// GetRetryAfterMs extracts the retry-after value in milliseconds.
func GetRetryAfterMs(err error) int {
	retryAfter := GetRetryAfter(err)
	if retryAfter == "" {
		return 0
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return 0
	}

	return seconds * 1000
}

// GetRateLimitResetDelayMs extracts the rate limit reset delay from headers.
// Window-based limits (e.g. 5hr Max/Pro) include a reset timestamp.
func GetRateLimitResetDelayMs(err *APIError) int {
	if err == nil {
		return 0
	}

	resetHeader := err.Headers["Anthropic-Ratelimit-Unified-Reset"]
	if resetHeader == "" {
		return 0
	}

	resetUnixSec, parseErr := strconv.ParseFloat(resetHeader, 64)
	if parseErr != nil {
		return 0
	}

	delayMs := int(resetUnixSec*1000) - int(time.Now().UnixMilli())
	if delayMs <= 0 {
		return 0
	}

	// Cap at persistent reset cap
	if delayMs > PersistentResetCapMs {
		return PersistentResetCapMs
	}

	return delayMs
}

// GetRetryDelayMs returns retry delay in milliseconds.
func GetRetryDelayMs(attempt int, retryAfterHeader string, maxDelayMs int) int {
	return int(GetRetryDelay(attempt, retryAfterHeader, maxDelayMs) / time.Millisecond)
}

// ExecuteWithRetry executes an operation with retry logic.
func (c *Client) ExecuteWithRetry(ctx context.Context, operation func() error, opts *RetryOptions) error {
	if opts == nil {
		opts = &RetryOptions{}
	}

	maxRetries := GetMaxRetries(opts)
	retryManager := NewRetryManager()
	retryManager.consecutive529Errors = opts.InitialConsecutive529Errors
	var lastErr error

	for attempt := 1; attempt <= maxRetries+1; attempt++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Log error for debugging
		if apiErr, ok := err.(*APIError); ok {
			fmt.Fprintf(os.Stderr, "[API] Error (attempt %d/%d): %d %s\n",
				attempt, maxRetries+1, apiErr.Status, apiErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "[API] Error (attempt %d/%d): %v\n",
				attempt, maxRetries+1, err)
		}

		// Handle cloud auth errors
		if HandleCloudAuthError(err) {
			// Re-initialize client with fresh credentials
			if utils.GetAPIProvider() == utils.APIProviderBedrock {
				if creds, credErr := GetAWSAuthManager().RefreshAndGetAWSCredentials(ctx); credErr == nil {
					c.awsCredentials = creds
				}
			} else if utils.GetAPIProvider() == utils.APIProviderVertex {
				if token, credErr := GetGoogleAuthManager().GetAccessToken(ctx); credErr == nil {
					c.googleAccessToken = token
				}
			}
		}

		// Handle OAuth 401 errors - force token refresh
		if apiErr, ok := err.(*APIError); ok && apiErr.Status == 401 {
			// Get the OAuth client
			oauthClient := oauth.NewOAuthClient()
			tokens, _ := oauthClient.LoadOAuthTokens()
			if tokens != nil && tokens.AccessToken != "" {
				// Try to handle the 401 error by refreshing the token
				refreshed, refreshErr := oauthClient.HandleOAuth401Error(ctx, tokens.AccessToken)
				if refreshErr == nil && refreshed {
					// Update the client's auth token if it was refreshed
					if newTokens := oauthClient.GetOAuthTokens(); newTokens != nil {
						c.options.AuthToken = newTokens.AccessToken
					}
				}
			}
		}

		// Handle OAuth token revoked errors (403 with specific message)
		if revokedErr, ok := err.(*APIError); ok && IsOAuthTokenRevokedError(revokedErr) {
			oauthClient := oauth.NewOAuthClient()
			tokens, _ := oauthClient.LoadOAuthTokens()
			if tokens != nil && tokens.AccessToken != "" {
				// Force refresh the token
				oauthClient.HandleOAuth401Error(ctx, tokens.AccessToken)
				if newTokens := oauthClient.GetOAuthTokens(); newTokens != nil {
					c.options.AuthToken = newTokens.AccessToken
				}
			}
		}

		// Non-foreground sources bail immediately on 529
		if apiErr, ok := err.(*APIError); ok && Is529Error(apiErr) && !ShouldRetry529(opts.QuerySource) {
			fmt.Fprintf(os.Stderr, "[API] Background 529 error - not retrying\n")
			return &CannotRetryError{
				OriginalError: err,
				RetryContext: RetryContext{
					Model: opts.Model,
				},
			}
		}

		// Track consecutive 529 errors for fallback
		if apiErr, ok := err.(*APIError); ok && Is529Error(apiErr) {
			// Check if fallback should be triggered
			shouldCheckFallback := os.Getenv("FALLBACK_FOR_ALL_PRIMARY_MODELS") != ""
			if !shouldCheckFallback {
				// Only check for non-custom Opus models (simplified check)
				shouldCheckFallback = strings.Contains(strings.ToLower(opts.Model), "opus")
			}

			if shouldCheckFallback {
				retryManager.mu.Lock()
				retryManager.consecutive529Errors++
				consecutive := retryManager.consecutive529Errors
				retryManager.mu.Unlock()

				if consecutive >= Max529Retries && opts.FallbackModel != "" {
					fmt.Fprintf(os.Stderr, "[API] Fallback triggered after %d consecutive 529 errors: %s -> %s\n",
						consecutive, opts.Model, opts.FallbackModel)
					return &FallbackTriggeredError{
						OriginalModel: opts.Model,
						FallbackModel: opts.FallbackModel,
					}
				}
			}
		}

		// Check if we should retry
		persistent := IsPersistentRetryEnabled()
		if apiErr, ok := err.(*APIError); ok {
			persistent = persistent && IsTransientCapacityError(apiErr)
		}

		if attempt > maxRetries && !persistent {
			return &CannotRetryError{
				OriginalError: err,
				RetryContext: RetryContext{
					Model: opts.Model,
				},
			}
		}

		if !ShouldRetry(err, opts) {
			return &CannotRetryError{
				OriginalError: err,
				RetryContext: RetryContext{
					Model: opts.Model,
				},
			}
		}

		// Handle max tokens overflow
		if apiErr, ok := err.(*APIError); ok {
			if overflow := ParseMaxTokensContextOverflowError(apiErr); overflow != nil {
				safetyBuffer := 1000
				availableContext := overflow.ContextLimit - overflow.InputTokens - safetyBuffer
				if availableContext < FloorOutputTokens {
					availableContext = FloorOutputTokens
				}
				opts.ThinkingConfig = availableContext
				fmt.Fprintf(os.Stderr, "[API] Adjusting max tokens due to context overflow: available=%d\n", availableContext)
				continue
			}
		}

		// Calculate retry delay
		retryAfter := GetRetryAfter(err)
		var delayMs int

		if persistent {
			retryManager.mu.Lock()
			retryManager.persistentAttempt++
			persistentAttempt := retryManager.persistentAttempt
			retryManager.mu.Unlock()

			apiErr, isAPIErr := err.(*APIError)
			if isAPIErr && apiErr.Status == 429 {
				// Check for rate limit reset header
				resetDelay := GetRateLimitResetDelayMs(apiErr)
				if resetDelay > 0 {
					delayMs = resetDelay
				} else {
					delayMs = GetRetryDelayMs(persistentAttempt, retryAfter, PersistentMaxBackoffMs)
					if delayMs > PersistentResetCapMs {
						delayMs = PersistentResetCapMs
					}
				}
			} else {
				delayMs = GetRetryDelayMs(persistentAttempt, retryAfter, PersistentMaxBackoffMs)
				if delayMs > PersistentResetCapMs {
					delayMs = PersistentResetCapMs
				}
			}
		} else {
			delayMs = GetRetryDelayMs(attempt, retryAfter, MaxDelayMs)
		}

		// Log retry event
		fmt.Fprintf(os.Stderr, "[API] Retry attempt %d/%d after %dms\n",
			attempt, maxRetries, delayMs)

		// Wait before retry with heartbeat for persistent mode
		if persistent && delayMs > HeartbeatIntervalMs {
			// Chunk long sleeps for heartbeat
			remaining := delayMs
			for remaining > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				chunk := remaining
				if chunk > HeartbeatIntervalMs {
					chunk = HeartbeatIntervalMs
				}

				time.Sleep(time.Duration(chunk) * time.Millisecond)
				remaining -= chunk

				// Heartbeat log
				fmt.Fprintf(os.Stderr, "[API] Persistent retry heartbeat - remaining: %dms\n", remaining)
			}
		} else {
			// Normal sleep
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(delayMs) * time.Millisecond):
			}
		}

		// Clamp attempt for persistent mode so loop never terminates
		if persistent && attempt >= maxRetries {
			attempt = maxRetries
		}
	}

	return &CannotRetryError{
		OriginalError: lastErr,
		RetryContext: RetryContext{
			Model: opts.Model,
		},
	}
}

// doRequestWithRetryEnhanced executes an HTTP request with enhanced retry logic.
func (c *Client) doRequestWithRetryEnhanced(ctx context.Context, req *http.Request, result interface{}, opts *RetryOptions) error {
	if opts == nil {
		opts = &RetryOptions{
			Model: c.options.Model,
		}
	}

	return c.ExecuteWithRetry(ctx, func() error {
		// Clone the request for retry
		reqClone := cloneRequest(req)

		// Execute the request
		var httpResp *http.Response
		var err error

		if c.options.FetchOverride != nil {
			httpResp, err = c.options.FetchOverride(reqClone)
		} else {
			httpResp, err = c.httpClient.Do(reqClone)
		}

		if err != nil {
			return err
		}
		defer httpResp.Body.Close()

		// Read response body
		bodyBytes, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return err
		}

		// Check for errors
		if httpResp.StatusCode >= 400 {
			apiErr := &APIError{
				Status:  httpResp.StatusCode,
				Message: string(bodyBytes),
				Headers: make(map[string]string),
			}
			for k, v := range httpResp.Header {
				if len(v) > 0 {
					apiErr.Headers[k] = v[0]
				}
			}
			return apiErr
		}

		// Parse response
		if result != nil && len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}, opts)
}

// cloneRequest creates a copy of an HTTP request for retry.
func cloneRequest(req *http.Request) *http.Request {
	// Create new request with same method, URL, and body
	reqClone := req.Clone(req.Context())

	// Copy headers
	for k, v := range req.Header {
		reqClone.Header[k] = v
	}

	// Copy body if present
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		reqClone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return reqClone
}
