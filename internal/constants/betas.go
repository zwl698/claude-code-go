package constants

// Beta Headers for Anthropic API
// These headers enable various beta features in the API

const (
	// Claude Code beta header
	ClaudeCodeBetaHeader = "claude-code-20250219"

	// Interleaved thinking beta header
	InterleavedThinkingBetaHeader = "interleaved-thinking-2025-05-14"

	// 1M context beta header
	Context1MBetaHeader = "context-1m-2025-08-07"

	// Context management beta header
	ContextManagementBetaHeader = "context-management-2025-06-27"

	// Structured outputs beta header
	StructuredOutputsBetaHeader = "structured-outputs-2025-12-15"

	// Web search beta header
	WebSearchBetaHeader = "web-search-2025-03-05"

	// Tool search beta headers differ by provider:
	// - Claude API / Foundry: advanced-tool-use-2025-11-20
	// - Vertex AI / Bedrock: tool-search-tool-2025-10-19
	ToolSearchBetaHeader1P = "advanced-tool-use-2025-11-20"
	ToolSearchBetaHeader3P = "tool-search-tool-2025-10-19"

	// Effort beta header
	EffortBetaHeader = "effort-2025-11-24"

	// Task budgets beta header
	TaskBudgetsBetaHeader = "task-budgets-2026-03-13"

	// Prompt caching scope beta header
	PromptCachingScopeBetaHeader = "prompt-caching-scope-2026-01-05"

	// Fast mode beta header
	FastModeBetaHeader = "fast-mode-2026-02-01"

	// Redact thinking beta header
	RedactThinkingBetaHeader = "redact-thinking-2026-02-12"

	// Token efficient tools beta header
	TokenEfficientToolsBetaHeader = "token-efficient-tools-2026-03-28"

	// Summarize connector text beta header
	SummarizeConnectorTextBetaHeader = "summarize-connector-text-2026-03-13"

	// AFK mode beta header
	AFKModeBetaHeader = "afk-mode-2026-01-31"

	// CLI internal beta header
	CLIInternalBetaHeader = "cli-internal-2026-02-09"

	// Advisor beta header
	AdvisorBetaHeader = "advisor-tool-2026-03-01"
)

// BedrockExtraParamsHeaders contains beta strings that should be in
// Bedrock extraBodyParams and not in Bedrock headers.
var BedrockExtraParamsHeaders = map[string]bool{
	InterleavedThinkingBetaHeader: true,
	Context1MBetaHeader:           true,
	ToolSearchBetaHeader3P:        true,
}

// VertexCountTokensAllowedBetas contains betas allowed on Vertex countTokens API.
var VertexCountTokensAllowedBetas = map[string]bool{
	ClaudeCodeBetaHeader:          true,
	InterleavedThinkingBetaHeader: true,
	ContextManagementBetaHeader:   true,
}
