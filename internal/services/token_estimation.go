package services

import (
	"encoding/json"
	"math"
	"strings"
)

// =============================================================================
// Token Estimation Constants
// =============================================================================

const (
	// DefaultBytesPerToken is the default ratio for token estimation
	// Roughly 4 characters per token for English text
	DefaultBytesPerToken = 4

	// JSONBytesPerToken is the ratio for dense JSON content
	// Dense JSON has many single-character tokens (`{`, `}`, `:`, `,`, `"`)
	// which makes the real ratio closer to 2 rather than the default 4.
	JSONBytesPerToken = 2

	// ImageMaxTokenSize is the maximum token count for images
	// https://platform.claude.com/docs/en/build-with-claude/vision#calculate-image-costs
	// tokens = (width px * height px)/750
	// Images are resized to max 2000x2000 (5333 tokens). Use a conservative
	// estimate that matches microCompact's IMAGE_MAX_TOKEN_SIZE to avoid
	// underestimating and triggering auto-compact too late.
	ImageMaxTokenSize = 2000

	// DocumentMaxTokenSize is the maximum token count for documents
	// Document: base64 PDF in source.data. Must NOT reach the
	// jsonStringify catch-all — a 1MB PDF is ~1.33M base64 chars →
	// ~325k estimated tokens, vs the ~2000 the API actually charges.
	DocumentMaxTokenSize = 2000
)

// =============================================================================
// Extended Token Estimator
// =============================================================================

// ExtendedTokenEstimator provides extended token estimation functionality
// with support for different content types and file types
type ExtendedTokenEstimator struct {
	bytesPerToken float64
}

// NewExtendedTokenEstimator creates a new extended token estimator with default settings
func NewExtendedTokenEstimator() *ExtendedTokenEstimator {
	return &ExtendedTokenEstimator{
		bytesPerToken: DefaultBytesPerToken,
	}
}

// NewExtendedTokenEstimatorForFileType creates a token estimator for a specific file type
func NewExtendedTokenEstimatorForFileType(fileExtension string) *ExtendedTokenEstimator {
	return &ExtendedTokenEstimator{
		bytesPerToken: float64(BytesPerTokenForFileType(fileExtension)),
	}
}

// BytesPerTokenForFileType returns the bytes-per-token ratio for a file type
func BytesPerTokenForFileType(fileExtension string) int {
	switch strings.ToLower(fileExtension) {
	case "json", "jsonl", "jsonc":
		return JSONBytesPerToken
	default:
		return DefaultBytesPerToken
	}
}

// RoughTokenCountEstimation estimates token count for a string
func (e *ExtendedTokenEstimator) RoughTokenCountEstimation(content string) int {
	return int(math.Round(float64(len(content)) / e.bytesPerToken))
}

// RoughTokenCountEstimationForFileType estimates tokens with file-type awareness
func (e *ExtendedTokenEstimator) RoughTokenCountEstimationForFileType(content string, fileExtension string) int {
	return int(math.Round(float64(len(content)) / float64(BytesPerTokenForFileType(fileExtension))))
}

// RoughTokenCountEstimationForMessages estimates tokens for a slice of messages
// This provides a more detailed estimation than the simple DefaultTokenEstimator
func (e *ExtendedTokenEstimator) RoughTokenCountEstimationForMessages(messages []*CompactMessage) int {
	total := 0
	for _, msg := range messages {
		total += e.RoughTokenCountEstimationForMessage(msg)
	}
	return total
}

// RoughTokenCountEstimationForMessage estimates tokens for a single message
func (e *ExtendedTokenEstimator) RoughTokenCountEstimationForMessage(message *CompactMessage) int {
	if message == nil {
		return 0
	}

	// Handle different message types
	switch message.Type {
	case "assistant":
		return e.estimateTokensForAssistantMessage(message)
	case "user":
		return e.estimateTokensForUserMessage(message)
	case "attachment":
		return e.estimateTokensForAttachmentMessage(message)
	default:
		return e.RoughTokenCountEstimation(string(message.Content))
	}
}

// estimateTokensForAssistantMessage estimates tokens for assistant messages
func (e *ExtendedTokenEstimator) estimateTokensForAssistantMessage(message *CompactMessage) int {
	if message.Role != "assistant" {
		return 0
	}

	// Parse content as array of blocks
	var blocks []map[string]interface{}
	if err := json.Unmarshal(message.Content, &blocks); err != nil {
		// Fallback to raw string estimation
		return e.RoughTokenCountEstimation(string(message.Content))
	}

	total := 0
	for _, block := range blocks {
		total += e.estimateTokensForBlock(block)
	}
	return total
}

