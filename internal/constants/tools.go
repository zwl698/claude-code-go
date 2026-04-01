package constants

// =============================================================================
// Tool Names
// =============================================================================

const (
	// Core file tools
	ToolFileRead  = "Read"
	ToolFileWrite = "Write"
	ToolFileEdit  = "Edit"
	ToolGlob      = "Glob"
	ToolGrep      = "Grep"

	// Shell tools
	ToolBash = "Bash"

	// Web tools
	ToolWebFetch  = "WebFetch"
	ToolWebSearch = "WebSearch"

	// Task tools
	ToolTaskCreate = "Task"
	ToolTaskOutput = "TaskOutput"
	ToolTaskStop   = "TaskStop"
	ToolTaskGet    = "TaskGet"
	ToolTaskList   = "TaskList"
	ToolTaskUpdate = "TaskUpdate"

	// Agent tools
	ToolAgent           = "Agent"
	ToolAskUser         = "AskUserQuestion"
	ToolSendMessage     = "SendMessage"
	ToolSkill           = "Skill"
	ToolSyntheticOutput = "SyntheticOutput"

	// Plan mode tools
	ToolEnterPlanMode = "EnterPlanMode"
	ToolExitPlanMode  = "ExitPlanMode"

	// Workflow tools
	ToolWorkflow      = "Workflow"
	ToolEnterWorktree = "EnterWorktree"
	ToolExitWorktree  = "ExitWorktree"

	// Todo tools
	ToolTodoWrite = "TodoWrite"

	// Schedule tools
	ToolCronCreate = "CronCreate"
	ToolCronDelete = "CronDelete"
	ToolCronList   = "CronList"

	// MCP tools
	ToolMCP              = "mcp__"
	ToolListMcpResources = "ListMcpResources"
	ToolReadMcpResource  = "ReadMcpResource"

	// Notebook tools
	ToolNotebookEdit = "NotebookEdit"

	// Config tool
	ToolConfig = "Config"

	// Tool search
	ToolToolSearch = "ToolSearch"
)

// =============================================================================
// Tool Groups
// =============================================================================

// AllAgentDisallowedTools are tools that cannot be used by sub-agents
var AllAgentDisallowedTools = map[string]bool{
	ToolTaskOutput:    true,
	ToolExitPlanMode:  true,
	ToolEnterPlanMode: true,
	ToolAgent:         true,
	ToolAskUser:       true,
	ToolTaskStop:      true,
	ToolWorkflow:      true,
}

// AsyncAgentAllowedTools are tools allowed for async agents
var AsyncAgentAllowedTools = map[string]bool{
	ToolFileRead:        true,
	ToolWebSearch:       true,
	ToolTodoWrite:       true,
	ToolGrep:            true,
	ToolWebFetch:        true,
	ToolGlob:            true,
	ToolBash:            true,
	ToolFileEdit:        true,
	ToolFileWrite:       true,
	ToolNotebookEdit:    true,
	ToolSkill:           true,
	ToolSyntheticOutput: true,
	ToolToolSearch:      true,
	ToolEnterWorktree:   true,
	ToolExitWorktree:    true,
}

// CoordinatorModeAllowedTools are tools allowed in coordinator mode
var CoordinatorModeAllowedTools = map[string]bool{
	ToolAgent:           true,
	ToolTaskStop:        true,
	ToolSendMessage:     true,
	ToolSyntheticOutput: true,
}

// =============================================================================
// Tool Limits
// =============================================================================

const (
	// Maximum result size in characters
	MaxResultSizeChars = 100000

	// Maximum file size to read (10MB)
	MaxFileSizeToRead = 10 * 1024 * 1024

	// Default command timeout in milliseconds
	DefaultCommandTimeoutMs = 120000 // 2 minutes

	// Maximum command timeout in milliseconds
	MaxCommandTimeoutMs = 600000 // 10 minutes

	// Default file read limit
	DefaultFileReadLimit = 2000

	// Maximum grep output lines
	MaxGrepOutputLines = 500
)

// =============================================================================
// Tool Descriptions
// =============================================================================

