package types

import (
	"encoding/json"
)

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// GetTokenCountFromUsage calculates total context window tokens from usage data.
// Includes input_tokens + cache tokens + output_tokens.
func GetTokenCountFromUsage(usage *TokenUsage) int {
	if usage == nil {
		return 0
	}

	return usage.InputTokens +
		usage.CacheCreationInputTokens +
		usage.CacheReadInputTokens +
		usage.OutputTokens
}

// TokenCountFromLastAPIResponse gets the token count from the last API response
func TokenCountFromLastAPIResponse(messages []Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		usage := GetTokenUsage(&messages[i])
		if usage != nil {
			return GetTokenCountFromUsage(usage)
		}
	}
	return 0
}

// GetTokenUsage extracts token usage from a message
func GetTokenUsage(message *Message) *TokenUsage {
	// Parse content to check if it's an assistant message with usage
	var content map[string]interface{}
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return nil
	}

	// Check for usage field
	if usage, ok := content["usage"]; ok {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			result := &TokenUsage{}

			if v, ok := usageMap["input_tokens"].(float64); ok {
				result.InputTokens = int(v)
			}
			if v, ok := usageMap["output_tokens"].(float64); ok {
				result.OutputTokens = int(v)
			}
			if v, ok := usageMap["cache_creation_input_tokens"].(float64); ok {
				result.CacheCreationInputTokens = int(v)
			}
			if v, ok := usageMap["cache_read_input_tokens"].(float64); ok {
				result.CacheReadInputTokens = int(v)
			}

			return result
		}
	}

	return nil
}

// CurrentUsage represents current token usage
type CurrentUsage struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// GetCurrentUsage gets the current usage from the last API response
func GetCurrentUsage(messages []Message) *CurrentUsage {
	for i := len(messages) - 1; i >= 0; i-- {
		usage := GetTokenUsage(&messages[i])
		if usage != nil {
			return &CurrentUsage{
				InputTokens:              usage.InputTokens,
				OutputTokens:             usage.OutputTokens,
				CacheCreationInputTokens: usage.CacheCreationInputTokens,
				CacheReadInputTokens:     usage.CacheReadInputTokens,
			}
		}
	}
	return nil
}

// DoesMostRecentAssistantMessageExceed200k checks if the most recent assistant message exceeds 200k tokens
func DoesMostRecentAssistantMessageExceed200k(messages []Message) bool {
	threshold := 200000

	// Find last assistant message (role == "assistant")
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" {
			usage := GetTokenUsage(&messages[i])
			if usage != nil {
				return GetTokenCountFromUsage(usage) > threshold
			}
			return false
		}
	}
	return false
}

// TokenCountWithEstimation gets the current context window size in tokens.
// This is the CANONICAL function for measuring context size when checking thresholds.
func TokenCountWithEstimation(messages []Message) int {
	// Find the last message with usage
	for i := len(messages) - 1; i >= 0; i-- {
		usage := GetTokenUsage(&messages[i])
		if usage != nil {
			// Get base token count from usage
			baseCount := GetTokenCountFromUsage(usage)

			// Add rough estimation for messages after the usage-bearing message
			estimate := RoughTokenCountEstimationForMessages(messages[i+1:])

			return baseCount + estimate
		}
	}

	// No usage found, estimate all messages
	return RoughTokenCountEstimationForMessages(messages)
}

// RoughTokenCountEstimationForMessages provides a rough token count estimation for messages
// This is a simplified version - the actual implementation would be more accurate
func RoughTokenCountEstimationForMessages(messages []Message) int {
	// Rough estimation: ~4 characters per token
	charCount := 0
	for _, msg := range messages {
		// Count message content length
		charCount += len(msg.Content)
	}
	return charCount / 4
}

// GetAssistantMessageContentLength calculates the character content length of an assistant message.
// Used for spinner token estimation (characters / 4 ≈ tokens).
func GetAssistantMessageContentLength(message *Message) int {
	contentLength := 0

	var content map[string]interface{}
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return 0
	}

	if contentArray, ok := content["content"].([]interface{}); ok {
		for _, block := range contentArray {
			if blockMap, ok := block.(map[string]interface{}); ok {
				switch blockMap["type"] {
				case "text":
					if text, ok := blockMap["text"].(string); ok {
						contentLength += len(text)
					}
				case "thinking":
					if thinking, ok := blockMap["thinking"].(string); ok {
						contentLength += len(thinking)
					}
				case "redacted_thinking":
					if data, ok := blockMap["data"].(string); ok {
						contentLength += len(data)
					}
				case "tool_use":
					if input, err := json.Marshal(blockMap["input"]); err == nil {
						contentLength += len(input)
					}
				}
			}
		}
	}

	return contentLength
}
