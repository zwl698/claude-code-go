// Package types contains core type definitions for the claude-code CLI.
// This file contains log-related types translated from TypeScript.
package types

import (
	"sort"
	"time"
)

// LogOption represents a session log entry for the resume picker.
// Matches TypeScript LogOption type from logs.ts.
type LogOption struct {
	Date            string                    `json:"date"`
	Messages        []SerializedMessage       `json:"messages,omitempty"`
	FullPath        string                    `json:"fullPath,omitempty"`
	Value           int64                     `json:"value"`
	Created         time.Time                 `json:"created"`
	Modified        time.Time                 `json:"modified"`
	FirstPrompt     string                    `json:"firstPrompt"`
	MessageCount    int                       `json:"messageCount"`
	FileSize        int64                     `json:"fileSize,omitempty"`
	IsSidechain     bool                      `json:"isSidechain"`
	IsLite          bool                      `json:"isLite,omitempty"`
	SessionId       string                    `json:"sessionId,omitempty"`
	TeamName        string                    `json:"teamName,omitempty"`
	AgentName       string                    `json:"agentName,omitempty"`
	AgentColor      string                    `json:"agentColor,omitempty"`
	AgentSetting    string                    `json:"agentSetting,omitempty"`
	IsTeammate      bool                      `json:"isTeammate,omitempty"`
	LeafUuid        string                    `json:"leafUuid,omitempty"`
	Summary         string                    `json:"summary,omitempty"`
	CustomTitle     string                    `json:"customTitle,omitempty"`
	Tag             string                    `json:"tag,omitempty"`
	GitBranch       string                    `json:"gitBranch,omitempty"`
	ProjectPath     string                    `json:"projectPath,omitempty"`
	PrNumber        int                       `json:"prNumber,omitempty"`
	PrUrl           string                    `json:"prUrl,omitempty"`
	PrRepository    string                    `json:"prRepository,omitempty"`
	Mode            string                    `json:"mode,omitempty"` // coordinator | normal
	WorktreeSession *PersistedWorktreeSession `json:"worktreeSession,omitempty"`
}

// SerializedMessage represents a message serialized to the transcript.
type SerializedMessage struct {
	Cwd        string      `json:"cwd"`
	UserType   string      `json:"userType"`
	Entrypoint string      `json:"entrypoint,omitempty"`
	SessionId  string      `json:"sessionId"`
	Timestamp  string      `json:"timestamp"`
	Version    string      `json:"version"`
	GitBranch  string      `json:"gitBranch,omitempty"`
	Slug       string      `json:"slug,omitempty"`
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
}

// SummaryMessage represents a summary entry in the transcript.
type SummaryMessage struct {
	Type     string `json:"type"`
	LeafUuid string `json:"leafUuid"`
	Summary  string `json:"summary"`
}

// CustomTitleMessage represents a user-set custom title.
type CustomTitleMessage struct {
	Type        string `json:"type"`
	SessionId   string `json:"sessionId"`
	CustomTitle string `json:"customTitle"`
}

// AiTitleMessage represents an AI-generated session title.
type AiTitleMessage struct {
	Type      string `json:"type"`
	SessionId string `json:"sessionId"`
	AiTitle   string `json:"aiTitle"`
}

// LastPromptMessage stores the last user prompt.
type LastPromptMessage struct {
	Type       string `json:"type"`
	SessionId  string `json:"sessionId"`
	LastPrompt string `json:"lastPrompt"`
}

