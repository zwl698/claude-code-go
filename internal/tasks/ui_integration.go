package tasks

import (
	"fmt"

	"claude-code-go/internal/types"
)

// =============================================================================
// UI Integration
// =============================================================================

// TaskPanelItem represents an item to display in the task panel.
type TaskPanelItem struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Type        TaskType   `json:"type"`
	Status      TaskStatus `json:"status"`
	StartTime   string     `json:"startTime"`
	EndTime     *string    `json:"endTime,omitempty"`
	Error       string     `json:"error,omitempty"`
	Progress    string     `json:"progress,omitempty"`
}

// GetPanelItems returns tasks formatted for the UI panel.
func GetPanelItems(tasks map[string]TaskState) []TaskPanelItem {
	items := make([]TaskPanelItem, 0)
	for _, task := range tasks {
		base := task.GetBase()
		if !IsPanelTask(task) {
			continue
		}

		item := TaskPanelItem{
			ID:          base.ID,
			Description: base.Description,
			Type:        base.Type,
			Status:      base.Status,
			StartTime:   base.StartTime.Format("15:04:05"),
		}

		if base.EndTime != nil {
			t := base.EndTime.Format("15:04:05")
			item.EndTime = &t
		}

		// Add progress info for agent tasks
		if agentTask, ok := task.(*LocalAgentTaskState); ok {
			if agentTask.Error != "" {
				item.Error = agentTask.Error
			}
			if agentTask.Progress != nil {
				item.Progress = fmt.Sprintf("%d tools, %d tokens",
					agentTask.Progress.ToolUseCount,
					agentTask.Progress.TokenCount)
			}
		}

		if shellTask, ok := task.(*LocalShellTaskState); ok {
			if shellTask.Error != "" {
				item.Error = shellTask.Error
			}
		}

		if remoteTask, ok := task.(*RemoteAgentTaskState); ok {
			if remoteTask.Error != "" {
				item.Error = remoteTask.Error
			}
		}

		items = append(items, item)
	}

	return items
}

// IsPanelTask returns true if a task should be shown in the panel.
func IsPanelTask(task TaskState) bool {
	base := task.GetBase()
	// Don't show completed/failed tasks that have been retrieved
	if base.Status == TaskStatusCompleted || base.Status == TaskStatusFailed || base.Status == TaskStatusKilled {
		if agentTask, ok := task.(*LocalAgentTaskState); ok {
			if agentTask.Retrieved {
				return false
			}
		}
	}
	return true
}

// GetBackgroundTaskCount returns the count of running/pending background tasks.
func GetBackgroundTaskCount(tasks map[string]TaskState) int {
	count := 0
	for _, task := range tasks {
		if IsBackgroundTask(task) {
			count++
		}
	}
	return count
}

// FormatTaskStatus formats a task status for display.
func FormatTaskStatus(status TaskStatus) string {
	switch status {
	case TaskStatusPending:
		return "⏳ Pending"
	case TaskStatusRunning:
		return "🔄 Running"
	case TaskStatusCompleted:
		return "✅ Completed"
	case TaskStatusFailed:
		return "❌ Failed"
	case TaskStatusKilled:
		return "💀 Killed"
	default:
		return string(status)
	}
}

// FormatTaskType formats a task type for display.
func FormatTaskType(taskType TaskType) string {
	switch taskType {
	case TaskTypeLocalAgent:
		return "🤖 Agent"
	case TaskTypeLocalShell:
		return "💻 Shell"
	case TaskTypeRemoteAgent:
		return "🌐 Remote Agent"
	case TaskTypeLocalWorkflow:
		return "📋 Workflow"
	case TaskTypeMonitorMcp:
		return "📡 MCP Monitor"
	default:
		return string(taskType)
	}
}

// =============================================================================
// AppState Integration
// =============================================================================

// SyncToAppState synchronizes task state to the app state.
func (m *Manager) SyncToAppState() {
	tasks := m.GetAllTasks()
	m.store.Update(func(state *types.AppState) *types.AppState {
		for id, task := range tasks {
			base := task.GetBase()
			existingTask := state.Tasks[id]
			existingTask.Id = id
			existingTask.Type = ConvertTaskType(base.Type)
			existingTask.Status = ConvertTaskStatus(base.Status)
			existingTask.Description = base.Description
			existingTask.StartTime = base.StartTime
			existingTask.EndTime = base.EndTime
			existingTask.Notified = base.Notified
			state.Tasks[id] = existingTask
		}
		return state
	})
}

// ConvertTaskType converts tasks.TaskType to types.TaskType.
func ConvertTaskType(t TaskType) types.TaskType {
	switch t {
	case TaskTypeLocalAgent:
		return types.TaskTypeLocalAgent
	case TaskTypeLocalShell:
		return types.TaskTypeLocalBash
	case TaskTypeRemoteAgent:
		return types.TaskTypeRemoteAgent
	case TaskTypeLocalWorkflow:
		return types.TaskTypeLocalWorkflow
	case TaskTypeMonitorMcp:
		return types.TaskTypeMonitorMcp
	default:
		return types.TaskType(t)
	}
}

// ConvertTaskStatus converts tasks.TaskStatus to types.TaskStatus.
func ConvertTaskStatus(s TaskStatus) types.TaskStatus {
	return types.TaskStatus(s)
}

// =============================================================================
// Task Suggestions
// =============================================================================

// TaskSuggestion represents a suggested task for the user.
type TaskSuggestion struct {
	Summary string `json:"summary"`
	Task    string `json:"task"`
}

// GenerateTaskSuggestions generates task suggestions based on current context.
func GenerateTaskSuggestions(tasks map[string]TaskState, currentContext string) []TaskSuggestion {
	suggestions := make([]TaskSuggestion, 0)

	// Check for failed tasks that could be retried
	for _, task := range tasks {
		base := task.GetBase()
		if base.Status == TaskStatusFailed {
			if agentTask, ok := task.(*LocalAgentTaskState); ok {
				suggestions = append(suggestions, TaskSuggestion{
					Summary: fmt.Sprintf("Retry failed task: %s", base.Description),
					Task:    fmt.Sprintf("retry:%s", agentTask.AgentID),
				})
			}
		}
	}

	// Add context-based suggestions
	if len(tasks) == 0 && currentContext != "" {
		suggestions = append(suggestions, TaskSuggestion{
			Summary: "Analyze current file",
			Task:    "analyze:current",
		})
		suggestions = append(suggestions, TaskSuggestion{
			Summary: "Run tests",
			Task:    "test:run",
		})
	}

	return suggestions
}

// =============================================================================
// Task Events
// =============================================================================

// TaskEvent represents an event that happened to a task.
type TaskEvent struct {
	Type      string      `json:"type"` // created, started, completed, failed, killed, retrieved
	TaskID    string      `json:"taskId"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// EventBus manages task events.
type EventBus struct {
	subscribers []chan TaskEvent
	mu          chan struct{}
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]chan TaskEvent, 0),
		mu:          make(chan struct{}, 1),
	}
}

// Subscribe returns a channel that receives task events.
func (b *EventBus) Subscribe() <-chan TaskEvent {
	ch := make(chan TaskEvent, 10)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Publish sends an event to all subscribers.
func (b *EventBus) Publish(event TaskEvent) {
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}
