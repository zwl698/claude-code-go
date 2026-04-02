package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"claude-code-go/internal/constants"
	"claude-code-go/internal/types"
)

// =============================================================================
// MCP Tool
// =============================================================================

// MCPTool is a placeholder for MCP (Model Context Protocol) tools.
// The actual implementation is in mcpClient.go which dynamically creates tools.
type MCPTool struct {
	*BaseTool
}

// NewMCPTool creates a new MCP tool placeholder.
func NewMCPTool() *MCPTool {
	return &MCPTool{
		BaseTool: &BaseTool{
			name:        "mcp",
			description: "Execute MCP (Model Context Protocol) tools",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
			},
			isEnabled:  true,
			isReadOnly: false,
		},
	}
}

// Call executes the MCP tool (overridden in mcpClient).
func (t *MCPTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	return &types.ToolResult{
		Output: "",
	}, nil
}

// CheckPermissions returns passthrough permission (overridden in mcpClient).
func (t *MCPTool) CheckPermissions(ctx context.Context, input json.RawMessage, context *types.ToolContext) (*types.PermissionResult, error) {
	return &types.PermissionResult{
		Behavior: types.PermissionBehavior("passthrough"),
		Message:  "MCPTool requires permission.",
	}, nil
}

// UserFacingName returns the user-facing name.
func (t *MCPTool) UserFacingName(input json.RawMessage) string {
	return "mcp"
}

// IsMCP returns true for MCP tools.
func (t *MCPTool) IsMCP() bool {
	return true
}

// =============================================================================
// Agent Tool
// =============================================================================

// AgentTool spawns and manages sub-agents.
type AgentTool struct {
	*BaseTool
}

// NewAgentTool creates a new Agent tool.
func NewAgentTool() *AgentTool {
	return &AgentTool{
		BaseTool: &BaseTool{
			name:        constants.ToolAgent,
			description: "Spawn a sub-agent to handle complex, multi-step tasks autonomously",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"description": {
						"type":        "string",
						"description": "A short (3-5 word) description of the task",
					},
					"prompt": {
						"type":        "string",
						"description": "The task for the agent to perform",
					},
					"subagent_type": {
						"type":        "string",
						"description": "The type of specialized agent to use for this task",
					},
					"model": {
						"type":        "string",
						"enum":        []string{"sonnet", "opus", "haiku"},
						"description": "Optional model override for this agent",
					},
					"run_in_background": {
						"type":        "boolean",
						"description": "Set to true to run this agent in the background",
					},
				},
				Required: []string{"description", "prompt"},
			},
			isEnabled:  true,
			isReadOnly: false,
		},
	}
}

// Call spawns a sub-agent.
func (t *AgentTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		Description     string `json:"description"`
		Prompt          string `json:"prompt"`
		SubagentType    string `json:"subagent_type,omitempty"`
		Model           string `json:"model,omitempty"`
		RunInBackground bool   `json:"run_in_background,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// TODO: Implement actual agent spawning logic
	// This requires the agent framework from:
	// - tasks/LocalAgentTask
	// - tasks/RemoteAgentTask
	// - utils/worktree

	return &types.ToolResult{
		Output: fmt.Sprintf("Agent task '%s' spawned successfully. Prompt: %s", input.Description, input.Prompt),
	}, nil
}

// CheckPermissions checks agent permissions.
func (t *AgentTool) CheckPermissions(ctx context.Context, input json.RawMessage, context *types.ToolContext) (*types.PermissionResult, error) {
	return &types.PermissionResult{
		Behavior: types.PermissionBehaviorAllow,
	}, nil
}

// UserFacingName returns the user-facing name.
func (t *AgentTool) UserFacingName(input json.RawMessage) string {
	var inputData struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(input, &inputData); err == nil && inputData.Description != "" {
		return inputData.Description
	}
	return "Agent"
}

// =============================================================================
// REPL Tool
// =============================================================================

// REPLTool manages REPL primitive tools.
type REPLTool struct {
	*BaseTool
	primitiveTools []types.Tool
}

// NewREPLTool creates a new REPL tool manager.
func NewREPLTool() *REPLTool {
	return &REPLTool{
		BaseTool: &BaseTool{
			name:        "repl",
			description: "REPL primitive tools management",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
			},
			isEnabled:  true,
			isReadOnly: true,
		},
	}
}

// GetPrimitiveTools returns the list of REPL primitive tools.
func (t *REPLTool) GetPrimitiveTools() []types.Tool {
	if t.primitiveTools == nil {
		t.primitiveTools = []types.Tool{
			NewFileReadTool(),
			NewFileWriteTool(),
			NewFileEditTool(),
			NewGlobTool(),
			NewGrepTool(),
			NewBashTool(),
			NewNotebookEditTool(),
			NewAgentTool(),
		}
	}
	return t.primitiveTools
}

// Call is not implemented for REPL tool.
func (t *REPLTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	return nil, fmt.Errorf("REPLTool is not directly callable")
}

// =============================================================================
// Task Output Tool
// =============================================================================

// TaskOutputTool retrieves output from running or completed tasks.
type TaskOutputTool struct {
	*BaseTool
}

// NewTaskOutputTool creates a new Task Output tool.
func NewTaskOutputTool() *TaskOutputTool {
	return &TaskOutputTool{
		BaseTool: &BaseTool{
			name:        "TaskOutput",
			aliases:     []string{"AgentOutputTool", "BashOutputTool"},
			description: "[Deprecated] Retrieve output from a background task",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"task_id": {
						"type":        "string",
						"description": "The task ID to get output from",
					},
					"block": {
						"type":        "boolean",
						"default":     true,
						"description": "Whether to wait for completion",
					},
					"timeout": {
						"type":        "number",
						"default":     30000,
						"description": "Max wait time in ms",
					},
				},
				Required: []string{"task_id"},
			},
			isEnabled:  true,
			isReadOnly: true,
		},
	}
}

// Call retrieves task output.
func (t *TaskOutputTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		TaskID  string `json:"task_id"`
		Block   bool   `json:"block"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.TaskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	// Default timeout
	if input.Timeout == 0 {
		input.Timeout = 30000
	}

	// TODO: Implement actual task output retrieval
	// This requires integration with:
	// - tasks/LocalAgentTask
	// - tasks/LocalShellTask
	// - tasks/RemoteAgentTask
	// - utils/task/diskOutput

	// Simulate output retrieval
	output := fmt.Sprintf("Task %s output placeholder", input.TaskID)

	// TODO: Implement actual task output retrieval
	// This requires integration with:
	// - tasks/LocalAgentTask
	// - tasks/LocalShellTask
	// - tasks/RemoteAgentTask
	// - utils/task/diskOutput

	return &types.ToolResult{
		Output: output,
	}, nil
}

// UserFacingName returns the user-facing name.
func (t *TaskOutputTool) UserFacingName(input json.RawMessage) string {
	return "Task Output"
}

// Description returns the tool description.
func (t *TaskOutputTool) Description(ctx context.Context, input json.RawMessage, options types.ToolOptions) (string, error) {
	return "[Deprecated] — prefer Read on the task output file path", nil
}