const (
	DescBash = `Executes a given bash command in a persistent shell session with optional timeout, handling both safe and destructive operations.

Command Execution:
- Commands execute in the current directory (confirm with 'pwd' if uncertain)
- Non-interactive commands are preferred; avoid commands requiring user input
- Use '| cat' for commands with pagers to prevent issues
- Background long-running processes with 'is_background: true' to avoid blocking
- Never update git config, skip hooks, or run destructive git commands unless explicitly requested
- Only commit changes when explicitly asked

Parameters:
- command: The shell command to execute
- timeout: Optional timeout in milliseconds (default 120000, max 600000)
- is_background: Set true for long-running processes to run non-blocking

Safety:
- Never run destructive commands without explicit user request
- Never skip git hooks (--no-verify) without explicit request
- Never force push to main/master branches
- Always be extremely careful with file edits`

	DescRead = `Reads a file from the local filesystem with direct access capability.

File Access:
- Can read any file directly with the file path parameter
- Images (jpeg, jpg, png, gif, webp) are converted to text content automatically
- PDF files are converted to text content automatically

Parameters:
- target_file: The file path to read (relative to workspace or absolute)
- offset: Optional line number to start reading from
- limit: Optional number of lines to read

Line Format:
- Output lines are numbered starting at 1: LINE_NUMBER|LINE_CONTENT
- Large files (>1K lines) should use offset/limit for efficient reading

Guidelines:
- Always read the whole file by not providing offset/limit parameters
- Use offset/limit only when the file is too large to read at once
- Never re-read the exact same content that was just provided
- Expand chunk ranges when needed to see imports or signatures`

	DescWrite = `Writes a file to the local filesystem, overwriting if exists.

Usage Guidelines:
- Always prefer editing existing files; NEVER write new files unless explicitly required
- NEVER proactively create documentation files (*.md) or README files
- Only create documentation if explicitly requested by the User

Parameters:
- file_path: The path to write to (relative to workspace or absolute)
- contents: The content to write

File Handling:
- This tool will overwrite existing files if they exist
- For existing files, MUST read the file first to understand current content
- ALWAYS prefer editing existing files over creating new ones`

	DescEdit = `Performs exact string replacements in files.

Usage:
- Preserve exact indentation (tabs/spaces) as they appear
- ALWAYS prefer editing existing files; NEVER write new files unless required
- Only use emojis if user explicitly requests
- Avoid adding emojis unless asked

Parameters:
- file_path: The target file to modify
- old_string: The text to replace (must match exactly including whitespace)
- new_string: The replacement text (must be different from old_string)
- replace_all: Set true to replace all occurrences (default false)

Editing Rules:
- The edit will FAIL if old_string is not unique
- Provide larger context string to make it unique if needed
- Use replace_all for replacing/renaming across the file

Critical Requirements:
- Edits follow the same requirements as the single string_replace tool
- Edits are atomic: either all succeed or none are applied
- Plan edits carefully to avoid conflicts in sequential operations`

	DescGlob = `Searches for files matching a glob pattern.

Usage:
- Works fast with codebases of any size
- Returns matching file paths sorted by modification time
- Use when you need to find files by name patterns

Parameters:
- pattern: The glob pattern to match files against
  (Patterns not starting with "**/" are automatically prepended with "**/")
- target_directory: Optional directory to search in (defaults to workspace root)

Pattern Examples:
- "*.js" → finds all .js files (becomes "**/*.js")
- "**/node_modules/**" → finds all node_modules directories
- "**/test/**/test_*.ts" → finds test_*.ts files in any test directory

Guidelines:
- Always provide ONE directory or file path; [] searches whole repo
- No globs or wildcards for target_directory parameter
- Bad: ["frontend/", "backend/"] (multiple paths)
- Bad: ["src/**/utils/**"] (globs)
- Bad: ["*.ts"] or ["**/*"] (wildcard paths)`

	DescGrep = `A powerful search tool built on ripgrep.

Usage:
- Prefer grep_search for exact symbol/string searches
- Faster and respects .gitignore/.cursorignore
- Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+")
- Escape special chars for exact matches (e.g., "functionCall\\(")

Parameters:
- pattern: The regular expression pattern to search (rg --regexp)
- path: Optional file or directory to search in (defaults to workspace root)
- glob: Optional glob pattern to filter files (e.g., "*.js", "*.{ts,tsx}")
- output_mode: "content", "files_with_matches", or "count"
- case_insensitive: Set true for case-insensitive search

Output Modes:
- "content": Shows matching lines with line numbers and context (default)
- "files_with_matches": Shows only file paths
- "count": Shows match counts

Guidelines:
- Avoid overly broad glob patterns (--glob *) that bypass .gitignore
- Note: import paths may not match source file types (.js vs .ts)
- Results are capped for responsiveness`

	DescWebFetch = `Fetches content from one or more specified URLs and returns the content.

Overview:
- Takes an array of URLs as input
- Fetches the URL content, converts HTML to markdown
- Returns the fetched content for all URLs
- Use when you need to retrieve and analyze web content

Usage Notes:
- If an MCP-provided web fetch tool is available, prefer using it
- URLs must be fully-formed valid URLs
- HTTP URLs are automatically upgraded to HTTPS
- Tool is read-only and does not modify files
- Results may be summarized if content is very large
- Supports batch fetching of multiple URLs at once
- Includes self-cleaning 15-minute cache for faster responses

Redirects:
- If a URL redirects to a different host, the tool informs and provides the redirect URL`

	DescTodoWrite = `Creates and manages a structured task list for coding sessions.

Purpose:
- Track progress for complex multi-step tasks (3+ distinct steps)
- Organize non-trivial tasks requiring careful planning
- Capture new requirements when user provides multiple tasks

When to Use:
- Complex multi-step tasks
- Non-trivial tasks requiring careful planning
- User explicitly requests todo list
- User provides multiple tasks (numbered/comma-separated)

When NOT to Use:
- Single, straightforward tasks
- Trivial tasks with no organizational benefit
- Conversational/informational requests

Task States:
- pending: Not yet started
- in_progress: Currently working on (only ONE at a time)
- completed: Finished successfully
- cancelled: No longer needed

Management Rules:
- Update status in real-time
- Mark complete IMMEDIATELY after finishing
- Only ONE task in_progress at a time
- Complete current tasks before starting new ones`

	DescTask = `Launches a new agent to handle complex, multi-step tasks autonomously.

Available Agent Types:
- general-agent: General-purpose agent for researching, coding, and multi-step tasks
- explorer-agent: Fast agent for codebase exploration and searches

Usage:
- Launch multiple agents concurrently for maximum performance
- Provide detailed task description for autonomous execution
- Specify if code should be written or just research performed

Guidelines:
- Tell the agent whether to write code or just do research
- For single file reads, use read_file instead of agents
- For specific class searches, use grep_search instead
- Agent outputs are generally trusted`
)

