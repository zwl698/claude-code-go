// Package api provides API client functionality for the claude-code CLI.
// This file contains API error handling translated from TypeScript.
package api

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// API error message constants
const (
	APIErrorMessagePrefix                  = "API Error"
	PromptTooLongErrorMessage              = "Prompt is too long"
	CreditBalanceTooLowErrorMessage        = "Credit balance is too low"
	InvalidAPIKeyErrorMessage              = "Not logged in · Please run /login"
	InvalidAPIKeyErrorMessageExternal      = "Invalid API key · Fix external API key"
	OrgDisabledErrorMessageEnvKeyWithOAuth = "Your ANTHROPIC_API_KEY belongs to a disabled organization · Unset the environment variable to use your subscription instead"
	OrgDisabledErrorMessageEnvKey          = "Your ANTHROPIC_API_KEY belongs to a disabled organization · Update or unset the environment variable"
	TokenRevokedErrorMessage               = "OAuth token revoked · Please run /login"
	CCRAuthErrorMessage                    = "Authentication error · This may be a temporary network issue, please try again"
	Repeated529ErrorMessage                = "Repeated 529 Overloaded errors"
	CustomOffSwitchMessage                 = "Opus is experiencing high load, please use /model to switch to Sonnet"
	APITimeoutErrorMessage                 = "Request timed out"
	OAuthOrgNotAllowedErrorMessage         = "Your account does not have access to Claude Code. Please run /login."
)

