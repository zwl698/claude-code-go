package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Tool Execution Service
// =============================================================================

// HookTimingDisplayThresholdMs is the minimum total hook duration to show inline timing
const HookTimingDisplayThresholdMs = 500

// SlowPhaseLogThresholdMs is the threshold for logging slow phases
const SlowPhaseLogThresholdMs = 2000

// McpServerType represents the type of MCP server
type McpServerType string

const (
	McpServerTypeStdIO         McpServerType = "stdio"
	McpServerTypeSSE           McpServerType = "sse"
	McpServerTypeHTTP          McpServerType = "http"
	McpServerTypeWS            McpServerType = "ws"
	McpServerTypeSDK           McpServerType = "sdk"
	McpServerTypeSSEIDE        McpServerType = "sse-ide"
	McpServerTypeWSIDE         McpServerType = "ws-ide"
	McpServerTypeClaudeAIProxy McpServerType = "claudeai-proxy"
)

// ToolExecutionContext contains context for tool execution
type ToolExecutionContext struct {
	AbortController         *AbortController
	Messages                []interface{}
	Options                 *ToolExecutionOptions
	GetAppState             func() *AppState
	SetAppState             func(func(*AppState) *AppState)
	AddNotification         func(notification *Notification)
	QueryTracking           *QueryTracking
	AgentID                 string
	ReadFileState           *FileStateCache
	LoadedNestedMemoryPaths map[string]bool
	PreserveToolUseResults  bool
	RequestPrompt           func() (string, error)
}

// ToolExecutionOptions contains options for tool execution
type ToolExecutionOptions struct {
	Tools                   []Tool
	McpClients              []McpServerConnection
	MainLoopModel           string
	IsNonInteractiveSession bool
	AppendSystemPrompt      string
	AgentDefinitions        *AgentDefinitions
	QuerySource             string
}

// Tool represents a tool that can be executed
type Tool interface {
	Name() string
	Description() string
	InputSchema() interface{}
	Call(ctx context.Context, input map[string]interface{}, context *ToolExecutionContext) (*ToolResult, error)
	ValidateInput(input map[string]interface{}) (*ValidationResult, error)
	MapToolResultToToolResultBlockParam(result interface{}, toolUseID string) *ToolResultBlockParam
	IsMcp() bool
	Aliases() []string
	MaxResultSizeChars() int
	GetToolUseSummary(input map[string]interface{}) string
	BackfillObservableInput(input map[string]interface{})
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Data             interface{}
	StructuredOutput interface{}
	NewMessages      []interface{}
	ContextModifier  func(ctx *ToolExecutionContext) *ToolExecutionContext
	McpMeta          interface{}
}

// ValidationResult represents the result of input validation
type ValidationResult struct {
	Result  bool
	Message string
	Code    string
}

// ToolResultBlockParam represents a tool result block parameter
type ToolResultBlockParam struct {
	Type      string      `json:"type"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error,omitempty"`
	ToolUseID string      `json:"tool_use_id"`
}

// AbortController manages abort signals
type AbortController struct {
	mu      sync.RWMutex
	aborted bool
	signal  chan struct{}
}

// NewAbortController creates a new abort controller
func NewAbortController() *AbortController {
	return &AbortController{
		signal: make(chan struct{}),
	}
}

// Abort aborts the controller
func (a *AbortController) Abort() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.aborted {
		a.aborted = true
		close(a.signal)
	}
}

// Signal returns the signal channel
func (a *AbortController) Signal() <-chan struct{} {
	return a.signal
}

// IsAborted checks if the controller is aborted
func (a *AbortController) IsAborted() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aborted
}

// AppState represents the application state
type AppState struct {
	ToolPermissionContext *ToolPermissionContext
	Mcp                   *McpState
	Tasks                 map[string]interface{}
}

// ToolPermissionContext contains tool permission settings
type ToolPermissionContext struct {
	Mode string
}

// McpState contains MCP state
type McpState struct {
	Clients []McpServerConnection
}

// McpServerConnection represents an MCP server connection
type McpServerConnection struct {
	Name   string
	Type   string
	Config *McpServerConfig
}

// McpServerConfig represents MCP server configuration
type McpServerConfig struct {
	Type  McpServerType
	URL   string
	Stdio *StdioConfig
}

// StdioConfig represents stdio configuration
type StdioConfig struct {
	Command string
	Args    []string
}

