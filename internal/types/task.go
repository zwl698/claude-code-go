package types

import (
	"fmt"
	"sync"
	"time"
)

// TaskType represents the different types of tasks that can be executed.
type TaskType string

const (
	TaskTypeLocalBash         TaskType = "local_bash"
	TaskTypeLocalAgent        TaskType = "local_agent"
	TaskTypeRemoteAgent       TaskType = "remote_agent"
	TaskTypeInProcessTeammate TaskType = "in_process_teammate"
	TaskTypeLocalWorkflow     TaskType = "local_workflow"
	TaskTypeMonitorMcp        TaskType = "monitor_mcp"
	TaskTypeDream             TaskType = "dream"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusKilled    TaskStatus = "killed"
)

// IsTerminalTaskStatus returns true when a task is in a terminal state
// and will not transition further.
func IsTerminalTaskStatus(status TaskStatus) bool {
	return status == TaskStatusCompleted || status == TaskStatusFailed || status == TaskStatusKilled
}

// TaskStateBase contains the base fields shared by all task states.
type TaskStateBase struct {
	Id            string     `json:"id"`
	Type          TaskType   `json:"type"`
	Status        TaskStatus `json:"status"`
	Description   string     `json:"description"`
	ToolUseId     string     `json:"toolUseId,omitempty"`
	StartTime     time.Time  `json:"startTime"`
	EndTime       *time.Time `json:"endTime,omitempty"`
	TotalPausedMs int64      `json:"totalPausedMs,omitempty"`
	OutputFile    string     `json:"outputFile"`
	OutputOffset  int64      `json:"outputOffset"`
	Notified      bool       `json:"notified"`
}

// TaskHandle represents a handle to a running task.
type TaskHandle struct {
	TaskId  string
	Cleanup func()
}

// LocalShellSpawnInput represents the input for spawning a local shell command.
type LocalShellSpawnInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Timeout     int    `json:"timeout,omitempty"`
	ToolUseId   string `json:"toolUseId,omitempty"`
	AgentId     string `json:"agentId,omitempty"`
	// UI display variant: description-as-label, dialog title, status bar pill.
	Kind string `json:"kind,omitempty"` // 'bash' | 'monitor'
}

// Task interface defines the common methods for all task types.
type Task interface {
	Name() string
	Type() TaskType
	Kill(taskId string, setAppState SetAppStateFunc) error
}

// SetAppStateFunc is a function type for updating application state.
// The function takes a updater function that receives the current state and returns a new state.
type SetAppStateFunc func(updater interface{})

// TaskContext provides context for task execution.
type TaskContext struct {
	AbortController AbortController
	GetAppState     func() interface{}
	SetAppState     SetAppStateFunc
}

// AbortController provides cancellation functionality.
type AbortController struct {
	cancel  func()
	aborted bool
	mu      sync.RWMutex
}

// NewAbortController creates a new AbortController.
func NewAbortController() *AbortController {
	return &AbortController{}
}

// Abort cancels the operation.
func (a *AbortController) Abort() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.aborted = true
	if a.cancel != nil {
		a.cancel()
	}
}

// IsAborted returns true if the operation has been aborted.
func (a *AbortController) IsAborted() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aborted
}

// SetCancel sets the cancel function for the controller.
func (a *AbortController) SetCancel(cancel func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cancel = cancel
}

// CreateTaskStateBase creates a new TaskStateBase with default values.
func CreateTaskStateBase(id string, taskType TaskType, description string, toolUseId string) TaskStateBase {
	return TaskStateBase{
		Id:          id,
		Type:        taskType,
		Status:      TaskStatusPending,
		Description: description,
		ToolUseId:   toolUseId,
		StartTime:   time.Now(),
		OutputFile:  GetTaskOutputPath(id),
		Notified:    false,
	}
}

// GetTaskOutputPath returns the path for task output files.
func GetTaskOutputPath(taskId string) string {
	// In production, this would use a proper temp directory
	return fmt.Sprintf("/tmp/claude-code-task-%s.log", taskId)
}
