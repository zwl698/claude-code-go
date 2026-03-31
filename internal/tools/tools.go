package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"claude-code-go/internal/types"
)

// BaseTool provides common functionality for all tools.
type BaseTool struct {
	name          string
	aliases       []string
	description   string
	inputSchema   types.ToolInputJSONSchema
	isEnabled     bool
	isReadOnly    bool
	isDestructive bool
}

// NewBaseTool creates a new base tool.
func NewBaseTool(name string, description string) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		isEnabled:   true,
		isReadOnly:  false,
	}
}

func (t *BaseTool) Name() string      { return t.name }
func (t *BaseTool) Aliases() []string { return t.aliases }
func (t *BaseTool) Description(ctx context.Context, input json.RawMessage, options types.ToolOptions) (string, error) {
	return t.description, nil
}
func (t *BaseTool) InputSchema() types.ToolInputJSONSchema       { return t.inputSchema }
func (t *BaseTool) IsEnabled() bool                              { return t.isEnabled }
func (t *BaseTool) IsConcurrencySafe(input json.RawMessage) bool { return false }
func (t *BaseTool) IsReadOnly(input json.RawMessage) bool        { return t.isReadOnly }
func (t *BaseTool) IsDestructive(input json.RawMessage) bool     { return t.isDestructive }

func (t *BaseTool) CheckPermissions(ctx context.Context, input json.RawMessage, context *types.ToolContext) (*types.PermissionResult, error) {
	return &types.PermissionResult{
		Behavior: types.PermissionBehaviorAllow,
	}, nil
}

func (t *BaseTool) UserFacingName(input json.RawMessage) string {
	return t.name
}

func (t *BaseTool) MapToolResultToAPI(content interface{}, toolUseID string) (interface{}, error) {
	return map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
		"content":     content,
	}, nil
}

func (t *BaseTool) RenderToolUseMessage(input json.RawMessage, options types.ToolRenderOptions) string {
	return fmt.Sprintf("Using tool: %s", t.name)
}

func (t *BaseTool) MaxResultSizeChars() int {
	return 100000 // Default max result size
}

// ========================================
// Bash Tool
// ========================================

// BashTool executes shell commands.
type BashTool struct {
	*BaseTool
}

func NewBashTool() *BashTool {
	return &BashTool{
		BaseTool: &BaseTool{
			name:        "Bash",
			description: "Execute shell commands",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"command": {
						"type":        "string",
						"description": "The command to execute",
					},
					"timeout": {
						"type":        "number",
						"description": "Timeout in milliseconds",
					},
				},
			},
			isEnabled:     true,
			isReadOnly:    false,
			isDestructive: true,
		},
	}
}

