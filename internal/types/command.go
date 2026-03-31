// Package types contains core type definitions for the claude-code CLI.
// This file contains command-related types translated from TypeScript.
// Note: Command and Message base types are defined in tool.go
package types

import (
	"context"
)

// CommandAvailability indicates which auth/provider environments a command is available in.
type CommandAvailability string

const (
	AvailabilityClaudeAI CommandAvailability = "claude-ai"
	AvailabilityConsole  CommandAvailability = "console"
)

// CommandResultDisplay indicates how to display a command result.
type CommandResultDisplay string

const (
	DisplaySkip   CommandResultDisplay = "skip"
	DisplaySystem CommandResultDisplay = "system"
	DisplayUser   CommandResultDisplay = "user"
)

// ResumeEntrypoint indicates how a session resume was initiated.
type ResumeEntrypoint string

const (
	ResumeEntrypointCLIFlag               ResumeEntrypoint = "cli_flag"
	ResumeEntrypointSlashCommandPicker    ResumeEntrypoint = "slash_command_picker"
	ResumeEntrypointSlashCommandSessionId ResumeEntrypoint = "slash_command_session_id"
	ResumeEntrypointSlashCommandTitle     ResumeEntrypoint = "slash_command_title"
	ResumeEntrypointFork                  ResumeEntrypoint = "fork"
)

// LocalCommandResult is the result of executing a local command.
type LocalCommandResult struct {
	Type string `json:"type"` // text | compact | skip

	// For type="text"
	Value string `json:"value,omitempty"`

	// For type="compact"
	CompactionResult *CompactionResult `json:"compactionResult,omitempty"`
	DisplayText      string            `json:"displayText,omitempty"`
}

// CompactionResult represents the result of context compaction.
type CompactionResult struct {
	ArchivedCount int      `json:"archivedCount"`
	KeptCount     int      `json:"keptCount"`
	Summary       string   `json:"summary,omitempty"`
	RemovedUuids  []string `json:"removedUuids,omitempty"`
}

// PromptCommand extends Command with prompt-specific fields.
type PromptCommand struct {
	Command
	ProgressMessage       string      `json:"progressMessage,omitempty"`
	ContentLength         int         `json:"contentLength,omitempty"`
	ArgNames              []string    `json:"argNames,omitempty"`
	AllowedTools          []string    `json:"allowedTools,omitempty"`
	Model                 string      `json:"model,omitempty"`
	Source                string      `json:"source,omitempty"` // userSettings | projectSettings | builtin | mcp | plugin | bundled
	PluginInfo            *PluginInfo `json:"pluginInfo,omitempty"`
	DisableNonInteractive bool        `json:"disableNonInteractive,omitempty"`
	Hooks                 interface{} `json:"hooks,omitempty"` // HooksSettings
	SkillRoot             string      `json:"skillRoot,omitempty"`
	Context               string      `json:"context,omitempty"` // inline | fork
	Agent                 string      `json:"agent,omitempty"`
	Effort                interface{} `json:"effort,omitempty"` // EffortValue
	Paths                 []string    `json:"paths,omitempty"`

	// Execution functions
	GetPromptFunc func(args string, ctx context.Context) ([]interface{}, error) `json:"-"`
}

// LocalCommand extends Command with local-specific fields.
type LocalCommand struct {
	Command
	SupportsNonInteractive bool `json:"supportsNonInteractive,omitempty"`

	// Execution function
	LoadFunc func() (LocalCommandModule, error) `json:"-"`
}

// LocalJSXCommand extends Command with JSX-specific fields.
type LocalJSXCommand struct {
	Command
	LoadFunc func() (LocalJSXCommandModule, error) `json:"-"`
}

// PluginInfo contains plugin-related information for a command.
type PluginInfo struct {
	PluginManifest interface{} `json:"pluginManifest"`
	Repository     string      `json:"repository"`
}

// LocalCommandModule is the interface for lazy-loaded local commands.
type LocalCommandModule interface {
	Call(args string, ctx *LocalJSXCommandContext) (*LocalCommandResult, error)
}

// LocalJSXCommandModule is the interface for lazy-loaded JSX commands.
type LocalJSXCommandModule interface {
	Call(onDone LocalJSXCommandOnDone, ctx *LocalJSXCommandContext, args string) (interface{}, error)
}

// LocalJSXCommandContext provides context for command execution.
type LocalJSXCommandContext struct {
	// ToolUseContext fields embedded
	Cwd               string                `json:"cwd"`
	Aborted           bool                  `json:"aborted,omitempty"`
	PermissionContext ToolPermissionContext `json:"permissionContext"`
	SessionId         string                `json:"sessionId,omitempty"`

	// LocalJSXCommandContext specific fields
	CanUseTool               func(toolName string) bool                                               `json:"-"`
	SetMessages              func(updater func([]Message) []Message)                                  `json:"-"`
	Options                  CommandContextOptions                                                    `json:"options"`
	OnChangeAPIKey           func()                                                                   `json:"-"`
	OnChangeDynamicMcpConfig func(func(map[string]interface{}))                                       `json:"-"`
	OnInstallIDEExtension    func(ide string)                                                         `json:"-"`
	Resume                   func(sessionId string, log LogOption, entrypoint ResumeEntrypoint) error `json:"-"`
}

// CommandContextOptions contains options for command context.
type CommandContextOptions struct {
	DynamicMcpConfig      map[string]interface{} `json:"dynamicMcpConfig,omitempty"`
	IDEInstallationStatus *IDEInstallStatus      `json:"ideInstallationStatus,omitempty"`
	Theme                 string                 `json:"theme"`
}

// IDEInstallStatus represents IDE extension installation status.
type IDEInstallStatus struct {
	IDE       string `json:"ide"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
}

// LocalJSXCommandOnDone is the callback when a command completes.
type LocalJSXCommandOnDone func(result string, options *CommandDoneOptions)

// CommandDoneOptions contains options for command completion.
type CommandDoneOptions struct {
	Display         CommandResultDisplay `json:"display,omitempty"`
	ShouldQuery     bool                 `json:"shouldQuery,omitempty"`
	MetaMessages    []string             `json:"metaMessages,omitempty"`
	NextInput       string               `json:"nextInput,omitempty"`
	SubmitNextInput bool                 `json:"submitNextInput,omitempty"`
}
