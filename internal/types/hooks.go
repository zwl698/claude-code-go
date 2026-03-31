// Package types contains core type definitions for the claude-code CLI.
// This file contains hook-related types translated from TypeScript.
package types

// HookEvent represents the different hook event types.
type HookEvent string

const (
	HookEventPreToolUse         HookEvent = "PreToolUse"
	HookEventUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookEventSessionStart       HookEvent = "SessionStart"
	HookEventSetup              HookEvent = "Setup"
	HookEventSubagentStart      HookEvent = "SubagentStart"
	HookEventPostToolUse        HookEvent = "PostToolUse"
	HookEventPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookEventPermissionDenied   HookEvent = "PermissionDenied"
	HookEventNotification       HookEvent = "Notification"
	HookEventPermissionRequest  HookEvent = "PermissionRequest"
	HookEventElicitation        HookEvent = "Elicitation"
	HookEventElicitationResult  HookEvent = "ElicitationResult"
	HookEventCwdChanged         HookEvent = "CwdChanged"
	HookEventFileChanged        HookEvent = "FileChanged"
	HookEventWorktreeCreate     HookEvent = "WorktreeCreate"
)

// HookEvents is the list of all valid hook events.
var HookEvents = []HookEvent{
	HookEventPreToolUse,
	HookEventUserPromptSubmit,
	HookEventSessionStart,
	HookEventSetup,
	HookEventSubagentStart,
	HookEventPostToolUse,
	HookEventPostToolUseFailure,
	HookEventPermissionDenied,
	HookEventNotification,
	HookEventPermissionRequest,
	HookEventElicitation,
	HookEventElicitationResult,
	HookEventCwdChanged,
	HookEventFileChanged,
	HookEventWorktreeCreate,
}

// IsHookEvent checks if a string is a valid hook event.
func IsHookEvent(value string) bool {
	for _, event := range HookEvents {
		if string(event) == value {
			return true
		}
	}
	return false
}

// PromptRequest represents a prompt elicitation request.
type PromptRequest struct {
	Prompt  string         `json:"prompt"`
	Message string         `json:"message"`
	Options []PromptOption `json:"options"`
}

// PromptOption represents an option in a prompt request.
type PromptOption struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// PromptResponse represents a response to a prompt request.
type PromptResponse struct {
	PromptResponse string `json:"prompt_response"`
	Selected       string `json:"selected"`
}