// =============================================================================
// Tool Input Schemas
// =============================================================================

// GetBashToolSchema returns the input schema for the Bash tool
func GetBashToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Timeout in milliseconds",
			},
			"is_background": map[string]interface{}{
				"type":        "boolean",
				"description": "Run command in background",
			},
		},
		"required": []string{"command"},
	}
}

// GetReadToolSchema returns the input schema for the Read tool
func GetReadToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"target_file": map[string]interface{}{
				"type":        "string",
				"description": "The path of the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "number",
				"description": "Line number to start reading from",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Number of lines to read",
			},
		},
		"required": []string{"target_file"},
	}
}

// GetWriteToolSchema returns the input schema for the Write tool
func GetWriteToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The path of the file to write",
			},
			"contents": map[string]interface{}{
				"type":        "string",
				"description": "The contents to write",
			},
		},
		"required": []string{"file_path", "contents"},
	}
}

// GetEditToolSchema returns the input schema for the Edit tool
func GetEditToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The path of the file to modify",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

// GetGlobToolSchema returns the input schema for the Glob tool
func GetGlobToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match",
			},
			"target_directory": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in",
			},
		},
		"required": []string{"pattern"},
	}
}

// GetGrepToolSchema returns the input schema for the Grep tool
func GetGrepToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regular expression pattern to search",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory to search in",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to filter files",
			},
			"output_mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"content", "files_with_matches", "count"},
				"description": "Output mode",
			},
			"case_insensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Case-insensitive search",
			},
		},
		"required": []string{"pattern"},
	}
}

// GetWebFetchToolSchema returns the input schema for the WebFetch tool
func GetWebFetchToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"urls": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Array of URLs to fetch",
			},
		},
		"required": []string{"urls"},
	}
}

// GetTodoWriteToolSchema returns the input schema for the TodoWrite tool
func GetTodoWriteToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique identifier",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Task description",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
							"description": "Task status",
						},
					},
					"required": []string{"id", "content", "status"},
				},
				"description": "Array of todo items",
			},
			"merge": map[string]interface{}{
				"type":        "boolean",
				"description": "Merge with existing todos",
			},
		},
		"required": []string{"todos"},
	}
}

// GetTaskToolSchema returns the input schema for the Task tool
func GetTaskToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Short task description",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Detailed task instructions",
			},
			"subagent_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"general-agent", "explorer-agent"},
				"description": "Type of agent to use",
			},
			"model": map[string]interface{}{
				"type":        "number",
				"description": "Model to use",
			},
		},
		"required": []string{"description", "prompt", "subagent_type"},
	}
}
