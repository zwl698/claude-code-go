package constants

// XML tag names used to mark skill/command metadata in messages
const (
	CommandNameTag    = "command-name"
	CommandMessageTag = "command-message"
	CommandArgsTag    = "command-args"
)

// XML tag names for terminal/bash command input and output in user messages
// These wrap content that represents terminal activity, not actual user prompts
const (
	BashInputTag          = "bash-input"
	BashStdoutTag         = "bash-stdout"
	BashStderrTag         = "bash-stderr"
	LocalCommandStdoutTag = "local-command-stdout"
	LocalCommandStderrTag = "local-command-stderr"
	LocalCommandCaveatTag = "local-command-caveat"
)

// TerminalOutputTags contains all terminal-related tags that indicate a message
// is terminal output, not a user prompt
var TerminalOutputTags = []string{
	BashInputTag,
	BashStdoutTag,
	BashStderrTag,
	LocalCommandStdoutTag,
	LocalCommandStderrTag,
	LocalCommandCaveatTag,
}

const TickTag = "tick"

// XML tag names for task notifications (background task completions)
const (
	TaskNotificationTag = "task-notification"
	TaskIDTag           = "task-id"
	ToolUseIDTag        = "tool-use-id"
	TaskTypeTag         = "task-type"
	OutputFileTag       = "output-file"
	StatusTag           = "status"
	SummaryTag          = "summary"
	ReasonTag           = "reason"
	WorktreeTag         = "worktree"
	WorktreePathTag     = "worktreePath"
	WorktreeBranchTag   = "worktreeBranch"
)

// XML tag names for ultraplan mode (remote parallel planning sessions)
const UltraplanTag = "ultraplan"

// RemoteReviewTag is the XML tag name for remote /review results (teleported review session output).
// Remote session wraps its final review in this tag; local poller extracts it.
const RemoteReviewTag = "remote-review"

// RemoteReviewProgressTag - run_hunt.sh's heartbeat echoes the orchestrator's progress.json
// inside this tag every ~10s. Local poller parses the latest for the task-status line.
const RemoteReviewProgressTag = "remote-review-progress"

// TeammateMessageTag is the XML tag name for teammate messages (swarm inter-agent communication)
const TeammateMessageTag = "teammate-message"

// XML tag names for external channel messages
const (
	ChannelMessageTag = "channel-message"
	ChannelTag        = "channel"
)

// CrossSessionMessageTag is the XML tag name for cross-session UDS messages
// (another Claude session's inbox)
const CrossSessionMessageTag = "cross-session-message"

// ForkBoilerplateTag wraps the rules/format boilerplate in a fork child's first message.
// Lets the transcript renderer collapse the boilerplate and show only the directive.
const ForkBoilerplateTag = "fork-boilerplate"

// ForkDirectivePrefix is the prefix before the directive text, stripped by the renderer.
// Keep in sync across buildChildMessage (generates) and UserForkBoilerplateMessage (parses).
const ForkDirectivePrefix = "Your directive: "

// CommonHelpArgs contains common argument patterns for slash commands that request help
var CommonHelpArgs = []string{"help", "-h", "--help"}

// CommonInfoArgs contains common argument patterns for slash commands that request current state/info
var CommonInfoArgs = []string{
	"list",
	"show",
	"display",
	"current",
	"view",
	"get",
	"check",
	"describe",
	"print",
	"version",
	"about",
	"status",
	"?",
}