// TaskSummaryMessage is a periodic summary written by forked threads.
type TaskSummaryMessage struct {
	Type      string `json:"type"`
	SessionId string `json:"sessionId"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
}

// TagMessage represents a tag added to a session.
type TagMessage struct {
	Type      string `json:"type"`
	SessionId string `json:"sessionId"`
	Tag       string `json:"tag"`
}

// AgentNameMessage stores an agent's custom name.
type AgentNameMessage struct {
	Type      string `json:"type"`
	SessionId string `json:"sessionId"`
	AgentName string `json:"agentName"`
}

// AgentColorMessage stores an agent's display color.
type AgentColorMessage struct {
	Type       string `json:"type"`
	SessionId  string `json:"sessionId"`
	AgentColor string `json:"agentColor"`
}

// AgentSettingMessage stores the agent definition used.
type AgentSettingMessage struct {
	Type         string `json:"type"`
	SessionId    string `json:"sessionId"`
	AgentSetting string `json:"agentSetting"`
}

// PRLinkMessage links a session to a GitHub PR.
type PRLinkMessage struct {
	Type         string `json:"type"`
	SessionId    string `json:"sessionId"`
	PrNumber     int    `json:"prNumber"`
	PrUrl        string `json:"prUrl"`
	PrRepository string `json:"prRepository"`
	Timestamp    string `json:"timestamp"`
}

// ModeEntry records the session mode.
type ModeEntry struct {
	Type      string `json:"type"`
	SessionId string `json:"sessionId"`
	Mode      string `json:"mode"` // coordinator | normal
}

// PersistedWorktreeSession represents worktree state persisted for resume.
type PersistedWorktreeSession struct {
	OriginalCwd        string `json:"originalCwd"`
	WorktreePath       string `json:"worktreePath"`
	WorktreeName       string `json:"worktreeName"`
	WorktreeBranch     string `json:"worktreeBranch,omitempty"`
	OriginalBranch     string `json:"originalBranch,omitempty"`
	OriginalHeadCommit string `json:"originalHeadCommit,omitempty"`
	SessionId          string `json:"sessionId"`
	TmuxSessionName    string `json:"tmuxSessionName,omitempty"`
	HookBased          bool   `json:"hookBased,omitempty"`
}

// WorktreeStateEntry records whether the session is in a worktree.
type WorktreeStateEntry struct {
	Type            string                    `json:"type"`
	SessionId       string                    `json:"sessionId"`
	WorktreeSession *PersistedWorktreeSession `json:"worktreeSession"`
}

// FileHistorySnapshotMessage stores a snapshot of file history.
type FileHistorySnapshotMessage struct {
	Type             string `json:"type"`
	MessageId        string `json:"messageId"`
	Snapshot         string `json:"snapshot"`
	IsSnapshotUpdate bool   `json:"isSnapshotUpdate"`
}

// FileAttributionState tracks Claude's contributions to a file.
type FileAttributionState struct {
	ContentHash        string `json:"contentHash"`
	ClaudeContribution int    `json:"claudeContribution"`
	Mtime              int64  `json:"mtime"`
}

// AttributionSnapshotMessage tracks character-level contributions.
type AttributionSnapshotMessage struct {
	Type                              string                          `json:"type"`
	MessageId                         string                          `json:"messageId"`
	Surface                           string                          `json:"surface"`
	FileStates                        map[string]FileAttributionState `json:"fileStates"`
	PromptCount                       int                             `json:"promptCount,omitempty"`
	PromptCountAtLastCommit           int                             `json:"promptCountAtLastCommit,omitempty"`
	PermissionPromptCount             int                             `json:"permissionPromptCount,omitempty"`
	PermissionPromptCountAtLastCommit int                             `json:"permissionPromptCountAtLastCommit,omitempty"`
	EscapeCount                       int                             `json:"escapeCount,omitempty"`
	EscapeCountAtLastCommit           int                             `json:"escapeCountAtLastCommit,omitempty"`
}

// TranscriptMessage represents a message in the transcript.
type TranscriptMessage struct {
	SerializedMessage
	ParentUuid        string `json:"parentUuid"`
	LogicalParentUuid string `json:"logicalParentUuid,omitempty"`
	IsSidechain       bool   `json:"isSidechain"`
	GitBranch         string `json:"gitBranch,omitempty"`
	AgentId           string `json:"agentId,omitempty"`
	TeamName          string `json:"teamName,omitempty"`
	AgentName         string `json:"agentName,omitempty"`
	AgentColor        string `json:"agentColor,omitempty"`
	PromptId          string `json:"promptId,omitempty"`
}

// SpeculationAcceptMessage records when speculation is accepted.
type SpeculationAcceptMessage struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	TimeSavedMs int64  `json:"timeSavedMs"`
}

// ContextCollapseCommitEntry records archived messages from context collapse.
type ContextCollapseCommitEntry struct {
	Type              string `json:"type"`
	SessionId         string `json:"sessionId"`
	CollapseId        string `json:"collapseId"`
	SummaryUuid       string `json:"summaryUuid"`
	SummaryContent    string `json:"summaryContent"`
	Summary           string `json:"summary"`
	FirstArchivedUuid string `json:"firstArchivedUuid"`
	LastArchivedUuid  string `json:"lastArchivedUuid"`
}

// ContextCollapseSnapshotEntry is a snapshot of staged queue state.
type ContextCollapseSnapshotEntry struct {
	Type            string       `json:"type"`
	SessionId       string       `json:"sessionId"`
	Staged          []StagedSpan `json:"staged"`
	Armed           bool         `json:"armed"`
	LastSpawnTokens int64        `json:"lastSpawnTokens"`
}

// StagedSpan represents a staged message span.
type StagedSpan struct {
	StartUuid string `json:"startUuid"`
	EndUuid   string `json:"endUuid"`
	Summary   string `json:"summary"`
	Risk      int    `json:"risk"`
	StagedAt  int64  `json:"stagedAt"`
}

// Entry is the union type for all transcript entry types.
type Entry struct {
	// Discriminator for the entry type
	Type string `json:"type"`

	// All possible fields from different entry types
	TranscriptMessage       *TranscriptMessage            `json:"-"`
	SummaryMessage          *SummaryMessage               `json:"-"`
	CustomTitleMessage      *CustomTitleMessage           `json:"-"`
	AiTitleMessage          *AiTitleMessage               `json:"-"`
	LastPromptMessage       *LastPromptMessage            `json:"-"`
	TaskSummaryMessage      *TaskSummaryMessage           `json:"-"`
	TagMessage              *TagMessage                   `json:"-"`
	AgentNameMessage        *AgentNameMessage             `json:"-"`
	AgentColorMessage       *AgentColorMessage            `json:"-"`
	AgentSettingMessage     *AgentSettingMessage          `json:"-"`
	PRLinkMessage           *PRLinkMessage                `json:"-"`
	FileHistorySnapshot     *FileHistorySnapshotMessage   `json:"-"`
	AttributionSnapshot     *AttributionSnapshotMessage   `json:"-"`
	SpeculationAccept       *SpeculationAcceptMessage     `json:"-"`
	ModeEntry               *ModeEntry                    `json:"-"`
	WorktreeStateEntry      *WorktreeStateEntry           `json:"-"`
	ContentReplacementEntry *ContentReplacementEntry      `json:"-"`
	ContextCollapseCommit   *ContextCollapseCommitEntry   `json:"-"`
	ContextCollapseSnapshot *ContextCollapseSnapshotEntry `json:"-"`
}

// ContentReplacementEntry records replaced content blocks.
type ContentReplacementEntry struct {
	Type         string                     `json:"type"`
	SessionId    string                     `json:"sessionId"`
	AgentId      string                     `json:"agentId,omitempty"`
	Replacements []ContentReplacementRecord `json:"replacements"`
}

// ContentReplacementRecord represents a single content replacement.
type ContentReplacementRecord struct {
	OriginalType    string `json:"originalType"`
	ReplacementType string `json:"replacementType"`
	MessageId       string `json:"messageId"`
	BlockIndex      int    `json:"blockIndex"`
	StorageKey      string `json:"storageKey,omitempty"`
}

// SortLogs sorts logs by modified date (newest first), then by created date.
// Matches TypeScript sortLogs function.
func SortLogs(logs []LogOption) []LogOption {
	sorted := make([]LogOption, len(logs))
	copy(sorted, logs)

	sort.Slice(sorted, func(i, j int) bool {
		// Sort by modified date (newest first)
		if !sorted[i].Modified.Equal(sorted[j].Modified) {
			return sorted[i].Modified.After(sorted[j].Modified)
		}
		// If modified dates are equal, sort by created date (newest first)
		return sorted[i].Created.After(sorted[j].Created)
	})

	return sorted
}
