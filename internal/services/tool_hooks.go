package services

import (
	"context"
	"fmt"
)

// =============================================================================
// Tool Hooks Service
// =============================================================================

// PostToolUseHooksResult represents the result of post-tool-use hooks
type PostToolUseHooksResult struct {
	Message              interface{}
	UpdatedMCPToolOutput interface{}
}

// ToolHooksService handles tool-related hooks
type ToolHooksService struct {
	analyticsLogger AnalyticsLogger
}

// NewToolHooksService creates a new tool hooks service
func NewToolHooksService() *ToolHooksService {
	return &ToolHooksService{}
}

// SetAnalyticsLogger sets the analytics logger
func (s *ToolHooksService) SetAnalyticsLogger(logger AnalyticsLogger) {
	s.analyticsLogger = logger
}

// RunPostToolUseHooks runs post-tool-use hooks
func (s *ToolHooksService) RunPostToolUseHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	messageID string,
	toolInput map[string]interface{},
	toolResponse interface{},
	requestID string,
	mcpServerType McpServerType,
	mcpServerBaseUrl string,
	toolCtx *ToolExecutionContext,
) <-chan PostToolUseHooksResult {
	resultChan := make(chan PostToolUseHooksResult, 10)

	go func() {
		defer close(resultChan)

		// Execute post-tool hooks
		hooks := s.getPostToolHooks(tool.Name())

		for _, hook := range hooks {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := hook.Execute(ctx, tool, toolInput, toolResponse, toolCtx)
				if err != nil {
					if s.analyticsLogger != nil {
						s.analyticsLogger.LogEvent("tengu_post_tool_hook_error", map[string]interface{}{
							"messageID": messageID,
							"toolName":  tool.Name(),
							"isMcp":     tool.IsMcp(),
							"error":     err.Error(),
						})
					}
					continue
				}

				if result.Message != nil {
					resultChan <- PostToolUseHooksResult{
						Message: result.Message,
					}
				}

				if result.BlockingError != "" {
					resultChan <- PostToolUseHooksResult{
						Message: createAttachmentMessage(Attachment{
							Type:          "hook_blocking_error",
							HookName:      fmt.Sprintf("PostToolUse:%s", tool.Name()),
							ToolUseID:     toolUseID,
							HookEvent:     "PostToolUse",
							BlockingError: result.BlockingError,
						}),
					}
				}

				if result.PreventContinuation {
					resultChan <- PostToolUseHooksResult{
						Message: createAttachmentMessage(Attachment{
							Type:      "hook_stopped_continuation",
							Message:   result.StopReason,
							HookName:  fmt.Sprintf("PostToolUse:%s", tool.Name()),
							ToolUseID: toolUseID,
							HookEvent: "PostToolUse",
						}),
					}
					return
				}

				if len(result.AdditionalContexts) > 0 {
					resultChan <- PostToolUseHooksResult{
						Message: createAttachmentMessage(Attachment{
							Type:      "hook_additional_context",
							Content:   result.AdditionalContexts,
							HookName:  fmt.Sprintf("PostToolUse:%s", tool.Name()),
							ToolUseID: toolUseID,
							HookEvent: "PostToolUse",
						}),
					}
				}

				// MCP tool output modification
				if result.UpdatedMCPToolOutput != nil && tool.IsMcp() {
					resultChan <- PostToolUseHooksResult{
						UpdatedMCPToolOutput: result.UpdatedMCPToolOutput,
					}
				}
			}
		}
	}()

	return resultChan
}