// SyncHookResponse represents a synchronous hook response.
type SyncHookResponse struct {
	Continue           *bool               `json:"continue,omitempty"`
	SuppressOutput     *bool               `json:"suppressOutput,omitempty"`
	StopReason         string              `json:"stopReason,omitempty"`
	Decision           string              `json:"decision,omitempty"` // approve | block
	Reason             string              `json:"reason,omitempty"`
	SystemMessage      string              `json:"systemMessage,omitempty"`
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookSpecificOutput contains event-specific hook output.
type HookSpecificOutput struct {
	HookEventName HookEvent `json:"hookEventName"`

	// PreToolUse fields
	PermissionDecision       *string                `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string                 `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             map[string]interface{} `json:"updatedInput,omitempty"`
	AdditionalContext        string                 `json:"additionalContext,omitempty"`

	// SessionStart fields
	InitialUserMessage string   `json:"initialUserMessage,omitempty"`
	WatchPaths         []string `json:"watchPaths,omitempty"`

	// PostToolUse fields
	UpdatedMCPToolOutput interface{} `json:"updatedMCPToolOutput,omitempty"`

	// PermissionDenied fields
	Retry *bool `json:"retry,omitempty"`

	// PermissionRequest fields
	PermissionRequestResult *PermissionRequestResult `json:"permissionRequestResult,omitempty"`

	// Elicitation/ElicitationResult fields
	Action  string                 `json:"action,omitempty"` // accept | decline | cancel
	Content map[string]interface{} `json:"content,omitempty"`

	// WorktreeCreate fields
	WorktreePath string `json:"worktreePath,omitempty"`
}

// AsyncHookResponse represents an asynchronous hook response.
type AsyncHookResponse struct {
	Async        bool `json:"async"`
	AsyncTimeout *int `json:"asyncTimeout,omitempty"`
}

// HookJSONOutput is the union of async and sync hook responses.
type HookJSONOutput struct {
	// Async fields
	Async        bool `json:"async,omitempty"`
	AsyncTimeout *int `json:"asyncTimeout,omitempty"`

	// Sync fields (embedded)
	SyncHookResponse
}

// IsSync returns true if the hook output is a sync response.
func (h *HookJSONOutput) IsSync() bool {
	return !h.Async
}

// IsAsync returns true if the hook output is an async response.
func (h *HookJSONOutput) IsAsync() bool {
	return h.Async
}

// PermissionRequestResult represents the result of a permission request hook.
type PermissionRequestResult struct {
	Behavior           string                 `json:"behavior"` // allow | deny
	UpdatedInput       map[string]interface{} `json:"updatedInput,omitempty"`
	UpdatedPermissions []PermissionUpdate     `json:"updatedPermissions,omitempty"`
	Message            string                 `json:"message,omitempty"`
	Interrupt          *bool                  `json:"interrupt,omitempty"`
}

// HookCallback is a function that implements a hook.
type HookCallback func(input HookInput, toolUseID string, abort <-chan struct{}) (*HookJSONOutput, error)

// HookConfigWithCallback represents a hook with a callback implementation.
type HookConfigWithCallback struct {
	Type     HookEvent    `json:"type"`
	Callback HookCallback `json:"-"`
	Timeout  int          `json:"timeout,omitempty"`
	Internal bool         `json:"internal,omitempty"`
}

// HookCallbackMatcher groups hooks with an optional matcher.
type HookCallbackMatcher struct {
	Matcher    string         `json:"matcher,omitempty"`
	Hooks      []HookCallback `json:"-"`
	PluginName string         `json:"pluginName,omitempty"`
}

// HookProgress represents progress information for a running hook.
type HookProgress struct {
	Type          HookEvent `json:"type"`
	HookEvent     HookEvent `json:"hookEvent"`
	HookName      string    `json:"hookName"`
	Command       string    `json:"command"`
	PromptText    string    `json:"promptText,omitempty"`
	StatusMessage string    `json:"statusMessage,omitempty"`
}

// HookBlockingError represents an error that blocks execution.
type HookBlockingError struct {
	BlockingError string `json:"blockingError"`
	Command       string `json:"command"`
}

// HookResult represents the result of hook execution.
type HookResult struct {
	Message                      interface{}              `json:"message,omitempty"`
	SystemMessage                interface{}              `json:"systemMessage,omitempty"`
	BlockingError                *HookBlockingError       `json:"blockingError,omitempty"`
	Outcome                      string                   `json:"outcome"` // success | blocking | non_blocking_error | cancelled
	PreventContinuation          bool                     `json:"preventContinuation,omitempty"`
	StopReason                   string                   `json:"stopReason,omitempty"`
	PermissionBehavior           string                   `json:"permissionBehavior,omitempty"` // ask | deny | allow | passthrough
	HookPermissionDecisionReason string                   `json:"hookPermissionDecisionReason,omitempty"`
	AdditionalContext            string                   `json:"additionalContext,omitempty"`
	InitialUserMessage           string                   `json:"initialUserMessage,omitempty"`
	UpdatedInput                 map[string]interface{}   `json:"updatedInput,omitempty"`
	UpdatedMCPToolOutput         interface{}              `json:"updatedMCPToolOutput,omitempty"`
	PermissionRequestResult      *PermissionRequestResult `json:"permissionRequestResult,omitempty"`
	Retry                        *bool                    `json:"retry,omitempty"`
}

// AggregatedHookResult combines results from multiple hooks.
type AggregatedHookResult struct {
	Message                      interface{}              `json:"message,omitempty"`
	BlockingErrors               []HookBlockingError      `json:"blockingErrors,omitempty"`
	PreventContinuation          bool                     `json:"preventContinuation,omitempty"`
	StopReason                   string                   `json:"stopReason,omitempty"`
	HookPermissionDecisionReason string                   `json:"hookPermissionDecisionReason,omitempty"`
	PermissionBehavior           string                   `json:"permissionBehavior,omitempty"`
	AdditionalContexts           []string                 `json:"additionalContexts,omitempty"`
	InitialUserMessage           string                   `json:"initialUserMessage,omitempty"`
	UpdatedInput                 map[string]interface{}   `json:"updatedInput,omitempty"`
	UpdatedMCPToolOutput         interface{}              `json:"updatedMCPToolOutput,omitempty"`
	PermissionRequestResult      *PermissionRequestResult `json:"permissionRequestResult,omitempty"`
	Retry                        *bool                    `json:"retry,omitempty"`
}

// HookInput represents the input passed to a hook.
type HookInput struct {
	EventName    HookEvent              `json:"eventName"`
	ToolName     string                 `json:"toolName,omitempty"`
	ToolInput    map[string]interface{} `json:"toolInput,omitempty"`
	ToolUseId    string                 `json:"toolUseId,omitempty"`
	Prompt       string                 `json:"prompt,omitempty"`
	Reason       string                 `json:"reason,omitempty"`
	Notification string                 `json:"notification,omitempty"`
	NewCwd       string                 `json:"newCwd,omitempty"`
	ChangedFiles []string               `json:"changedFiles,omitempty"`
	WorktreePath string                 `json:"worktreePath,omitempty"`
	SessionId    string                 `json:"sessionId,omitempty"`
	AgentId      string                 `json:"agentId,omitempty"`
}
