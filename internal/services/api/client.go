// Package api provides API client functionality for the claude-code CLI.
// This file contains the Anthropic API client implementation.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/utils"
)

// ClientRequestIDHeader is the header for client request IDs.
const ClientRequestIDHeader = "x-client-request-id"

// ClientOptions contains options for creating an API client.
type ClientOptions struct {
	APIKey           string
	AuthToken        string
	BaseURL          string
	MaxRetries       int
	Timeout          time.Duration
	Model            string
	FetchOverride    func(req *http.Request) (*http.Response, error)
	Source           string
	IsNonInteractive bool
}

// Client is the API client for Anthropic.
type Client struct {
	options        ClientOptions
	httpClient     *http.Client
	defaultHeaders map[string]string
	mu             sync.RWMutex

	// AWS Bedrock authentication
	awsCredentials *AWSCredentials
	awsRegion      string

	// Google Vertex authentication
	googleAccessToken string
	googleTokenExpiry time.Time

	// Azure Foundry authentication
	azureAccessToken string
	azureTokenExpiry time.Time
}

// AnthropicRequest represents a request to the Anthropic API.
type AnthropicRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    json.RawMessage `json:"messages"`
	System      json.RawMessage `json:"system,omitempty"`
	Tools       json.RawMessage `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// AnthropicResponse represents a response from the Anthropic API.
type AnthropicResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      json.RawMessage `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Usage        Usage           `json:"usage"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Default values
const (
	DefaultTimeout = 600 * time.Second
	DefaultBaseURL = "https://api.anthropic.com"
)

// NewClient creates a new API client.
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = DefaultMaxRetries
	}
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultBaseURL
	}

	// Get timeout from environment if set
	if timeoutStr := os.Getenv("API_TIMEOUT_MS"); timeoutStr != "" {
		if timeoutMs, err := strconv.Atoi(timeoutStr); err == nil {
			opts.Timeout = time.Duration(timeoutMs) * time.Millisecond
		}
	}

	client := &Client{
		options: opts,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		defaultHeaders: make(map[string]string),
	}

	// Set up default headers
	client.setupDefaultHeaders()

	return client, nil
}

// setupDefaultHeaders sets up the default headers for requests.
func (c *Client) setupDefaultHeaders() {
	c.defaultHeaders["x-app"] = "cli"
	c.defaultHeaders["User-Agent"] = utils.GetUserAgent()
	c.defaultHeaders["X-Claude-Code-Session-Id"] = os.Getenv("CLAUDE_CODE_SESSION_ID")
	c.defaultHeaders["Content-Type"] = "application/json"

	// Add custom headers
	customHeaders := utils.GetCustomHeaders()
	for k, v := range customHeaders {
		c.defaultHeaders[k] = v
	}

	// Add container ID if present
	if containerID := os.Getenv("CLAUDE_CODE_CONTAINER_ID"); containerID != "" {
		c.defaultHeaders["x-claude-remote-container-id"] = containerID
	}

	// Add remote session ID if present
	if remoteSessionID := os.Getenv("CLAUDE_CODE_REMOTE_SESSION_ID"); remoteSessionID != "" {
		c.defaultHeaders["x-claude-remote-session-id"] = remoteSessionID
	}

	// Add client app if present
	if clientApp := os.Getenv("CLAUDE_AGENT_SDK_CLIENT_APP"); clientApp != "" {
		c.defaultHeaders["x-client-app"] = clientApp
	}

	// Add additional protection header if enabled
	if utils.IsEnvTruthy(os.Getenv("CLAUDE_CODE_ADDITIONAL_PROTECTION")) {
		c.defaultHeaders["x-anthropic-additional-protection"] = "true"
	}
}

// GetAnthropicClient creates and returns an Anthropic API client.
// This is the main entry point for creating API clients.
func GetAnthropicClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	// Check which provider to use
	provider := utils.GetAPIProvider()

	switch provider {
	case utils.APIProviderBedrock:
		return NewBedrockClient(opts)
	case utils.APIProviderVertex:
		return NewVertexClient(ctx, opts)
	case utils.APIProviderFoundry:
		return NewFoundryClient(opts)
	default:
		return NewClient(opts)
	}
}

// DoRequest performs an API request.
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	reqURL := c.options.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.mu.RLock()
	for k, v := range c.defaultHeaders {
		req.Header.Set(k, v)
	}
	c.mu.RUnlock()

	// Set auth based on provider
	provider := utils.GetAPIProvider()
	switch provider {
	case utils.APIProviderBedrock:
		// AWS Bedrock authentication
		if c.awsCredentials != nil && c.awsCredentials.AccessKeyID != "" {
			// Sign the request with AWS SigV4
			signer := NewAWSSigner(c.awsCredentials, c.awsRegion)
			if err := signer.SignRequest(req); err != nil {
				return fmt.Errorf("failed to sign request: %w", err)
			}
		}
		// If using bearer token, it's already set in defaultHeaders
	case utils.APIProviderVertex:
		// Google Vertex authentication - token already in defaultHeaders
		if c.googleAccessToken != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+c.googleAccessToken)
		}
	case utils.APIProviderFoundry:
		// Azure Foundry authentication - token or api-key already in defaultHeaders
		if c.azureAccessToken != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+c.azureAccessToken)
		}
	default:
		// First-party API authentication
		if c.options.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.options.AuthToken)
		} else if c.options.APIKey != "" {
			req.Header.Set("x-api-key", c.options.APIKey)
		}
	}

	// Add client request ID for first-party API
	if provider == utils.APIProviderFirstParty && utils.IsFirstPartyAnthropicBaseURL() {
		if req.Header.Get(ClientRequestIDHeader) == "" {
			req.Header.Set(ClientRequestIDHeader, utils.GenerateUUID())
		}
	}

	// Execute request with retry
	return c.doRequestWithRetry(req, result)
}

// doRequestWithRetry executes a request with retry logic.
func (c *Client) doRequestWithRetry(req *http.Request, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.options.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}

		// Clone request body for retry
		var bodyBytes []byte
		if req.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(req.Body)
			if err != nil {
				return err
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Execute request
		var httpResp *http.Response
		var err error

		if c.options.FetchOverride != nil {
			httpResp, err = c.options.FetchOverride(req)
		} else {
			httpResp, err = c.httpClient.Do(req)
		}

		if err != nil {
			lastErr = err
			if isRetryableError(err) {
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}

		// Read response body
		respBody, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Check for errors
		if httpResp.StatusCode >= 400 {
			apiErr := &APIError{
				Status:  httpResp.StatusCode,
				Message: string(respBody),
				Headers: make(map[string]string),
			}
			for k, v := range httpResp.Header {
				if len(v) > 0 {
					apiErr.Headers[k] = v[0]
				}
			}

			if isRetryableStatus(httpResp.StatusCode) && attempt < c.options.MaxRetries {
				lastErr = apiErr
				continue
			}

			return apiErr
		}

		// Parse response
		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// CreateMessage creates a message using the Messages API.
func (c *Client) CreateMessage(ctx context.Context, req *AnthropicRequest) (*AnthropicResponse, error) {
	var resp AnthropicResponse
	err := c.DoRequest(ctx, http.MethodPost, "/v1/messages", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateMessageStream creates a streaming message using the Messages API.
func (c *Client) CreateMessageStream(ctx context.Context, req *AnthropicRequest) (*StreamIterator, error) {
	req.Stream = true

	var bodyBytes []byte
	var err error
	if req != nil {
		bodyBytes, err = json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := c.options.BaseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.mu.RLock()
	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}
	c.mu.RUnlock()

	// Set auth
	if c.options.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.options.AuthToken)
	} else if c.options.APIKey != "" {
		httpReq.Header.Set("x-api-key", c.options.APIKey)
	}

	// Add streaming headers
	httpReq.Header.Set("Accept", "text/event-stream")

	// Add client request ID
	if utils.GetAPIProvider() == utils.APIProviderFirstParty && utils.IsFirstPartyAnthropicBaseURL() {
		if httpReq.Header.Get(ClientRequestIDHeader) == "" {
			httpReq.Header.Set(ClientRequestIDHeader, utils.GenerateUUID())
		}
	}

	// Execute request
	var httpResp *http.Response
	if c.options.FetchOverride != nil {
		httpResp, err = c.options.FetchOverride(httpReq)
	} else {
		httpResp, err = c.httpClient.Do(httpReq)
	}

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		defer httpResp.Body.Close()
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			Status:  httpResp.StatusCode,
			Message: string(body),
		}
	}

	return NewStreamIterator(httpResp.Body), nil
}

// StreamIterator handles streaming responses.
type StreamIterator struct {
	reader  io.ReadCloser
	decoder *json.Decoder
	buffer  []byte
}

// NewStreamIterator creates a new stream iterator.
func NewStreamIterator(reader io.ReadCloser) *StreamIterator {
	return &StreamIterator{
		reader: reader,
		buffer: make([]byte, 4096),
	}
}

// Next returns the next event from the stream.
func (s *StreamIterator) Next() (map[string]interface{}, error) {
	// Read SSE format
	for {
		line, err := s.readLine()
		if err != nil {
			return nil, err
		}

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse SSE data line
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			var event map[string]interface{}
			if err := json.Unmarshal(data, &event); err != nil {
				continue
			}
			return event, nil
		}
	}
}

// readLine reads a single line from the stream.
func (s *StreamIterator) readLine() ([]byte, error) {
	var line []byte
	for {
		var buf [1]byte
		_, err := s.reader.Read(buf[:])
		if err != nil {
			return nil, err
		}

		if buf[0] == '\n' {
			return bytes.TrimSpace(line), nil
		}
		line = append(line, buf[0])
	}
}

// Close closes the stream iterator.
func (s *StreamIterator) Close() error {
	return s.reader.Close()
}

// isRetryableError checks if an error is retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF")
}

// isRetryableStatus checks if an HTTP status code is retryable.
func isRetryableStatus(status int) bool {
	return status == 429 || status == 500 || status == 502 || status == 503 || status == 529
}

// ConfigureAPIKeyHeaders configures API key headers for authentication.
func ConfigureAPIKeyHeaders(headers map[string]string, authManager *utils.AuthManager, isNonInteractive bool) error {
	// Check for auth token
	token := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	if token == "" {
		// Try to get from API key helper
		ctx := context.Background()
		helperToken, err := authManager.GetAPIKeyFromAPIKeyHelper(ctx)
		if err == nil && helperToken != "" {
			token = helperToken
		}
	}

	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}

	return nil
}