// RunPostToolUseFailureHooks runs post-tool-use failure hooks
func (s *ToolHooksService) RunPostToolUseFailureHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	messageID string,
	toolInput map[string]interface{},
	errorMsg string,
	isInterrupt bool,
	requestID string,
	mcpServerType McpServerType,
	mcpServerBaseUrl string,
	toolCtx *ToolExecutionContext,
) <-chan PostToolUseHooksResult {
	resultChan := make(chan PostToolUseHooksResult, 10)

	go func() {
		defer close(resultChan)

		hooks := s.getPostToolFailureHooks(tool.Name())

		for _, hook := range hooks {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := hook.ExecuteFailure(ctx, tool, toolInput, errorMsg, isInterrupt, toolCtx)
				if err != nil {
					if s.analyticsLogger != nil {
						s.analyticsLogger.LogEvent("tengu_post_tool_failure_hook_error", map[string]interface{}{
							"messageID": messageID,
							"toolName":  tool.Name(),
							"isMcp":     tool.IsMcp(),
							"error":     err.Error(),
						})
					}
					continue
				}

				if result.Message != nil {
					resultChan <- PostToolUseHooksResult{
						Message: result.Message,
					}
				}

				if result.BlockingError != "" {
					resultChan <- PostToolUseHooksResult{
						Message: createAttachmentMessage(Attachment{
							Type:          "hook_blocking_error",
							HookName:      fmt.Sprintf("PostToolUseFailure:%s", tool.Name()),
							ToolUseID:     toolUseID,
							HookEvent:     "PostToolUseFailure",
							BlockingError: result.BlockingError,
						}),
					}
				}

				if len(result.AdditionalContexts) > 0 {
					resultChan <- PostToolUseHooksResult{
						Message: createAttachmentMessage(Attachment{
							Type:      "hook_additional_context",
							Content:   result.AdditionalContexts,
							HookName:  fmt.Sprintf("PostToolUseFailure:%s", tool.Name()),
							ToolUseID: toolUseID,
							HookEvent: "PostToolUseFailure",
						}),
					}
				}
			}
		}
	}()

	return resultChan
}

// =============================================================================
// Pre-Tool Hooks
// =============================================================================

// PreToolHookResultInternal represents the internal result of a pre-tool hook
type PreToolHookResultInternal struct {
	Message                      interface{}
	BlockingError                string
	PreventContinuation          bool
	StopReason                   string
	PermissionBehavior           string
	HookSource                   string
	HookPermissionDecisionReason string
	UpdatedInput                 map[string]interface{}
	AdditionalContexts           []string
}

// RunPreToolUseHooks runs pre-tool-use hooks
func (s *ToolHooksService) RunPreToolUseHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	messageID string,
	toolInput map[string]interface{},
	requestID string,
	mcpServerType McpServerType,
	mcpServerBaseUrl string,
	toolCtx *ToolExecutionContext,
) <-chan PreToolHookResultInternal {
	resultChan := make(chan PreToolHookResultInternal, 10)

	go func() {
		defer close(resultChan)

		hooks := s.getPreToolHooks(tool.Name())

		for _, hook := range hooks {
			select {
			case <-ctx.Done():
				resultChan <- PreToolHookResultInternal{
					Message: createAttachmentMessage(Attachment{
						Type:      "hook_cancelled",
						HookName:  fmt.Sprintf("PreToolUse:%s", tool.Name()),
						ToolUseID: toolUseID,
						HookEvent: "PreToolUse",
					}),
				}
				return
			default:
				result, err := hook.ExecutePre(ctx, tool, toolInput, toolCtx)
				if err != nil {
					if s.analyticsLogger != nil {
						s.analyticsLogger.LogEvent("tengu_pre_tool_hook_error", map[string]interface{}{
							"messageID": messageID,
							"toolName":  tool.Name(),
							"isMcp":     tool.IsMcp(),
							"error":     err.Error(),
						})
					}
					continue
				}

				if result.Message != nil {
					resultChan <- PreToolHookResultInternal{
						Message: result.Message,
					}
				}

				if result.BlockingError != "" {
					denialMessage := s.getPreToolHookBlockingMessage(
						fmt.Sprintf("PreToolUse:%s", tool.Name()),
						result.BlockingError,
					)
					resultChan <- PreToolHookResultInternal{
						PermissionBehavior: "deny",
						BlockingError:      denialMessage,
					}
				}

				if result.PreventContinuation {
					resultChan <- PreToolHookResultInternal{
						PreventContinuation: true,
						StopReason:          result.StopReason,
					}
				}

				if result.PermissionBehavior != "" {
					resultChan <- PreToolHookResultInternal{
						PermissionBehavior:           result.PermissionBehavior,
						HookSource:                   result.HookSource,
						HookPermissionDecisionReason: result.HookPermissionDecisionReason,
						UpdatedInput:                 result.UpdatedInput,
					}
				}

				if result.UpdatedInput != nil && result.PermissionBehavior == "" {
					resultChan <- PreToolHookResultInternal{
						UpdatedInput: result.UpdatedInput,
					}
				}

				if len(result.AdditionalContexts) > 0 {
					resultChan <- PreToolHookResultInternal{
						Message: createAttachmentMessage(Attachment{
							Type:      "hook_additional_context",
							Content:   result.AdditionalContexts,
							HookName:  fmt.Sprintf("PreToolUse:%s", tool.Name()),
							ToolUseID: toolUseID,
							HookEvent: "PreToolUse",
						}),
					}
				}
			}
		}
	}()

	return resultChan
}

