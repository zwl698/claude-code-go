package constants

// =============================================================================
// User Message Constants
// =============================================================================

const (
	// NoContentMessage is displayed when there is no content
	NoContentMessage = "(no content)"

	// InterruptMessage is displayed when a request is interrupted by the user
	InterruptMessage = "[Request interrupted by user]"

	// InterruptMessageForToolUse is displayed when a request is interrupted during tool use
	InterruptMessageForToolUse = "[Request interrupted by user for tool use]"

	// CancelMessage is displayed when the user cancels an action
	CancelMessage = "The user doesn't want to take this action right now. STOP what you are doing and wait for the user to tell you how to proceed."

	// RejectMessage is displayed when a tool use is rejected
	RejectMessage = "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed."

	// RejectMessageWithReasonPrefix is the prefix for rejection messages with a reason
	RejectMessageWithReasonPrefix = "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). To tell you how to proceed, the user said:\n"

	// SubagentRejectMessage is displayed when a subagent's tool use is rejected
	SubagentRejectMessage = "Permission for this tool use was denied. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). Try a different approach or report the limitation to complete your task."

	// SubagentRejectMessageWithReasonPrefix is the prefix for subagent rejection messages with a reason
	SubagentRejectMessageWithReasonPrefix = "Permission for this tool use was denied. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). The user said:\n"

	// PlanRejectionPrefix is the prefix for plan rejection messages
	PlanRejectionPrefix = "The agent proposed a plan that was rejected by the user. The user chose to stay in plan mode rather than proceed with implementation.\n\nRejected plan:\n"

	// NoResponseRequested is displayed when no response is needed
	NoResponseRequested = "No response requested."

	// SyntheticToolResultPlaceholder is inserted when a tool_use block has no matching tool_result
	SyntheticToolResultPlaceholder = "[Tool result missing due to internal error]"

	// SyntheticModel is the model name for synthetic messages
	SyntheticModel = "<synthetic>"
)

// DenialWorkaroundGuidance provides shared guidance for permission denials
const DenialWorkaroundGuidance = "IMPORTANT: You *may* attempt to accomplish this action using other tools that might naturally be used to accomplish this goal, " +
	"e.g. using head instead of cat. But you *should not* attempt to work around this denial in malicious ways, " +
	"e.g. do not use your ability to run tests to execute non-test actions. " +
	"You should only try to work around this restriction in reasonable ways that do not attempt to bypass the intent behind this denial. " +
	"If you believe this capability is essential to complete the user's request, STOP and explain to the user " +
	"what you were trying to do and why you need this permission. Let the user decide how to proceed."

// AutoRejectMessage returns a rejection message for auto mode
func AutoRejectMessage(toolName string) string {
	return "Permission to use " + toolName + " has been denied. " + DenialWorkaroundGuidance
}

// DontAskRejectMessage returns a rejection message for don't ask mode
func DontAskRejectMessage(toolName string) string {
	return "Permission to use " + toolName + " has been denied because Claude Code is running in don't ask mode. " + DenialWorkaroundGuidance
}

// BuildYoloRejectionMessage builds a rejection message for auto mode classifier denials
func BuildYoloRejectionMessage(reason string) string {
	return "Permission for this action has been denied. Reason: " + reason + ". " +
		"If you have other tasks that don't depend on this action, continue working on those. " +
		DenialWorkaroundGuidance + " " +
		"To allow this type of action in the future, the user can add a Bash permission rule to their settings."
}

// BuildClassifierUnavailableMessage builds a message for when the classifier is unavailable
func BuildClassifierUnavailableMessage(toolName string, classifierModel string) string {
	return classifierModel + " is temporarily unavailable, so auto mode cannot determine the safety of " + toolName + " right now. " +
		"Wait briefly and then try this action again. " +
		"If it keeps failing, continue with other tasks that don't require this action and come back to it later. " +
		"Note: reading files, searching code, and other read-only operations do not require the classifier and can still be used."
}

// IsSyntheticMessage checks if a message content is synthetic
func IsSyntheticMessage(content string) bool {
	return content == InterruptMessage ||
		content == InterruptMessageForToolUse ||
		content == CancelMessage ||
		content == RejectMessage ||
		content == NoResponseRequested
}

// IsClassifierDenial checks if a tool result message is a classifier denial
func IsClassifierDenial(content string) bool {
	return len(content) > 39 && content[:39] == "Permission for this action has been denied"
}

// =============================================================================
// Memory Correction Hint
// =============================================================================

// MemoryCorrectionHint is appended to rejection/cancellation messages when auto-memory is enabled
const MemoryCorrectionHint = "\n\nNote: The user's next message may contain a correction or preference. Pay close attention — if they explain what went wrong or how they'd prefer you to work, consider saving that to memory for future sessions."