func (t *BashTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Check permissions
	permResult, err := t.CheckPermissions(ctx, args, toolCtx)
	if err != nil {
		return nil, err
	}
	if permResult.Behavior == types.PermissionBehaviorDeny {
		return &types.ToolResult{
			Error:     fmt.Errorf("permission denied: %s", permResult.Message),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	// Execute command
	timeout := time.Duration(input.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", input.Command)
	// Use current working directory if available
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &types.ToolResult{
			Output:    string(output),
			Error:     fmt.Errorf("command failed: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	return &types.ToolResult{
		Output:    string(output),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

func (t *BashTool) IsConcurrencySafe(input json.RawMessage) bool { return false }

// ========================================
// File Read Tool
// ========================================

// FileReadTool reads file contents.
type FileReadTool struct {
	*BaseTool
}

func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: &BaseTool{
			name:        "Read",
			description: "Read file contents",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The path to the file to read",
					},
					"offset": {
						"type":        "number",
						"description": "Line number to start reading from",
					},
					"limit": {
						"type":        "number",
						"description": "Number of lines to read",
					},
				},
			},
			isEnabled:  true,
			isReadOnly: true,
		},
	}
}

func (t *FileReadTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		FilePath string `json:"file_path"`
		Offset   int    `json:"offset,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Read file
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return &types.ToolResult{
			Error:     fmt.Errorf("failed to read file: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	// Handle offset and limit
	lines := strings.Split(string(content), "\n")
	if input.Offset > 0 && input.Offset < len(lines) {
		lines = lines[input.Offset:]
	}
	if input.Limit > 0 && input.Limit < len(lines) {
		lines = lines[:input.Limit]
	}

	return &types.ToolResult{
		Output:    strings.Join(lines, "\n"),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

// ========================================
// File Write Tool
// ========================================

// FileWriteTool writes content to files.
type FileWriteTool struct {
	*BaseTool
}

func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: &BaseTool{
			name:        "Write",
			description: "Write content to a file",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The path to write to",
					},
					"content": {
						"type":        "string",
						"description": "The content to write",
					},
				},
			},
			isEnabled:     true,
			isReadOnly:    false,
			isDestructive: true,
		},
	}
}

func (t *FileWriteTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Write file
	err := os.WriteFile(input.FilePath, []byte(input.Content), 0644)
	if err != nil {
		return &types.ToolResult{
			Error:     fmt.Errorf("failed to write file: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	return &types.ToolResult{
		Output:    fmt.Sprintf("Successfully wrote to %s", input.FilePath),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

// ========================================
// File Edit Tool
// ========================================

// FileEditTool edits files with string replacement.
type FileEditTool struct {
	*BaseTool
}

func NewFileEditTool() *FileEditTool {
	return &FileEditTool{
		BaseTool: &BaseTool{
			name:        "Edit",
			description: "Edit a file with string replacement",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The path to the file to edit",
					},
					"old_string": {
						"type":        "string",
						"description": "The text to replace",
					},
					"new_string": {
						"type":        "string",
						"description": "The replacement text",
					},
					"replace_all": {
						"type":        "boolean",
						"description": "Replace all occurrences",
					},
				},
			},
			isEnabled:     true,
			isReadOnly:    false,
			isDestructive: true,
		},
	}
}

func (t *FileEditTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		FilePath   string `json:"file_path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Read file
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return &types.ToolResult{
			Error:     fmt.Errorf("failed to read file: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	// Replace string - check for uniqueness when not replacing all
	oldContent := string(content)
	var newContent string
	if input.ReplaceAll {
		newContent = strings.ReplaceAll(oldContent, input.OldString, input.NewString)
	} else {
		count := strings.Count(oldContent, input.OldString)
		if count == 0 {
			return &types.ToolResult{
				Error:     fmt.Errorf("old_string not found in file"),
				ToolUseID: toolCtx.ToolUseId,
			}, nil
		}
		if count > 1 {
			return &types.ToolResult{
				Error:     fmt.Errorf("old_string appears %d times; either provide more context to make it unique or use replace_all", count),
				ToolUseID: toolCtx.ToolUseId,
			}, nil
		}
		// Exactly one occurrence - replace it
		newContent = strings.Replace(oldContent, input.OldString, input.NewString, 1)
	}

	// Write back
	err = os.WriteFile(input.FilePath, []byte(newContent), 0644)
	if err != nil {
		return &types.ToolResult{
			Error:     fmt.Errorf("failed to write file: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	return &types.ToolResult{
		Output:    fmt.Sprintf("Successfully edited %s", input.FilePath),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

// ========================================
// Glob Tool
// ========================================

// GlobTool finds files matching patterns.
type GlobTool struct {
	*BaseTool
}

func NewGlobTool() *GlobTool {
	return &GlobTool{
		BaseTool: &BaseTool{
			name:        "Glob",
			description: "Find files matching a pattern",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"pattern": {
						"type":        "string",
						"description": "The glob pattern to match",
					},
					"path": {
						"type":        "string",
						"description": "The directory to search in",
					},
				},
			},
			isEnabled:  true,
			isReadOnly: true,
		},
	}
}

func (t *GlobTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Path == "" {
		input.Path = "."
	}

	// Use filepath.Glob
	matches, err := filepath.Glob(filepath.Join(input.Path, input.Pattern))
	if err != nil {
		return &types.ToolResult{
			Error:     fmt.Errorf("glob failed: %w", err),
			ToolUseID: toolCtx.ToolUseId,
		}, nil
	}

	return &types.ToolResult{
		Output:    strings.Join(matches, "\n"),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

// ========================================
// Grep Tool
// ========================================

// GrepTool searches for patterns in files.
type GrepTool struct {
	*BaseTool
}

func NewGrepTool() *GrepTool {
	return &GrepTool{
		BaseTool: &BaseTool{
			name:        "Grep",
			description: "Search for patterns in files",
			inputSchema: types.ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"pattern": {
						"type":        "string",
						"description": "The pattern to search for",
					},
					"path": {
						"type":        "string",
						"description": "The file or directory to search in",
					},
					"type": {
						"type":        "string",
						"description": "File type to search",
					},
				},
			},
			isEnabled:  true,
			isReadOnly: true,
		},
	}
}

func (t *GrepTool) Call(ctx context.Context, args json.RawMessage, toolCtx *types.ToolContext, canUseTool types.CanUseToolFunc, parentMessage *types.Message, onProgress func(progress interface{})) (*types.ToolResult, error) {
	var input struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path,omitempty"`
		Type    string `json:"type,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Path == "" {
		input.Path = "."
	}

	// Use ripgrep if available, fallback to grep
	cmd := exec.CommandContext(ctx, "rg", "--no-heading", "--with-filename", "--line-number", input.Pattern, input.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to grep
		cmd = exec.CommandContext(ctx, "grep", "-rn", input.Pattern, input.Path)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return &types.ToolResult{
				Output:    "",
				ToolUseID: toolCtx.ToolUseId,
			}, nil
		}
	}

	return &types.ToolResult{
		Output:    string(output),
		ToolUseID: toolCtx.ToolUseId,
	}, nil
}

// ========================================
// Tool Registry
// ========================================

// Registry manages all available tools.
type Registry struct {
	tools map[string]types.Tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]types.Tool),
	}

	// Register built-in tools
	r.Register(NewBashTool())
	r.Register(NewFileReadTool())
	r.Register(NewFileWriteTool())
	r.Register(NewFileEditTool())
	r.Register(NewGlobTool())
	r.Register(NewGrepTool())

	return r
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool types.Tool) {
	r.tools[tool.Name()] = tool
	for _, alias := range tool.Aliases() {
		r.tools[alias] = tool
	}
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (types.Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools.
func (r *Registry) List() []types.Tool {
	result := make([]types.Tool, 0, len(r.tools))
	seen := make(map[string]bool)
	for _, tool := range r.tools {
		if !seen[tool.Name()] {
			result = append(result, tool)
			seen[tool.Name()] = true
		}
	}
	return result
}

// ListEnabled returns all enabled tools.
func (r *Registry) ListEnabled() []types.Tool {
	result := make([]types.Tool, 0)
	seen := make(map[string]bool)
	for _, tool := range r.tools {
		if !seen[tool.Name()] && tool.IsEnabled() {
			result = append(result, tool)
			seen[tool.Name()] = true
		}
	}
	return result
}