// AgentDefinitions contains agent definitions
type AgentDefinitions struct {
	ActiveAgents []AgentDefinition
}

// AgentDefinition represents an agent definition
type AgentDefinition struct {
	Name string
}

// Notification represents a notification
type Notification struct {
	Key      string
	Text     string
	Priority string
	Color    string
}

// QueryTracking contains query tracking info
type QueryTracking struct {
	ChainID string
	Depth   int
}

// FileStateCache is a cache for file states
type FileStateCache struct {
	mu    sync.RWMutex
	cache map[string]*FileState
}

// NewFileStateCache creates a new file state cache
func NewFileStateCache() *FileStateCache {
	return &FileStateCache{
		cache: make(map[string]*FileState),
	}
}

// Get gets a file state
func (c *FileStateCache) Get(path string) (*FileState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.cache[path]
	return state, ok
}

// Set sets a file state
func (c *FileStateCache) Set(path string, state *FileState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[path] = state
}

// Clear clears all file states
func (c *FileStateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*FileState)
}

// ToMap converts the cache to a map
func (c *FileStateCache) ToMap() map[string]*FileState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]*FileState, len(c.cache))
	for k, v := range c.cache {
		result[k] = v
	}
	return result
}

// =============================================================================
// Tool Use Block
// =============================================================================

// ToolUseBlock represents a tool use block from the API
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// AssistantMessage represents an assistant message
type AssistantMessage struct {
	Message    AssistantMessageContent `json:"message"`
	RequestID  string                  `json:"request_id,omitempty"`
	UUID       string                  `json:"uuid,omitempty"`
	IsApiError bool                    `json:"is_api_error_message,omitempty"`
}

// AssistantMessageContent represents the content of an assistant message
type AssistantMessageContent struct {
	ID      string        `json:"id"`
	Content []interface{} `json:"content"`
}

// PermissionDecision represents a permission decision
type PermissionDecision struct {
	Behavior       string
	Message        string
	UpdatedInput   map[string]interface{}
	UserModified   bool
	DecisionReason *PermissionDecisionReason
	ContentBlocks  []interface{}
	AcceptFeedback string
}

// PermissionDecisionReason represents the reason for a permission decision
type PermissionDecisionReason struct {
	Type       string
	HookName   string
	HookSource string
	Reason     string
	Rule       *PermissionRule
	Classifier string
	ToolResult interface{}
}

// PermissionRule represents a permission rule
type PermissionRule struct {
	Source string
}

// CanUseToolFn is the function type for checking tool usage
type CanUseToolFn func(tool Tool, input map[string]interface{}, ctx *ToolExecutionContext, assistant *AssistantMessage, toolUseID string) (*PermissionDecision, error)

// =============================================================================
// Tool Execution Service
// =============================================================================

// ToolExecutionService handles tool execution
type ToolExecutionService struct {
	mu              sync.RWMutex
	toolDecisions   map[string]*ToolDecisionInfo
	hookExecutor    ToolHookExecutor
	analyticsLogger AnalyticsLogger
	errorClassifier ErrorClassifier
}

// ToolDecisionInfo contains info about a tool decision
type ToolDecisionInfo struct {
	Decision string
	Source   string
}

// ToolHookExecutor executes tool hooks
type ToolHookExecutor interface {
	ExecutePreToolHooks(ctx context.Context, toolName string, toolUseID string, input map[string]interface{}, toolCtx *ToolExecutionContext) (<-chan PreToolHookResult, error)
	ExecutePostToolHooks(ctx context.Context, toolName string, toolUseID string, input map[string]interface{}, output interface{}, toolCtx *ToolExecutionContext) (<-chan PostToolHookResult, error)
	ExecutePostToolFailureHooks(ctx context.Context, toolName string, toolUseID string, input map[string]interface{}, error string, toolCtx *ToolExecutionContext) (<-chan PostToolHookResult, error)
}