// estimateTokensForUserMessage estimates tokens for user messages
func (e *ExtendedTokenEstimator) estimateTokensForUserMessage(message *CompactMessage) int {
	if message.Role != "user" {
		return 0
	}

	// Try to parse as string first
	var contentStr string
	if err := json.Unmarshal(message.Content, &contentStr); err == nil {
		return e.RoughTokenCountEstimation(contentStr)
	}

	// Try to parse as array of blocks
	var blocks []map[string]interface{}
	if err := json.Unmarshal(message.Content, &blocks); err == nil {
		total := 0
		for _, block := range blocks {
			total += e.estimateTokensForBlock(block)
		}
		return total
	}

	// Fallback to raw string estimation
	return e.RoughTokenCountEstimation(string(message.Content))
}

// estimateTokensForAttachmentMessage estimates tokens for attachment messages
func (e *ExtendedTokenEstimator) estimateTokensForAttachmentMessage(message *CompactMessage) int {
	if message.Type != "attachment" {
		return 0
	}

	// Parse attachment content
	var attachment map[string]interface{}
	if err := json.Unmarshal(message.Content, &attachment); err != nil {
		return e.RoughTokenCountEstimation(string(message.Content))
	}

	// Estimate based on attachment type
	if attType, ok := attachment["type"].(string); ok {
		switch attType {
		case "file":
			if content, ok := attachment["content"].(string); ok {
				return e.RoughTokenCountEstimation(content)
			}
		case "skill_discovery", "skill_listing":
			// These are typically small metadata
			return e.RoughTokenCountEstimation(string(message.Content))
		}
	}

	return e.RoughTokenCountEstimation(string(message.Content))
}

// estimateTokensForBlock estimates tokens for a content block
func (e *ExtendedTokenEstimator) estimateTokensForBlock(block map[string]interface{}) int {
	blockType, hasType := block["type"].(string)
	if !hasType {
		// Unknown block type, estimate from JSON representation
		data, _ := json.Marshal(block)
		return e.RoughTokenCountEstimation(string(data))
	}

	switch blockType {
	case "text":
		if text, ok := block["text"].(string); ok {
			return e.RoughTokenCountEstimation(text)
		}

	case "image":
		// Images are resized to max 2000x2000 (5333 tokens)
		return ImageMaxTokenSize

	case "document":
		// Documents use a conservative estimate
		return DocumentMaxTokenSize

	case "tool_result":
		return e.estimateTokensForToolResult(block)

	case "tool_use":
		return e.estimateTokensForToolUse(block)

	case "thinking":
		if thinking, ok := block["thinking"].(string); ok {
			return e.RoughTokenCountEstimation(thinking)
		}

	case "redacted_thinking":
		if data, ok := block["data"].(string); ok {
			return e.RoughTokenCountEstimation(data)
		}

	default:
		// server_tool_use, web_search_tool_result, mcp_tool_use, etc.
		// Text-like payloads (tool inputs, search results, no base64).
		// Stringify-length tracks the serialized form the API sees.
		data, _ := json.Marshal(block)
		return e.RoughTokenCountEstimation(string(data))
	}

	return 0
}

// estimateTokensForToolResult estimates tokens for tool_result blocks
func (e *ExtendedTokenEstimator) estimateTokensForToolResult(block map[string]interface{}) int {
	total := 0

	// Add tool_use_id tokens
	if toolUseID, ok := block["tool_use_id"].(string); ok {
		total += e.RoughTokenCountEstimation(toolUseID)
	}

	// Estimate content tokens
	if content, ok := block["content"]; ok {
		switch c := content.(type) {
		case string:
			total += e.RoughTokenCountEstimation(c)
		case []interface{}:
			for _, item := range c {
				if itemMap, ok := item.(map[string]interface{}); ok {
					total += e.estimateTokensForBlock(itemMap)
				}
			}
		}
	}

	return total
}

// estimateTokensForToolUse estimates tokens for tool_use blocks
func (e *ExtendedTokenEstimator) estimateTokensForToolUse(block map[string]interface{}) int {
	total := 0

	// Add name tokens
	if name, ok := block["name"].(string); ok {
		total += e.RoughTokenCountEstimation(name)
	}

	// Add id tokens
	if id, ok := block["id"].(string); ok {
		total += e.RoughTokenCountEstimation(id)
	}

	// Input is the JSON the model generated — arbitrarily large (bash
	// commands, Edit diffs, file contents). Stringify for the char count.
	if input, ok := block["input"]; ok {
		data, _ := json.Marshal(input)
		total += e.RoughTokenCountEstimation(string(data))
	}

	return total
}

// =============================================================================
// Standalone Functions
// =============================================================================

// RoughTokenCountEstimation is a standalone function for simple token estimation
func RoughTokenCountEstimation(content string, bytesPerToken ...int) int {
	bpt := DefaultBytesPerToken
	if len(bytesPerToken) > 0 && bytesPerToken[0] > 0 {
		bpt = bytesPerToken[0]
	}
	return int(math.Round(float64(len(content)) / float64(bpt)))
}

// RoughTokenCountEstimationForFileType estimates tokens with file type awareness
func RoughTokenCountEstimationForFileType(content string, fileExtension string) int {
	return int(math.Round(float64(len(content)) / float64(BytesPerTokenForFileType(fileExtension))))
}