// WithMemoryCorrectionHint appends the memory correction hint if enabled
func WithMemoryCorrectionHint(message string, autoMemoryEnabled bool, featureEnabled bool) string {
	if autoMemoryEnabled && featureEnabled {
		return message + MemoryCorrectionHint
	}
	return message
}

// =============================================================================
// Permission Messages
// =============================================================================

// Permission request messages
const (
	// PermissionTitleFileRead is the title for file read permission
	PermissionTitleFileRead = "Read file"

	// PermissionTitleFileWrite is the title for file write permission
	PermissionTitleFileWrite = "Write file"

	// PermissionTitleFileEdit is the title for file edit permission
	PermissionTitleFileEdit = "Edit file"

	// PermissionTitleBash is the title for bash command permission
	PermissionTitleBash = "Run command"

	// PermissionTitleWebFetch is the title for web fetch permission
	PermissionTitleWebFetch = "Fetch URL"

	// PermissionTitleWebSearch is the title for web search permission
	PermissionTitleWebSearch = "Web search"

	// PermissionTitleAgent is the title for agent launch permission
	PermissionTitleAgent = "Launch agent"

	// PermissionTitleTodoWrite is the title for todo write permission
	PermissionTitleTodoWrite = "Update todo list"
)

// Permission prompts
const (
	// PermissionPromptAllow asks if the user wants to allow the action
	PermissionPromptAllow = "Do you want to allow this action?"

	// PermissionPromptAllowOnce allows the action once
	PermissionPromptAllowOnce = "Allow once"

	// PermissionPromptAllowAlways allows the action always
	PermissionPromptAllowAlways = "Always allow"

	// PermissionPromptDeny denies the action
	PermissionPromptDeny = "Deny"

	// PermissionPromptDenyAlways always denies the action
	PermissionPromptDenyAlways = "Always deny"
)

// =============================================================================
// Error Messages
// =============================================================================

// Error messages for common scenarios
const (
	// ErrFileNotFound is returned when a file is not found
	ErrFileNotFound = "file not found"

	// ErrFileTooLarge is returned when a file is too large to process
	ErrFileTooLarge = "file too large to process"

	// ErrPermissionDenied is returned when permission is denied
	ErrPermissionDenied = "permission denied"

	// ErrInvalidInput is returned when input is invalid
	ErrInvalidInput = "invalid input"

	// ErrAPITimeout is returned when the API request times out
	ErrAPITimeout = "API request timed out"

	// ErrAPIError is returned for general API errors
	ErrAPIError = "API error occurred"

	// ErrContextLengthExceeded is returned when context length is exceeded
	ErrContextLengthExceeded = "context length exceeded"

	// ErrRateLimited is returned when rate limited
	ErrRateLimited = "rate limited, please wait"

	// ErrNetworkError is returned for network errors
	ErrNetworkError = "network error occurred"

	// ErrToolExecutionFailed is returned when tool execution fails
	ErrToolExecutionFailed = "tool execution failed"
)

// =============================================================================
// Status Messages
// =============================================================================

// Status messages for various operations
const (
	// StatusThinking indicates the agent is thinking
	StatusThinking = "Thinking..."

	// StatusReading indicates the agent is reading a file
	StatusReading = "Reading file..."

	// StatusWriting indicates the agent is writing a file
	StatusWriting = "Writing file..."

	// StatusEditing indicates the agent is editing a file
	StatusEditing = "Editing file..."

	// StatusSearching indicates the agent is searching
	StatusSearching = "Searching..."

	// StatusRunningCommand indicates a command is being executed
	StatusRunningCommand = "Running command..."

	// StatusFetching indicates content is being fetched
	StatusFetching = "Fetching content..."

	// StatusProcessing indicates processing is in progress
	StatusProcessing = "Processing..."

	// StatusComplete indicates an operation is complete
	StatusComplete = "Complete"

	// StatusWaiting indicates waiting for input
	StatusWaiting = "Waiting for input..."
)

// =============================================================================
// Help Messages
// =============================================================================

// Help and usage messages
const (
	// HelpIntro is the introduction for help
	HelpIntro = "Claude Code - AI-powered coding assistant"

	// HelpUsagePrefix is the prefix for usage instructions
	HelpUsagePrefix = "Usage:"

	// HelpOptionsPrefix is the prefix for options
	HelpOptionsPrefix = "Options:"

	// HelpExamplesPrefix is the prefix for examples
	HelpExamplesPrefix = "Examples:"

	// HelpExit indicates how to exit
	HelpExit = "Press Ctrl+C or type 'exit' to quit"

	// HelpClear indicates how to clear the screen
	HelpClear = "Type 'clear' to clear the screen"

	// HelpHelp indicates how to get help
	HelpHelp = "Type 'help' or '--help' for this message"
)

// =============================================================================
// Git Messages
// =============================================================================