// PreToolHookResult is the result of a pre-tool hook
type PreToolHookResult struct {
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

// PostToolHookResult is the result of a post-tool hook
type PostToolHookResult struct {
	Message              interface{}
	BlockingError        string
	PreventContinuation  bool
	StopReason           string
	UpdatedMCPToolOutput interface{}
	AdditionalContexts   []string
}

// AnalyticsLogger logs analytics events
type AnalyticsLogger interface {
	LogEvent(name string, metadata map[string]interface{})
}

// ErrorClassifier classifies errors
type ErrorClassifier interface {
	ClassifyError(err error) string
}

// NewToolExecutionService creates a new tool execution service
func NewToolExecutionService() *ToolExecutionService {
	return &ToolExecutionService{
		toolDecisions: make(map[string]*ToolDecisionInfo),
	}
}

// SetHookExecutor sets the hook executor
func (s *ToolExecutionService) SetHookExecutor(executor ToolHookExecutor) {
	s.hookExecutor = executor
}

// SetAnalyticsLogger sets the analytics logger
func (s *ToolExecutionService) SetAnalyticsLogger(logger AnalyticsLogger) {
	s.analyticsLogger = logger
}

// SetErrorClassifier sets the error classifier
func (s *ToolExecutionService) SetErrorClassifier(classifier ErrorClassifier) {
	s.errorClassifier = classifier
}

// =============================================================================
// RunToolUse - Main Tool Execution Entry Point
// =============================================================================

// RunToolUse runs a tool use operation
func (s *ToolExecutionService) RunToolUse(
	ctx context.Context,
	toolUse *ToolUseBlock,
	assistant *AssistantMessage,
	canUseTool CanUseToolFn,
	toolCtx *ToolExecutionContext,
) (<-chan MessageUpdateLazy, error) {
	resultChan := make(chan MessageUpdateLazy, 100)

	go func() {
		defer close(resultChan)

		// Find the tool
		tool := findToolByName(toolCtx.Options.Tools, toolUse.Name)

		// Check if tool exists
		if tool == nil {
			s.analyticsLogger.LogEvent("tengu_tool_use_error", map[string]interface{}{
				"error":     fmt.Sprintf("No such tool available: %s", toolUse.Name),
				"toolName":  toolUse.Name,
				"toolUseID": toolUse.ID,
				"isMcp":     false,
			})
			resultChan <- MessageUpdateLazy{
				Message: createUserMessageFromToolError(
					fmt.Sprintf("Error: No such tool available: %s", toolUse.Name),
					toolUse.ID,
					assistant.UUID,
				),
			}
			return
		}

		// Check if aborted
		if toolCtx.AbortController.IsAborted() {
			s.analyticsLogger.LogEvent("tengu_tool_use_cancelled", map[string]interface{}{
				"toolName":  tool.Name(),
				"toolUseID": toolUse.ID,
				"isMcp":     tool.IsMcp(),
			})
			resultChan <- MessageUpdateLazy{
				Message: createUserMessageFromToolError(
					"Request was aborted",
					toolUse.ID,
					assistant.UUID,
				),
			}
			return
		}

		// Execute the tool
		messages, err := s.checkPermissionsAndCallTool(
			ctx,
			tool,
			toolUse.ID,
			toolUse.Input,
			toolCtx,
			canUseTool,
			assistant,
		)

		if err != nil {
			resultChan <- MessageUpdateLazy{
				Message: createUserMessageFromToolError(
					fmt.Sprintf("Error calling tool (%s): %s", tool.Name(), err.Error()),
					toolUse.ID,
					assistant.UUID,
				),
			}
			return
		}

		for _, msg := range messages {
			resultChan <- msg
		}
	}()

	return resultChan, nil
}

// checkPermissionsAndCallTool checks permissions and calls the tool
func (s *ToolExecutionService) checkPermissionsAndCallTool(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input map[string]interface{},
	toolCtx *ToolExecutionContext,
	canUseTool CanUseToolFn,
	assistant *AssistantMessage,
) ([]MessageUpdateLazy, error) {
	resultingMessages := make([]MessageUpdateLazy, 0)

	// Validate input
	validationResult, err := tool.ValidateInput(input)
	if err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	if validationResult != nil && !validationResult.Result {
		return []MessageUpdateLazy{{
			Message: createUserMessageFromToolError(
				fmt.Sprintf("InputValidationError: %s", validationResult.Message),
				toolUseID,
				assistant.UUID,
			),
		}}, nil
	}

	processedInput := input

	// Run pre-tool hooks
	preToolHookStart := time.Now()
	preToolHookResults := s.runPreToolHooks(ctx, tool, toolUseID, processedInput, toolCtx)
	preToolHookDuration := time.Since(preToolHookStart).Milliseconds()

	for _, result := range preToolHookResults {
		if result.Message != nil {
			resultingMessages = append(resultingMessages, MessageUpdateLazy{
				Message: result.Message,
			})
		}
		if result.StopReason != "" {
			return resultingMessages, nil
		}
		if result.PermissionBehavior != "" {
			// Handle permission behavior
		}
	}

	if preToolHookDuration >= SlowPhaseLogThresholdMs {
		// Log slow hooks
	}

	// Check permissions
	permissionStart := time.Now()
	permissionDecision, err := canUseTool(tool, processedInput, toolCtx, assistant, toolUseID)
	permissionDuration := time.Since(permissionStart).Milliseconds()

	if err != nil {
		return nil, fmt.Errorf("permission check failed: %w", err)
	}

	if permissionDecision.Behavior != "allow" {
		// Permission denied
		s.analyticsLogger.LogEvent("tengu_tool_use_can_use_tool_rejected", map[string]interface{}{
			"toolName":  tool.Name(),
			"toolUseID": toolUseID,
		})

		errorMessage := permissionDecision.Message
		if errorMessage == "" {
			errorMessage = "Permission denied"
		}

		return []MessageUpdateLazy{{
			Message: createUserMessageFromToolError(errorMessage, toolUseID, assistant.UUID),
		}}, nil
	}

	if permissionDecision.UpdatedInput != nil {
		processedInput = permissionDecision.UpdatedInput
	}

	s.analyticsLogger.LogEvent("tengu_tool_use_can_use_tool_allowed", map[string]interface{}{
		"toolName":  tool.Name(),
		"toolUseID": toolUseID,
		"duration":  permissionDuration,
	})

	// Call the tool
	startTime := time.Now()
	result, err := tool.Call(ctx, processedInput, toolCtx)
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		// Handle error
		s.analyticsLogger.LogEvent("tengu_tool_use_error", map[string]interface{}{
			"toolName":  tool.Name(),
			"toolUseID": toolUseID,
			"error":     s.classifyToolError(err),
			"duration":  durationMs,
		})

		// Run post-tool failure hooks
		postFailureResults := s.runPostToolFailureHooks(ctx, tool, toolUseID, processedInput, err.Error(), toolCtx)
		for _, r := range postFailureResults {
			resultingMessages = append(resultingMessages, MessageUpdateLazy{
				Message: r.Message,
			})
		}

		return append(resultingMessages, MessageUpdateLazy{
			Message: createUserMessageFromToolError(
				fmt.Sprintf("Error: %s", err.Error()),
				toolUseID,
				assistant.UUID,
			),
		}), nil
	}

	// Log success
	s.analyticsLogger.LogEvent("tengu_tool_use_success", map[string]interface{}{
		"toolName":  tool.Name(),
		"toolUseID": toolUseID,
		"duration":  durationMs,
	})

	// Run post-tool hooks
	postToolResults := s.runPostToolHooks(ctx, tool, toolUseID, processedInput, result.Data, toolCtx)
	for _, r := range postToolResults {
		resultingMessages = append(resultingMessages, MessageUpdateLazy{
			Message: r.Message,
		})
	}

	// Map tool result to tool result block
	toolResultBlock := tool.MapToolResultToToolResultBlockParam(result.Data, toolUseID)

	resultingMessages = append(resultingMessages, MessageUpdateLazy{
		Message: createUserMessageFromToolResult(toolResultBlock, assistant.UUID),
	})

	// Add new messages from result
	if result.NewMessages != nil {
		for _, msg := range result.NewMessages {
			resultingMessages = append(resultingMessages, MessageUpdateLazy{
				Message: msg,
			})
		}
	}

	return resultingMessages, nil
}