// APIError represents an API error with status code and message.
type APIError struct {
	Status  int               `json:"status"`
	Message string            `json:"message"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error: %d %s", e.Status, e.Message)
}

// APIConnectionError represents a connection error.
type APIConnectionError struct {
	Message string `json:"message"`
}

func (e *APIConnectionError) Error() string {
	return e.Message
}

// APIConnectionTimeoutError represents a timeout error.
type APIConnectionTimeoutError struct {
	Message string `json:"message"`
}

func (e *APIConnectionTimeoutError) Error() string {
	return e.Message
}

// ImageSizeError represents an image size validation error.
type ImageSizeError struct {
	Message string `json:"message"`
}

func (e *ImageSizeError) Error() string {
	return e.Message
}

// ImageResizeError represents an image resize error.
type ImageResizeError struct {
	Message string `json:"message"`
}

func (e *ImageResizeError) Error() string {
	return e.Message
}

// AssistantMessageError represents the type of error in an assistant message.
type AssistantMessageError string

const (
	AssistantMessageErrorUnknown              AssistantMessageError = "unknown"
	AssistantMessageErrorRateLimit            AssistantMessageError = "rate_limit"
	AssistantMessageErrorInvalidRequest       AssistantMessageError = "invalid_request"
	AssistantMessageErrorBillingError         AssistantMessageError = "billing_error"
	AssistantMessageErrorAuthenticationFailed AssistantMessageError = "authentication_failed"
)

// AssistantAPIErrorMessage represents an API error message.
type AssistantAPIErrorMessage struct {
	Content      string                `json:"content"`
	Error        AssistantMessageError `json:"error,omitempty"`
	ErrorDetails string                `json:"errorDetails,omitempty"`
}

// StartsWithAPIErrorPrefix checks if text starts with the API error prefix.
func StartsWithAPIErrorPrefix(text string) bool {
	return strings.HasPrefix(text, APIErrorMessagePrefix) ||
		strings.HasPrefix(text, "Please run /login · "+APIErrorMessagePrefix)
}

// IsPromptTooLongMessage checks if an assistant message is a prompt-too-long error.
func IsPromptTooLongMessage(msg *AssistantAPIErrorMessage) bool {
	if msg == nil {
		return false
	}
	return strings.HasPrefix(msg.Content, PromptTooLongErrorMessage)
}

// ParsePromptTooLongTokenCounts extracts token counts from a prompt-too-long error message.
func ParsePromptTooLongTokenCounts(rawMessage string) (actualTokens, limitTokens int) {
	// Match pattern like "prompt is too long: 137500 tokens > 135000 maximum"
	re := regexp.MustCompile(`(?i)prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)`)
	matches := re.FindStringSubmatch(rawMessage)
	if len(matches) >= 3 {
		actual, _ := strconv.Atoi(matches[1])
		limit, _ := strconv.Atoi(matches[2])
		return actual, limit
	}
	return 0, 0
}

// GetPromptTooLongTokenGap returns the number of tokens over the limit.
func GetPromptTooLongTokenGap(msg *AssistantAPIErrorMessage) int {
	if msg == nil || !IsPromptTooLongMessage(msg) || msg.ErrorDetails == "" {
		return 0
	}
	actual, limit := ParsePromptTooLongTokenCounts(msg.ErrorDetails)
	if actual > limit {
		return actual - limit
	}
	return 0
}

// IsMediaSizeError checks if the raw error is a media size rejection.
func IsMediaSizeError(raw string) bool {
	return (strings.Contains(raw, "image exceeds") && strings.Contains(raw, "maximum")) ||
		(strings.Contains(raw, "image dimensions exceed") && strings.Contains(raw, "many-image")) ||
		regexp.MustCompile(`maximum of \d+ PDF pages`).MatchString(raw)
}

// IsMediaSizeErrorMessage checks if an assistant message is a media size error.
func IsMediaSizeErrorMessage(msg *AssistantAPIErrorMessage) bool {
	return msg != nil && msg.ErrorDetails != "" && IsMediaSizeError(msg.ErrorDetails)
}

// GetAssistantMessageFromError converts an error to an assistant API error message.
func GetAssistantMessageFromError(err error, model string, isNonInteractive bool) *AssistantAPIErrorMessage {
	if err == nil {
		return nil
	}

	// Check for timeout errors
	var timeoutErr *APIConnectionTimeoutError
	var connErr *APIConnectionError
	if errors.As(err, &timeoutErr) ||
		(errors.As(err, &connErr) && strings.Contains(strings.ToLower(connErr.Message), "timeout")) {
		return &AssistantAPIErrorMessage{
			Content: APITimeoutErrorMessage,
			Error:   AssistantMessageErrorUnknown,
		}
	}

	// Check for image size/resize errors
	var imgSizeErr *ImageSizeError
	var imgResizeErr *ImageResizeError
	if errors.As(err, &imgSizeErr) || errors.As(err, &imgResizeErr) {
		return &AssistantAPIErrorMessage{
			Content: GetImageTooLargeErrorMessage(isNonInteractive),
		}
	}

	// Check for emergency capacity off switch
	if strings.Contains(err.Error(), CustomOffSwitchMessage) {
		return &AssistantAPIErrorMessage{
			Content: CustomOffSwitchMessage,
			Error:   AssistantMessageErrorRateLimit,
		}
	}

	// Check for API errors
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return handleAPIError(apiErr, model, isNonInteractive)
	}

	// Generic error handling
	return &AssistantAPIErrorMessage{
		Content: APIErrorMessagePrefix + ": " + err.Error(),
		Error:   AssistantMessageErrorUnknown,
	}
}

// handleAPIError handles specific API error status codes.
func handleAPIError(err *APIError, model string, isNonInteractive bool) *AssistantAPIErrorMessage {
	switch err.Status {
	case 429:
		return handleRateLimitError(err, model, isNonInteractive)
	case 400:
		return handleBadRequestError(err, model, isNonInteractive)
	case 401, 403:
		return handleAuthError(err, isNonInteractive)
	case 404:
		return handleNotFoundError(err, model, isNonInteractive)
	case 413:
		return &AssistantAPIErrorMessage{
			Content: GetRequestTooLargeErrorMessage(isNonInteractive),
			Error:   AssistantMessageErrorInvalidRequest,
		}
	case 529:
		return &AssistantAPIErrorMessage{
			Content: APIErrorMessagePrefix + ": Server overloaded. Please retry.",
			Error:   AssistantMessageErrorRateLimit,
		}
	}

	if err.Status >= 500 {
		return &AssistantAPIErrorMessage{
			Content: APIErrorMessagePrefix + ": Server error (" + strconv.Itoa(err.Status) + ")",
			Error:   AssistantMessageErrorUnknown,
		}
	}

	return &AssistantAPIErrorMessage{
		Content: APIErrorMessagePrefix + ": " + err.Message,
		Error:   AssistantMessageErrorUnknown,
	}
}

// handleRateLimitError handles 429 rate limit errors.
func handleRateLimitError(err *APIError, model string, isNonInteractive bool) *AssistantAPIErrorMessage {
	// Check for extra usage requirement
	if strings.Contains(err.Message, "Extra usage is required for long context") {
		hint := "enable extra usage at claude.ai/settings/usage, or use --model to switch to standard context"
		if !isNonInteractive {
			hint = "run /extra-usage to enable, or /model to switch to standard context"
		}
		return &AssistantAPIErrorMessage{
			Content: APIErrorMessagePrefix + ": Extra usage is required for 1M context · " + hint,
			Error:   AssistantMessageErrorRateLimit,
		}
	}

	// Extract inner message if JSON-stringified
	stripped := strings.TrimPrefix(err.Message, "429 ")
	innerMsg := extractJSONMessage(stripped)
	if innerMsg == "" {
		innerMsg = stripped
	}

	return &AssistantAPIErrorMessage{
		Content: APIErrorMessagePrefix + ": Request rejected (429) · " + innerMsg,
		Error:   AssistantMessageErrorRateLimit,
	}
}

// handleBadRequestError handles 400 bad request errors.
func handleBadRequestError(err *APIError, model string, isNonInteractive bool) *AssistantAPIErrorMessage {
	msg := err.Message

	// Prompt too long
	if strings.Contains(strings.ToLower(msg), "prompt is too long") {
		return &AssistantAPIErrorMessage{
			Content:      PromptTooLongErrorMessage,
			Error:        AssistantMessageErrorInvalidRequest,
			ErrorDetails: msg,
		}
	}

	// PDF errors
	if regexp.MustCompile(`maximum of \d+ PDF pages`).MatchString(msg) {
		return &AssistantAPIErrorMessage{
			Content:      GetPdfTooLargeErrorMessage(isNonInteractive),
			Error:        AssistantMessageErrorInvalidRequest,
			ErrorDetails: msg,
		}
	}

	if strings.Contains(msg, "The PDF specified is password protected") {
		return &AssistantAPIErrorMessage{
			Content: GetPdfPasswordProtectedErrorMessage(isNonInteractive),
			Error:   AssistantMessageErrorInvalidRequest,
		}
	}

	if strings.Contains(msg, "The PDF specified was not valid") {
		return &AssistantAPIErrorMessage{
			Content: GetPdfInvalidErrorMessage(isNonInteractive),
			Error:   AssistantMessageErrorInvalidRequest,
		}
	}

	// Image errors
	if strings.Contains(msg, "image exceeds") && strings.Contains(msg, "maximum") {
		return &AssistantAPIErrorMessage{
			Content:      GetImageTooLargeErrorMessage(isNonInteractive),
			ErrorDetails: msg,
		}
	}

	if strings.Contains(msg, "image dimensions exceed") && strings.Contains(msg, "many-image") {
		content := "An image in the conversation exceeds the dimension limit for many-image requests (2000px)."
		if isNonInteractive {
			content += " Start a new session with fewer images."
		} else {
			content += " Run /compact to remove old images from context, or start a new session."
		}
		return &AssistantAPIErrorMessage{
			Content:      content,
			Error:        AssistantMessageErrorInvalidRequest,
			ErrorDetails: msg,
		}
	}

	// Tool use errors
	if strings.Contains(msg, "`tool_use` ids were found without `tool_result` blocks immediately after") {
		content := "API Error: 400 due to tool use concurrency issues."
		if !isNonInteractive {
			content += " Run /rewind to recover the conversation."
		}
		return &AssistantAPIErrorMessage{
			Content: content,
			Error:   AssistantMessageErrorInvalidRequest,
		}
	}

	if strings.Contains(msg, "`tool_use` ids must be unique") {
		content := "API Error: 400 duplicate tool_use ID in conversation history."
		if !isNonInteractive {
			content += " Run /rewind to recover the conversation."
		}
		return &AssistantAPIErrorMessage{
			Content:      content,
			Error:        AssistantMessageErrorInvalidRequest,
			ErrorDetails: msg,
		}
	}

	return &AssistantAPIErrorMessage{
		Content: APIErrorMessagePrefix + ": " + msg,
		Error:   AssistantMessageErrorInvalidRequest,
	}
}

// handleAuthError handles 401/403 authentication errors.
func handleAuthError(err *APIError, isNonInteractive bool) *AssistantAPIErrorMessage {
	msg := err.Message

	// OAuth token revoked
	if strings.Contains(msg, "OAuth token has been revoked") {
		return &AssistantAPIErrorMessage{
			Error:   AssistantMessageErrorAuthenticationFailed,
			Content: GetTokenRevokedErrorMessage(isNonInteractive),
		}
	}

	// OAuth org not allowed
	if strings.Contains(msg, "OAuth authentication is currently not allowed for this organization") {
		return &AssistantAPIErrorMessage{
			Error:   AssistantMessageErrorAuthenticationFailed,
			Content: GetOAuthOrgNotAllowedErrorMessage(isNonInteractive),
		}
	}

	// Generic auth error
	content := "Please run /login · " + APIErrorMessagePrefix + ": " + msg
	if isNonInteractive {
		content = "Failed to authenticate. " + APIErrorMessagePrefix + ": " + msg
	}
	return &AssistantAPIErrorMessage{
		Error:   AssistantMessageErrorAuthenticationFailed,
		Content: content,
	}
}

// handleNotFoundError handles 404 not found errors.
func handleNotFoundError(err *APIError, model string, isNonInteractive bool) *AssistantAPIErrorMessage {
	switchCmd := "--model"
	if !isNonInteractive {
		switchCmd = "/model"
	}
	return &AssistantAPIErrorMessage{
		Content: fmt.Sprintf("There's an issue with the selected model (%s). Run %s to pick a different model.", model, switchCmd),
		Error:   AssistantMessageErrorInvalidRequest,
	}
}

// extractJSONMessage extracts the message field from a JSON string.
func extractJSONMessage(jsonStr string) string {
	re := regexp.MustCompile(`"message"\s*:\s*"([^"]*)"`)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// GetPdfTooLargeErrorMessage returns the PDF too large error message.