// Git-related messages
const (
	// GitNotARepo indicates not a git repository
	GitNotARepo = "Not a git repository"

	// GitNoChanges indicates no changes to commit
	GitNoChanges = "No changes to commit"

	// GitCommitCreated indicates a commit was created
	GitCommitCreated = "Commit created successfully"

	// GitCommitFailed indicates commit creation failed
	GitCommitFailed = "Failed to create commit"

	// GitPushSuccess indicates push was successful
	GitPushSuccess = "Pushed successfully"

	// GitPushFailed indicates push failed
	GitPushFailed = "Push failed"

	// GitPullSuccess indicates pull was successful
	GitPullSuccess = "Pulled successfully"

	// GitPullFailed indicates pull failed
	GitPullFailed = "Pull failed"

	// GitBranchCreated indicates a branch was created
	GitBranchCreated = "Branch created"

	// GitBranchSwitched indicates branch was switched
	GitBranchSwitched = "Switched to branch"

	// GitMergeConflict indicates merge conflict
	GitMergeConflict = "Merge conflict detected"
)

// =============================================================================
// Todo Messages
// =============================================================================

// Todo-related messages
const (
	// TodoEmptyList indicates the todo list is empty
	TodoEmptyList = "No tasks in the list"

	// TodoTaskAdded indicates a task was added
	TodoTaskAdded = "Task added"

	// TodoTaskCompleted indicates a task was completed
	TodoTaskCompleted = "Task completed"

	// TodoTaskCancelled indicates a task was cancelled
	TodoTaskCancelled = "Task cancelled"

	// TodoTaskInProgress indicates a task is in progress
	TodoTaskInProgress = "Task in progress"

	// TodoAllComplete indicates all tasks are complete
	TodoAllComplete = "All tasks completed"
)

// =============================================================================
// Agent Messages
// =============================================================================

// Agent-related messages
const (
	// AgentStarting indicates an agent is starting
	AgentStarting = "Starting agent..."

	// AgentRunning indicates an agent is running
	AgentRunning = "Agent is running"

	// AgentComplete indicates an agent completed
	AgentComplete = "Agent completed"

	// AgentFailed indicates an agent failed
	AgentFailed = "Agent failed"

	// AgentStopped indicates an agent was stopped
	AgentStopped = "Agent stopped"

	// AgentTimeout indicates an agent timed out
	AgentTimeout = "Agent timed out"

	// AgentNotFound indicates an agent was not found
	AgentNotFound = "Agent not found"
)

// =============================================================================
// Confirmation Messages
// =============================================================================

// Confirmation prompts
const (
	// ConfirmOverwrite asks to confirm file overwrite
	ConfirmOverwrite = "File already exists. Overwrite?"

	// ConfirmDelete asks to confirm deletion
	ConfirmDelete = "Are you sure you want to delete?"

	// ConfirmContinue asks to confirm continuation
	ConfirmContinue = "Do you want to continue?"

	// ConfirmExit asks to confirm exit
	ConfirmExit = "Are you sure you want to exit?"

	// YesOption is the yes option
	YesOption = "Yes"

	// NoOption is the no option
	NoOption = "No"

	// CancelOption is the cancel option
	CancelOption = "Cancel"
)

// =============================================================================
// Output Style Messages
// =============================================================================

// Output style related constants
const (
	// OutputStyleDefault is the default output style
	OutputStyleDefault = "default"

	// OutputStyleJSON is the JSON output style
	OutputStyleJSON = "json"

	// OutputStyleQuiet is the quiet output style
	OutputStyleQuiet = "quiet"

	// OutputStyleVerbose is the verbose output style
	OutputStyleVerbose = "verbose"
)

// =============================================================================
// Tool Reference Messages
// =============================================================================

// Tool reference related constants
const (
	// ToolReferenceTurnBoundary indicates a tool was loaded
	ToolReferenceTurnBoundary = "Tool loaded."

	// ToolNotFound indicates a tool was not found
	ToolNotFound = "Tool not found"

	// ToolLoadError indicates a tool failed to load
	ToolLoadError = "Failed to load tool"

	// ToolDisabled indicates a tool is disabled
	ToolDisabled = "Tool is disabled"
)

// =============================================================================
// Bash Tool Specific Messages
// =============================================================================

// Bash tool specific messages
const (
	// BashCommandPrompt is the prompt for bash commands
	BashCommandPrompt = "Enter command:"

	// BashTimeoutWarning warns about timeout
	BashTimeoutWarning = "Command may time out if it takes too long"

	// BashBackgroundInfo informs about background execution
	BashBackgroundInfo = "Running in background. You will be notified when complete."

	// BashSandboxInfo informs about sandbox mode
	BashSandboxInfo = "Running in sandbox mode with restricted access"

	// BashDestructiveWarning warns about destructive commands
	BashDestructiveWarning = "This command may be destructive. Continue?"
)