// =============================================================================
// Hook Registry
// =============================================================================

// HookInterface represents a hook that can be executed
type HookInterface interface {
	Execute(ctx context.Context, tool Tool, input map[string]interface{}, output interface{}, toolCtx *ToolExecutionContext) (*HookExecutionResult, error)
	ExecuteFailure(ctx context.Context, tool Tool, input map[string]interface{}, errorMsg string, isInterrupt bool, toolCtx *ToolExecutionContext) (*HookExecutionResult, error)
	ExecutePre(ctx context.Context, tool Tool, input map[string]interface{}, toolCtx *ToolExecutionContext) (*PreHookExecutionResult, error)
	Name() string
}

// HookExecutionResult represents the result of hook execution
type HookExecutionResult struct {
	Message              interface{}
	BlockingError        string
	PreventContinuation  bool
	StopReason           string
	UpdatedMCPToolOutput interface{}
	AdditionalContexts   []string
}

// PreHookExecutionResult represents the result of pre-hook execution
type PreHookExecutionResult struct {
	Message                      interface{}
	BlockingError                string
	PreventContinuation          bool
	StopReason                   string
	PermissionBehavior           string
	HookSource                   string
	HookPermissionDecisionReason string
	UpdatedInput                 map[string]interface{}
	AdditionalContexts           []string
}

// HookRegistry manages registered hooks
type HookRegistry struct {
	preToolHooks         map[string][]HookInterface
	postToolHooks        map[string][]HookInterface
	postToolFailureHooks map[string][]HookInterface
}

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		preToolHooks:         make(map[string][]HookInterface),
		postToolHooks:        make(map[string][]HookInterface),
		postToolFailureHooks: make(map[string][]HookInterface),
	}
}

// RegisterPreToolHook registers a pre-tool hook
func (r *HookRegistry) RegisterPreToolHook(toolName string, hook HookInterface) {
	r.preToolHooks[toolName] = append(r.preToolHooks[toolName], hook)
}

// RegisterPostToolHook registers a post-tool hook
func (r *HookRegistry) RegisterPostToolHook(toolName string, hook HookInterface) {
	r.postToolHooks[toolName] = append(r.postToolHooks[toolName], hook)
}

// RegisterPostToolFailureHook registers a post-tool failure hook
func (r *HookRegistry) RegisterPostToolFailureHook(toolName string, hook HookInterface) {
	r.postToolFailureHooks[toolName] = append(r.postToolFailureHooks[toolName], hook)
}

// Global hook registry
var globalHookRegistry = NewHookRegistry()

// getPreToolHooks gets pre-tool hooks for a tool
func (s *ToolHooksService) getPreToolHooks(toolName string) []HookInterface {
	hooks := globalHookRegistry.preToolHooks[toolName]
	if hooks == nil {
		return []HookInterface{}
	}
	return hooks
}

// getPostToolHooks gets post-tool hooks for a tool
func (s *ToolHooksService) getPostToolHooks(toolName string) []HookInterface {
	hooks := globalHookRegistry.postToolHooks[toolName]
	if hooks == nil {
		return []HookInterface{}
	}
	return hooks
}