func GetPdfTooLargeErrorMessage(isNonInteractive bool) string {
	limits := "max 200 pages, 32MB"
	if isNonInteractive {
		return "PDF too large (" + limits + "). Try reading the file a different way (e.g., extract text with pdftotext)."
	}
	return "PDF too large (" + limits + "). Double press esc to go back and try again, or use pdftotext to convert to text first."
}

// GetPdfPasswordProtectedErrorMessage returns the PDF password protected error message.
func GetPdfPasswordProtectedErrorMessage(isNonInteractive bool) string {
	if isNonInteractive {
		return "PDF is password protected. Try using a CLI tool to extract or convert the PDF."
	}
	return "PDF is password protected. Please double press esc to edit your message and try again."
}

// GetPdfInvalidErrorMessage returns the invalid PDF error message.
func GetPdfInvalidErrorMessage(isNonInteractive bool) string {
	if isNonInteractive {
		return "The PDF file was not valid. Try converting it to text first (e.g., pdftotext)."
	}
	return "The PDF file was not valid. Double press esc to go back and try again with a different file."
}

// GetImageTooLargeErrorMessage returns the image too large error message.
func GetImageTooLargeErrorMessage(isNonInteractive bool) string {
	if isNonInteractive {
		return "Image was too large. Try resizing the image or using a different approach."
	}
	return "Image was too large. Double press esc to go back and try again with a smaller image."
}