// runPreToolHooks runs pre-tool hooks
func (s *ToolExecutionService) runPreToolHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input map[string]interface{},
	toolCtx *ToolExecutionContext,
) []PreToolHookResult {
	if s.hookExecutor == nil {
		return nil
	}

	resultChan, err := s.hookExecutor.ExecutePreToolHooks(ctx, tool.Name(), toolUseID, input, toolCtx)
	if err != nil {
		return []PreToolHookResult{{BlockingError: err.Error()}}
	}

	var results []PreToolHookResult
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// runPostToolHooks runs post-tool hooks
func (s *ToolExecutionService) runPostToolHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input map[string]interface{},
	output interface{},
	toolCtx *ToolExecutionContext,
) []PostToolHookResult {
	if s.hookExecutor == nil {
		return nil
	}

	resultChan, err := s.hookExecutor.ExecutePostToolHooks(ctx, tool.Name(), toolUseID, input, output, toolCtx)
	if err != nil {
		return []PostToolHookResult{{BlockingError: err.Error()}}
	}

	var results []PostToolHookResult
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// runPostToolFailureHooks runs post-tool failure hooks
func (s *ToolExecutionService) runPostToolFailureHooks(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input map[string]interface{},
	errorMsg string,
	toolCtx *ToolExecutionContext,
) []PostToolHookResult {
	if s.hookExecutor == nil {
		return nil
	}

	resultChan, err := s.hookExecutor.ExecutePostToolFailureHooks(ctx, tool.Name(), toolUseID, input, errorMsg, toolCtx)
	if err != nil {
		return []PostToolHookResult{{BlockingError: err.Error()}}
	}

	var results []PostToolHookResult
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// classifyToolError classifies a tool error
func (s *ToolExecutionService) classifyToolError(err error) string {
	if s.errorClassifier != nil {
		return s.errorClassifier.ClassifyError(err)
	}
	return "Error"
}

// =============================================================================
// Helper Functions
// =============================================================================

// findToolByName finds a tool by name
func findToolByName(tools []Tool, name string) Tool {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
		// Check aliases
		for _, alias := range tool.Aliases() {
			if alias == name {
				return tool
			}
		}
	}
	return nil
}

// MessageUpdateLazy represents a lazy message update
type MessageUpdateLazy struct {
	Message         interface{}
	ContextModifier *ContextModifier
}

// ContextModifier modifies context
type ContextModifier struct {
	ToolUseID     string
	ModifyContext func(ctx *ToolExecutionContext) *ToolExecutionContext
}

// createUserMessageFromToolError creates a user message from a tool error
func createUserMessageFromToolError(errorMsg, toolUseID, sourceAssistantUUID string) interface{} {
	return map[string]interface{}{
		"type": "user",
		"content": []interface{}{
			map[string]interface{}{
				"type":        "tool_result",
				"content":     fmt.Sprintf("<tool_use_error>%s</tool_use_error>", errorMsg),
				"is_error":    true,
				"tool_use_id": toolUseID,
			},
		},
		"tool_use_result":            errorMsg,
		"source_tool_assistant_uuid": sourceAssistantUUID,
	}
}

// createUserMessageFromToolResult creates a user message from a tool result
func createUserMessageFromToolResult(result *ToolResultBlockParam, sourceAssistantUUID string) interface{} {
	return map[string]interface{}{
		"type":                       "user",
		"content":                    []interface{}{result},
		"source_tool_assistant_uuid": sourceAssistantUUID,
	}
}

// =============================================================================
// Default Error Classifier
// =============================================================================

// DefaultErrorClassifier is a default error classifier
type DefaultErrorClassifier struct{}

// ClassifyError classifies an error
func (c *DefaultErrorClassifier) ClassifyError(err error) string {
	if err == nil {
		return "UnknownError"
	}

	// Check for specific error types
	errMsg := err.Error()
	if contains(errMsg, "ENOENT") {
		return "Error:ENOENT"
	}
	if contains(errMsg, "EACCES") {
		return "Error:EACCES"
	}
	if contains(errMsg, "ETIMEDOUT") {
		return "Error:ETIMEDOUT"
	}

	// Check error name
	if customErr, ok := err.(interface{ Name() string }); ok {
		name := customErr.Name()
		if name != "" && name != "Error" && len(name) > 3 {
			return name
		}
	}

	return "Error"
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Permission Result Types
// =============================================================================

// PermissionResult represents a permission result
type PermissionResult struct {
	Behavior       string
	Message        string
	UpdatedInput   map[string]interface{}
	UserModified   bool
	DecisionReason *PermissionDecisionReason
	ContentBlocks  []interface{}
	AcceptFeedback string
}

// CheckRuleBasedPermissions checks rule-based permissions
func CheckRuleBasedPermissions(tool Tool, input map[string]interface{}, ctx *ToolExecutionContext) *PermissionResult {
	appState := ctx.GetAppState()
	if appState == nil || appState.ToolPermissionContext == nil {
		return nil
	}

	// Check for deny rules
	if appState.ToolPermissionContext.Mode == "deny" {
		return &PermissionResult{
			Behavior: "deny",
			Message:  "Tool use denied by mode setting",
		}
	}

	return nil
}

// =============================================================================
// JSON Helper Functions
// =============================================================================

// JSONStringify converts a value to JSON string
func JSONStringify(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// JSONParse parses a JSON string
func JSONParse(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