// getPostToolFailureHooks gets post-tool failure hooks for a tool
func (s *ToolHooksService) getPostToolFailureHooks(toolName string) []HookInterface {
	hooks := globalHookRegistry.postToolFailureHooks[toolName]
	if hooks == nil {
		return []HookInterface{}
	}
	return hooks
}

// getPreToolHookBlockingMessage gets the blocking message for a pre-tool hook
func (s *ToolHooksService) getPreToolHookBlockingMessage(hookName, blockingError string) string {
	return fmt.Sprintf("Hook %s blocked execution: %s", hookName, blockingError)
}

// =============================================================================
// Resolve Hook Permission Decision
// =============================================================================

// ResolveHookPermissionDecision resolves a hook permission decision
func ResolveHookPermissionDecision(
	hookPermissionResult *PermissionResult,
	tool Tool,
	input map[string]interface{},
	toolCtx *ToolExecutionContext,
	canUseTool CanUseToolFn,
	assistant *AssistantMessage,
	toolUseID string,
) (*PermissionDecision, map[string]interface{}, error) {
	requiresInteraction := false // tool.RequiresUserInteraction()
	requireCanUseTool := false   // toolCtx.RequireCanUseTool

	if hookPermissionResult != nil && hookPermissionResult.Behavior == "allow" {
		hookInput := hookPermissionResult.UpdatedInput
		if hookInput == nil {
			hookInput = input
		}

		interactionSatisfied := requiresInteraction && hookPermissionResult.UpdatedInput != nil

		if (requiresInteraction && !interactionSatisfied) || requireCanUseTool {
			decision, err := canUseTool(tool, hookInput, toolCtx, assistant, toolUseID)
			if err != nil {
				return nil, hookInput, err
			}
			return decision, hookInput, nil
		}

		// Check rule-based permissions
		ruleCheck := CheckRuleBasedPermissions(tool, hookInput, toolCtx)
		if ruleCheck == nil {
			return &PermissionDecision{
				Behavior:       hookPermissionResult.Behavior,
				Message:        hookPermissionResult.Message,
				UpdatedInput:   hookPermissionResult.UpdatedInput,
				DecisionReason: hookPermissionResult.DecisionReason,
			}, hookInput, nil
		}

		if ruleCheck.Behavior == "deny" {
			return &PermissionDecision{
				Behavior:       "deny",
				Message:        ruleCheck.Message,
				DecisionReason: ruleCheck.DecisionReason,
			}, hookInput, nil
		}

		decision, err := canUseTool(tool, hookInput, toolCtx, assistant, toolUseID)
		if err != nil {
			return nil, hookInput, err
		}
		return decision, hookInput, nil
	}

	if hookPermissionResult != nil && hookPermissionResult.Behavior == "deny" {
		return &PermissionDecision{
			Behavior:       "deny",
			Message:        hookPermissionResult.Message,
			DecisionReason: hookPermissionResult.DecisionReason,
		}, input, nil
	}

	// No hook decision or 'ask' — normal permission flow
	askInput := input
	if hookPermissionResult != nil && hookPermissionResult.Behavior == "ask" && hookPermissionResult.UpdatedInput != nil {
		askInput = hookPermissionResult.UpdatedInput
	}

	decision, err := canUseTool(tool, askInput, toolCtx, assistant, toolUseID)
	if err != nil {
		return nil, askInput, err
	}
	return decision, askInput, nil
}

// =============================================================================
// Attachment Helper
// =============================================================================

// Attachment represents an attachment message
type Attachment struct {
	Type           string
	Message        string
	HookName       string
	ToolUseID      string
	HookEvent      string
	BlockingError  string
	Content        interface{}
	Decision       string
	PlanFilePath   string
	PlanContent    string
	IsSubAgent     bool
	ReminderType   string
	Skills         []SkillAttachment
	TaskID         string
	TaskType       string
	Description    string
	Status         string
	DeltaSummary   string
	OutputFilePath string
}

// SkillAttachment represents a skill attachment
type SkillAttachment struct {
	Name    string
	Path    string
	Content string
}

// createAttachmentMessage creates an attachment message
func createAttachmentMessage(att Attachment) interface{} {
	return map[string]interface{}{
		"type":       "attachment",
		"attachment": att,
	}
}
