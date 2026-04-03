package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// =============================================================================
// Hook Types
// =============================================================================

// HookEvent represents a hook event type.
type HookEvent string

const (
	HookEventSessionStart       HookEvent = "SessionStart"
	HookEventSessionEnd         HookEvent = "SessionEnd"
	HookEventPreToolUse         HookEvent = "PreToolUse"
	HookEventPostToolUse        HookEvent = "PostToolUse"
	HookEventPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookEventUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookEventPermissionRequest  HookEvent = "PermissionRequest"
	HookEventPermissionDenied   HookEvent = "PermissionDenied"
	HookEventNotification       HookEvent = "Notification"
	HookEventCwdChanged         HookEvent = "CwdChanged"
	HookEventFileChanged        HookEvent = "FileChanged"
	HookEventSubagentStart      HookEvent = "SubagentStart"
	HookEventSetup              HookEvent = "Setup"
	HookEventElicitation        HookEvent = "Elicitation"
	HookEventElicitationResult  HookEvent = "ElicitationResult"
)

// HookEvents contains all valid hook events.
var HookEvents = []HookEvent{
	HookEventSessionStart,
	HookEventSessionEnd,
	HookEventPreToolUse,
	HookEventPostToolUse,
	HookEventPostToolUseFailure,
	HookEventUserPromptSubmit,
	HookEventPermissionRequest,
	HookEventPermissionDenied,
	HookEventNotification,
	HookEventCwdChanged,
	HookEventFileChanged,
	HookEventSubagentStart,
	HookEventSetup,
	HookEventElicitation,
	HookEventElicitationResult,
}

// IsHookEvent checks if a string is a valid hook event.
func IsHookEvent(s string) bool {
	for _, e := range HookEvents {
		if string(e) == s {
			return true
		}
	}
	return false
}

// =============================================================================
// Hook Input/Output
// =============================================================================

// HookInput represents the input to a hook.
type HookInput struct {
	EventName         HookEvent          `json:"eventName"`
	SessionID         string             `json:"sessionId"`
	ToolName          string             `json:"toolName,omitempty"`
	ToolInput         interface{}        `json:"toolInput,omitempty"`
	ToolResult        interface{}        `json:"toolResult,omitempty"`
	Prompt            string             `json:"prompt,omitempty"`
	PermissionRequest *PermissionRequest `json:"permissionRequest,omitempty"`
	Message           string             `json:"message,omitempty"`
	Cwd               string             `json:"cwd,omitempty"`
	File              string             `json:"file,omitempty"`
	AgentID           string             `json:"agentId,omitempty"`
}

// HookOutput represents the output from a hook.
type HookOutput struct {
	Continue                 bool                   `json:"continue,omitempty"`
	SuppressOutput           bool                   `json:"suppressOutput,omitempty"`
	StopReason               string                 `json:"stopReason,omitempty"`
	Decision                 string                 `json:"decision,omitempty"` // approve, block
	Reason                   string                 `json:"reason,omitempty"`
	SystemMessage            string                 `json:"systemMessage,omitempty"`
	AdditionalContext        string                 `json:"additionalContext,omitempty"`
	UpdatedInput             map[string]interface{} `json:"updatedInput,omitempty"`
	PermissionDecision       string                 `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string                 `json:"permissionDecisionReason,omitempty"`
	UpdatedPermissions       []PermissionUpdate     `json:"updatedPermissions,omitempty"`
	WatchPaths               []string               `json:"watchPaths,omitempty"`
	InitialUserMessage       string                 `json:"initialUserMessage,omitempty"`
}

// PermissionRequest represents a permission request in a hook.
type PermissionRequest struct {
	ToolName string                 `json:"toolName"`
	Input    map[string]interface{} `json:"input"`
}

// PermissionUpdate represents a permission update from a hook.
type PermissionUpdate struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
	Behavior    string `json:"behavior"` // allow, deny
}

// =============================================================================
// Hook Handler
// =============================================================================

// HookHandler is a function that handles a hook event.
type HookHandler func(ctx context.Context, input HookInput) (HookOutput, error)

// =============================================================================
// Hook Registry
// =============================================================================

// Registry manages hook registrations.
type Registry struct {
	mu      sync.RWMutex
	hooks   map[HookEvent][]HookHandler
	enabled bool
}

// NewRegistry creates a new hook registry.
func NewRegistry() *Registry {
	return &Registry{
		hooks:   make(map[HookEvent][]HookHandler),
		enabled: true,
	}
}

// Register registers a hook handler for an event.
func (r *Registry) Register(event HookEvent, handler HookHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks[event] = append(r.hooks[event], handler)
}

// Unregister removes all handlers for an event.
func (r *Registry) Unregister(event HookEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.hooks, event)
}

// Enable enables hook execution.
func (r *Registry) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.enabled = true
}

// Disable disables hook execution.
func (r *Registry) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.enabled = false
}

// Execute executes all handlers for an event.
func (r *Registry) Execute(ctx context.Context, input HookInput) ([]HookOutput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.enabled {
		return nil, nil
	}

	handlers := r.hooks[input.EventName]
	if len(handlers) == 0 {
		return nil, nil
	}

	outputs := make([]HookOutput, 0, len(handlers))
	for _, handler := range handlers {
		output, err := handler(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("hook handler failed: %w", err)
		}
		outputs = append(outputs, output)

		// If a handler blocks, stop execution
		if output.Decision == "block" {
			break
		}
	}

	return outputs, nil
}

// =============================================================================
// Built-in Hooks
// =============================================================================

// LoggingHook logs all hook events.
func LoggingHook(ctx context.Context, input HookInput) (HookOutput, error) {
	data, _ := json.Marshal(input)
	fmt.Printf("[HOOK] %s: %s\n", input.EventName, string(data))
	return HookOutput{Continue: true}, nil
}

// PermissionCheckHook checks permissions before tool use.
func PermissionCheckHook(ctx context.Context, input HookInput) (HookOutput, error) {
	if input.EventName != HookEventPreToolUse {
		return HookOutput{Continue: true}, nil
	}

	// Check for dangerous operations
	if input.ToolName == "Bash" {
		toolInput, ok := input.ToolInput.(map[string]interface{})
		if !ok {
			return HookOutput{Continue: true}, nil
		}

		if cmd, ok := toolInput["command"].(string); ok {
			// Check for dangerous patterns
			dangerous := []string{"rm -rf", "sudo", "> /dev/", "mkfs"}
			for _, pattern := range dangerous {
				if contains(cmd, pattern) {
					return HookOutput{
						Continue:      false,
						Decision:      "block",
						Reason:        fmt.Sprintf("Potentially dangerous command detected: %s", pattern),
						SystemMessage: "This command was blocked for safety. If you need to run it, please use explicit approval.",
					}, nil
				}
			}
		}
	}

	return HookOutput{Continue: true}, nil
}

// SessionTrackingHook tracks session events.
func SessionTrackingHook(ctx context.Context, input HookInput) (HookOutput, error) {
	switch input.EventName {
	case HookEventSessionStart:
		fmt.Printf("Session started: %s\n", input.SessionID)
	case HookEventSessionEnd:
		fmt.Printf("Session ended: %s\n", input.SessionID)
	case HookEventSubagentStart:
		fmt.Printf("Subagent started: %s\n", input.AgentID)
	}
	return HookOutput{Continue: true}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
