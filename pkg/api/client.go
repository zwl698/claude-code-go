package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Anthropic API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Config contains client configuration.
type Config struct {
	APIKey     string
	BaseURL    string
	MaxRetries int
	Timeout    time.Duration
	MaxTokens  int
}

// NewClient creates a new API client.
func NewClient(config Config) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &Client{
		apiKey:  config.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// MessageRequest represents a message request.
type MessageRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Messages    []Message              `json:"messages"`
	System      interface{}            `json:"system,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Thinking    *ThinkingConfig        `json:"thinking,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
}

// Message represents a single message.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a block of content.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	// For tool use
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   interface{}     `json:"content,omitempty"`

	// For thinking
	Thinking string `json:"thinking,omitempty"`
}

// ToolDefinition defines a tool for the API.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ThinkingConfig configures thinking mode.
type ThinkingConfig struct {
	Type        string `json:"type"`
	BudgetToken int    `json:"budget_tokens"`
}

// MessageResponse represents the API response.
type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// Usage contains token usage information.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Type    string           `json:"type"`
	Index   int              `json:"index,omitempty"`
	Delta   *EventDelta      `json:"delta,omitempty"`
	Message *MessageResponse `json:"message,omitempty"`
	Usage   *Usage           `json:"usage,omitempty"`
}

// EventDelta contains the delta for streaming events.
type EventDelta struct {
	Type        string          `json:"type,omitempty"`
	Text        string          `json:"text,omitempty"`
	StopReason  string          `json:"stop_reason,omitempty"`
	Name        string          `json:"name,omitempty"`
	Input       json.RawMessage `json:"input,omitempty"`
	PartialJSON string          `json:"partial_json,omitempty"`
	Thinking    string          `json:"thinking,omitempty"`
}

// CreateMessage sends a message request.
func (c *Client) CreateMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error: %s - %s", apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var result MessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// StreamMessage sends a message request with streaming.
func (c *Client) StreamMessage(ctx context.Context, req MessageRequest, onEvent func(event StreamEvent) error) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var event StreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode event: %w", err)
		}

		if event.Type == "message_stop" {
			break
		}

		if err := onEvent(event); err != nil {
			return err
		}
	}

	return nil
}

// CountTokens counts tokens for a message.
func (c *Client) CountTokens(ctx context.Context, req MessageRequest) (int, error) {
	body, err := json.Marshal(map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages/count_tokens", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.InputTokens, nil
}

// SetAPIKey sets the API key.
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// SetBaseURL sets the base URL.
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}
