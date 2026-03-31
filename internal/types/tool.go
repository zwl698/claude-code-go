package types

import (
	"context"
	"encoding/json"
)

// ToolInputJSONSchema represents the JSON schema for a tool's input.
type ToolInputJSONSchema struct {
	Type       string                            `json:"type"`
	Properties map[string]map[string]interface{} `json:"properties,omitempty"`
	Required   []string                          `json:"required,omitempty"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Output    interface{} `json:"output"`
	Error     error       `json:"error,omitempty"`
	ToolUseID string      `json:"toolUseId"`
}

// ToolContext provides context for tool execution.
type ToolContext struct {
	Options         ToolOptions
	AbortController *AbortController
	ReadFileState   FileStateCache
	GetAppState     func() interface{}
	SetAppState     SetAppStateFunc
	Messages        []Message
	ToolUseId       string
	AgentId         AgentId
	AgentType       string
}

// ToolOptions contains options for tool execution.
type ToolOptions struct {
	Commands           []Command
	Debug              bool
	MainLoopModel      string
	Tools              []Tool
	Verbose            bool
	IsNonInteractive   bool
	AgentDefinitions   AgentDefinitionsResult
	MaxBudgetUsd       float64
	CustomSystemPrompt string
	AppendSystemPrompt string
	QuerySource        string
	RefreshTools       func() []Tool
}

// FileStateCache provides a cache for file state.
type FileStateCache interface {
	Get(path string) (interface{}, bool)
	Set(path string, value interface{})
}

// Tool interface defines the contract for all tools.
type Tool interface {
	// Name returns the tool's name.
	Name() string

	// Aliases returns optional aliases for the tool.
	Aliases() []string

	// Description returns a description of the tool.
	Description(ctx context.Context, input json.RawMessage, options ToolOptions) (string, error)

	// InputSchema returns the JSON schema for the tool's input.
	InputSchema() ToolInputJSONSchema

	// Call executes the tool.
	Call(ctx context.Context, args json.RawMessage, context *ToolContext, canUseTool CanUseToolFunc, parentMessage *Message, onProgress func(progress interface{})) (*ToolResult, error)

	// IsEnabled returns whether the tool is currently enabled.
	IsEnabled() bool

	// IsConcurrencySafe returns whether the tool can be called concurrently.
	IsConcurrencySafe(input json.RawMessage) bool

	// IsReadOnly returns whether the tool only reads (doesn't modify files).
	IsReadOnly(input json.RawMessage) bool

	// IsDestructive returns whether the tool performs destructive operations.
	IsDestructive(input json.RawMessage) bool

	// CheckPermissions checks if the tool can be used with the given input.
	CheckPermissions(ctx context.Context, input json.RawMessage, context *ToolContext) (*PermissionResult, error)
}

// CanUseToolFunc is a function type for checking tool permissions.
type CanUseToolFunc func(ctx context.Context, toolName string, input json.RawMessage) (*PermissionDecision, error)

// Message represents a message in the conversation.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	// Additional fields for different message types
	Id        string  `json:"id,omitempty"`
	Timestamp float64 `json:"timestamp,omitempty"`
}

// Command represents a slash command.
type Command struct {
	Type                   string        `json:"type"`
	Name                   string        `json:"name"`
	Aliases                []string      `json:"aliases,omitempty"`
	Description            string        `json:"description"`
	HasUserSpecifiedDesc   bool          `json:"hasUserSpecifiedDescription,omitempty"`
	IsEnabled              func() bool   `json:"-"`
	IsHidden               bool          `json:"isHidden,omitempty"`
	IsMcp                  bool          `json:"isMcp,omitempty"`
	ArgumentHint           string        `json:"argumentHint,omitempty"`
	WhenToUse              string        `json:"whenToUse,omitempty"`
	Version                string        `json:"version,omitempty"`
	DisableModelInvocation bool          `json:"disableModelInvocation,omitempty"`
	UserInvocable          bool          `json:"userInvocable,omitempty"`
	LoadedFrom             string        `json:"loadedFrom,omitempty"`
	Kind                   string        `json:"kind,omitempty"`
	Immediate              bool          `json:"immediate,omitempty"`
	IsSensitive            bool          `json:"isSensitive,omitempty"`
	UserFacingName         func() string `json:"-"`
	Availability           []string      `json:"availability,omitempty"`
}

// AgentDefinitionsResult contains the result of loading agent definitions.
type AgentDefinitionsResult struct {
	Definitions []AgentDefinition `json:"definitions"`
	Errors      []AgentLoadError  `json:"errors,omitempty"`
}

// AgentDefinition represents a defined agent.
type AgentDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Model       string   `json:"model,omitempty"`
	Color       string   `json:"color,omitempty"`
	Tools       []string `json:"tools,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
	IsCustom    bool     `json:"isCustom"`
	IsBuiltIn   bool     `json:"isBuiltIn"`
}

// AgentLoadError represents an error loading an agent.
type AgentLoadError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// ToolProgressData represents progress data from tool execution.
type ToolProgressData struct {
	Type       string      `json:"type"`
	Message    string      `json:"message,omitempty"`
	Percentage int         `json:"percentage,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

// BashProgress represents progress from the Bash tool.
type BashProgress struct {
	Command     string `json:"command"`
	Output      string `json:"output,omitempty"`
	ExitCode    int    `json:"exitCode,omitempty"`
	IsStreaming bool   `json:"isStreaming,omitempty"`
}

// MCPProgress represents progress from MCP tools.
type MCPProgress struct {
	ServerName string      `json:"serverName"`
	ToolName   string      `json:"toolName"`
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
}

// ValidationResult represents the result of input validation.
type ValidationResult struct {
	Result    bool   `json:"result"`
	Message   string `json:"message,omitempty"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

// ThinkingConfig contains configuration for thinking mode.
type ThinkingConfig struct {
	Enabled     bool   `json:"enabled"`
	BudgetToken int    `json:"budgetToken,omitempty"`
	Type        string `json:"type,omitempty"`
}

// MCPServerConnection represents a connection to an MCP server.
type MCPServerConnection struct {
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Config map[string]interface{} `json:"config"`
	Tools  []Tool                 `json:"tools,omitempty"`
}

// ServerResource represents a resource provided by an MCP server.
type ServerResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}