// GetRequestTooLargeErrorMessage returns the request too large error message.
func GetRequestTooLargeErrorMessage(isNonInteractive bool) string {
	limits := "max 32MB"
	if isNonInteractive {
		return "Request too large (" + limits + "). Try with a smaller file."
	}
	return "Request too large (" + limits + "). Double press esc to go back and try with a smaller file."
}

// GetTokenRevokedErrorMessage returns the token revoked error message.
func GetTokenRevokedErrorMessage(isNonInteractive bool) string {
	if isNonInteractive {
		return "Your account does not have access to Claude. Please login again or contact your administrator."
	}
	return TokenRevokedErrorMessage
}

// GetOAuthOrgNotAllowedErrorMessage returns the OAuth org not allowed error message.
func GetOAuthOrgNotAllowedErrorMessage(isNonInteractive bool) string {
	if isNonInteractive {
		return "Your organization does not have access to Claude. Please login again or contact your administrator."
	}
	return OAuthOrgNotAllowedErrorMessage
}

// ClassifyAPIError classifies an API error into a specific type for analytics.
func ClassifyAPIError(err error) string {
	if err == nil {
		return "unknown"
	}

	errMsg := err.Error()

	// Aborted requests
	if errMsg == "Request was aborted." {
		return "aborted"
	}

	// Timeout errors
	var timeoutErr *APIConnectionTimeoutError
	var connErr *APIConnectionError
	if errors.As(err, &timeoutErr) ||
		(errors.As(err, &connErr) && strings.Contains(strings.ToLower(connErr.Message), "timeout")) {
		return "api_timeout"
	}

	// Repeated 529 errors
	if strings.Contains(errMsg, Repeated529ErrorMessage) {
		return "repeated_529"
	}

	// Emergency capacity off switch
	if strings.Contains(errMsg, CustomOffSwitchMessage) {
		return "capacity_off_switch"
	}

	// API errors
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case 429:
			return "rate_limit"
		case 529:
			return "server_overload"
		case 400:
			return classifyBadRequest(apiErr.Message)
		case 401, 403:
			return classifyAuthError(apiErr.Message)
		case 404:
			return "model_not_found"
		}

		if apiErr.Status >= 500 {
			return "server_error"
		}
		if apiErr.Status >= 400 {
			return "client_error"
		}
	}

	// Connection errors
	if errors.As(err, &connErr) {
		return "connection_error"
	}

	return "unknown"
}