// TruncateToTokens truncates content to roughly maxTokens, keeping the head.
// RoughTokenCountEstimation uses ~4 chars/token, so char budget = maxTokens * 4
// minus the marker so the result stays within budget.
func TruncateToTokens(content string, maxTokens int, truncationMarker ...string) string {
	estimator := NewExtendedTokenEstimator()
	if estimator.RoughTokenCountEstimation(content) <= maxTokens {
		return content
	}

	marker := "\n\n[... content truncated for compaction; use Read to get the full content if needed]"
	if len(truncationMarker) > 0 {
		marker = truncationMarker[0]
	}

	charBudget := maxTokens*DefaultBytesPerToken - len(marker)
	if charBudget < 0 {
		charBudget = 0
	}

	if len(content) <= charBudget {
		return content
	}

	return content[:charBudget] + marker
}

// =============================================================================
// Token Count for API
// =============================================================================

// APITokenCounter handles token counting via API (when available)
type APITokenCounter struct {
	// In a real implementation, this would have an API client
	// and support for different providers (Anthropic, Bedrock, Vertex)
}

// NewAPITokenCounter creates a new API token counter
func NewAPITokenCounter() *APITokenCounter {
	return &APITokenCounter{}
}

// CountTokensWithAPI counts tokens via API for a string content
// Returns nil if API is not available or fails
func (c *APITokenCounter) CountTokensWithAPI(content string) *int {
	// Special case for empty content - API doesn't accept empty messages
	if content == "" {
		zero := 0
		return &zero
	}

	// In a real implementation, this would call the Anthropic API
	// For now, return nil to indicate API counting is not available
	return nil
}

// CountMessagesTokensWithAPI counts tokens via API for messages
// Returns nil if API is not available or fails
func (c *APITokenCounter) CountMessagesTokensWithAPI(messages []*CompactMessage, tools []interface{}) *int {
	// In a real implementation, this would:
	// 1. Normalize messages for API
	// 2. Strip tool search-specific fields
	// 3. Call the appropriate API (Anthropic, Bedrock, Vertex)
	// 4. Return the token count from the response

	// For now, return nil to indicate API counting is not available
	return nil
}

// =============================================================================
// Thinking Block Detection
// =============================================================================

// HasThinkingBlocks checks if messages contain thinking blocks
func HasThinkingBlocks(messages []*CompactMessage) bool {
	for _, message := range messages {
		if message.Role == "assistant" {
			var blocks []map[string]interface{}
			if err := json.Unmarshal(message.Content, &blocks); err != nil {
				continue
			}

			for _, block := range blocks {
				if blockType, ok := block["type"].(string); ok {
					if blockType == "thinking" || blockType == "redacted_thinking" {
						return true
					}
				}
			}
		}
	}
	return false
}

// =============================================================================
// Token Budget Calculator
// =============================================================================

// TokenBudgetCalculator calculates token budgets for various operations
type TokenBudgetCalculator struct {
	maxContextTokens   int
	systemPromptTokens int
	reservedTokens     int
}

// NewTokenBudgetCalculator creates a new token budget calculator
func NewTokenBudgetCalculator(maxContextTokens, systemPromptTokens int) *TokenBudgetCalculator {
	// Reserve some tokens for the response
	reservedTokens := 4096

	return &TokenBudgetCalculator{
		maxContextTokens:   maxContextTokens,
		systemPromptTokens: systemPromptTokens,
		reservedTokens:     reservedTokens,
	}
}

// AvailableTokensForMessages calculates available tokens for messages
func (c *TokenBudgetCalculator) AvailableTokensForMessages() int {
	return c.maxContextTokens - c.systemPromptTokens - c.reservedTokens
}

// CanAddMessages checks if messages can be added within the budget
func (c *TokenBudgetCalculator) CanAddMessages(currentTokens int, additionalTokens int) bool {
	return currentTokens+additionalTokens <= c.AvailableTokensForMessages()
}

// ShouldAutoCompact checks if auto-compact should be triggered
func (c *TokenBudgetCalculator) ShouldAutoCompact(currentTokens int, thresholdPercent float64) bool {
	threshold := int(float64(c.AvailableTokensForMessages()) * thresholdPercent)
	return currentTokens >= threshold
}

// =============================================================================
// Token Usage Statistics
// =============================================================================

// TokenUsageStats represents token usage statistics
type TokenUsageStats struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// TotalTokens returns the total tokens used
func (s *TokenUsageStats) TotalTokens() int {
	return s.InputTokens + s.OutputTokens + s.CacheReadInputTokens + s.CacheCreationInputTokens
}

// CacheHitRate calculates the cache hit rate
func (s *TokenUsageStats) CacheHitRate() float64 {
	total := s.InputTokens + s.CacheCreationInputTokens + s.CacheReadInputTokens
	if total == 0 {
		return 0
	}
	return float64(s.CacheReadInputTokens) / float64(total)
}
