package constants

// Constants related to tool result size limits

// DefaultMaxResultSizeChars is the default maximum size in characters for tool results
// before they get persisted to disk. When exceeded, the result is saved to a file
// and the model receives a preview with the file path instead of the full content.
//
// Individual tools may declare a lower maxResultSizeChars, but this constant
// acts as a system-wide cap regardless of what tools declare.
const DefaultMaxResultSizeChars = 50000

// MaxToolResultTokens is the maximum size for tool results in tokens.
// Based on analysis of tool result sizes, we set this to a reasonable upper bound
// to prevent excessively large tool results from consuming too much context.
//
// This is approximately 400KB of text (assuming ~4 bytes per token).
const MaxToolResultTokens = 100000

// BytesPerToken is the bytes per token estimate for calculating token count from byte size.
// This is a conservative estimate - actual token count may vary.
const BytesPerToken = 4

// MaxToolResultBytes is the maximum size for tool results in bytes (derived from token limit).
const MaxToolResultBytes = MaxToolResultTokens * BytesPerToken

// MaxToolResultsPerMessageChars is the default maximum aggregate size in characters for
// tool_result blocks within a SINGLE user message (one turn's batch of parallel tool results).
// When a message's blocks together exceed this, the largest blocks in that message are
// persisted to disk and replaced with previews until under budget.
// Messages are evaluated independently — a 150K result in one turn and a 150K result in
// the next are both untouched.
//
// This prevents N parallel tools from each hitting the per-tool max and collectively
// producing e.g. 10 × 40K = 400K in one turn's user message.
const MaxToolResultsPerMessageChars = 200000

// ToolSummaryMaxLength is the maximum character length for tool summary strings in compact views.
// Used by getToolUseSummary() implementations to truncate long inputs for display
// in grouped agent rendering.
const ToolSummaryMaxLength = 50