// classifyBadRequest classifies 400 errors.
func classifyBadRequest(msg string) string {
	msgLower := strings.ToLower(msg)

	if strings.Contains(msgLower, "prompt is too long") {
		return "prompt_too_long"
	}
	if regexp.MustCompile(`maximum of \d+ PDF pages`).MatchString(msg) {
		return "pdf_too_large"
	}
	if strings.Contains(msg, "The PDF specified is password protected") {
		return "pdf_password_protected"
	}
	if strings.Contains(msg, "image exceeds") && strings.Contains(msg, "maximum") {
		return "image_too_large"
	}
	if strings.Contains(msg, "image dimensions exceed") && strings.Contains(msg, "many-image") {
		return "image_too_large"
	}
	if strings.Contains(msg, "`tool_use` ids were found without `tool_result` blocks") {
		return "tool_use_mismatch"
	}
	if strings.Contains(msg, "unexpected `tool_use_id` found in `tool_result`") {
		return "unexpected_tool_result"
	}
	if strings.Contains(msg, "`tool_use` ids must be unique") {
		return "duplicate_tool_use_id"
	}
	if strings.Contains(msgLower, "invalid model name") {
		return "invalid_model"
	}

	return "invalid_request"
}

// classifyAuthError classifies 401/403 errors.
func classifyAuthError(msg string) string {
	if strings.Contains(msg, "OAuth token has been revoked") {
		return "token_revoked"
	}
	if strings.Contains(msg, "OAuth authentication is currently not allowed for this organization") {
		return "oauth_org_not_allowed"
	}
	if strings.Contains(strings.ToLower(msg), "x-api-key") {
		return "invalid_api_key"
	}
	return "auth_error"
}
