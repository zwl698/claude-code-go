package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"claude-code-go/internal/types"
)

// =============================================================================
// Task Executor
// =============================================================================

// Executor manages task execution lifecycle.
type Executor struct {
	registry    *Registry
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex
}

// NewExecutor creates a new task executor.
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry:    registry,
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// ExecuteLocalAgent starts a local agent task.
func (e *Executor) ExecuteLocalAgent(ctx context.Context, task *LocalAgentTaskState, handler AgentHandler) error {
	e.mu.Lock()
	execCtx, cancel := context.WithCancel(ctx)
	e.cancelFuncs[task.ID] = cancel
	e.mu.Unlock()

	task.Status = TaskStatusRunning
	e.registry.Register(task)

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.cancelFuncs, task.ID)
			e.mu.Unlock()
		}()

		result, err := handler(execCtx, task)
		if err != nil {
			task.Error = err.Error()
			task.Status = TaskStatusFailed
		} else {
			task.Result = result
			task.Status = TaskStatusCompleted
		}
		now := time.Now()
		task.EndTime = &now
		e.registry.Register(task)
	}()

	return nil
}

// ExecuteLocalShell starts a local shell task.
func (e *Executor) ExecuteLocalShell(ctx context.Context, task *LocalShellTaskState, handler ShellHandler) error {
	e.mu.Lock()
	execCtx, cancel := context.WithCancel(ctx)
	e.cancelFuncs[task.ID] = cancel
	e.mu.Unlock()

	task.Status = TaskStatusRunning
	e.registry.Register(task)

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.cancelFuncs, task.ID)
			e.mu.Unlock()
		}()

		exitCode, err := handler(execCtx, task)
		if err != nil {
			task.Error = err.Error()
			task.Status = TaskStatusFailed
		} else {
			task.ExitCode = exitCode
			task.Status = TaskStatusCompleted
		}
		now := time.Now()
		task.EndTime = &now
		e.registry.Register(task)
	}()

	return nil
}

// ExecuteRemoteAgent starts a remote agent task.
func (e *Executor) ExecuteRemoteAgent(ctx context.Context, task *RemoteAgentTaskState, handler RemoteAgentHandler) error {
	e.mu.Lock()
	execCtx, cancel := context.WithCancel(ctx)
	e.cancelFuncs[task.ID] = cancel
	e.mu.Unlock()

	task.Status = TaskStatusRunning
	task.IsBackgrounded = true
	e.registry.Register(task)

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.cancelFuncs, task.ID)
			e.mu.Unlock()
		}()

		result, err := handler(execCtx, task)
		if err != nil {
			task.Error = err.Error()
			task.Status = TaskStatusFailed
		} else {
			task.Status = TaskStatusCompleted
			_ = result
		}
		now := time.Now()
		task.EndTime = &now
		e.registry.Register(task)
	}()

	return nil
}

// KillTask kills a running task.
func (e *Executor) KillTask(taskID string) error {
	e.mu.RLock()
	cancel, exists := e.cancelFuncs[taskID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found or not running", taskID)
	}

	cancel()

	task := e.registry.Get(taskID)
	if task != nil {
		KillTask(task)
		e.registry.Register(task)
	}

	return nil
}

// AgentHandler is a function that handles agent execution.
type AgentHandler func(ctx context.Context, task *LocalAgentTaskState) (interface{}, error)

// ShellHandler is a function that handles shell execution.
type ShellHandler func(ctx context.Context, task *LocalShellTaskState) (*int, error)

// RemoteAgentHandler is a function that handles remote agent execution.
type RemoteAgentHandler func(ctx context.Context, task *RemoteAgentTaskState) (interface{}, error)

// =============================================================================
// Task Manager
// =============================================================================

// Manager provides high-level task management.
type Manager struct {
	registry *Registry
	executor *Executor
	store    *types.AppStateStore
	taskChan chan TaskNotification
	mu       sync.RWMutex
}

// NewManager creates a new task manager.
func NewManager(store *types.AppStateStore) *Manager {
	registry := NewRegistry()
	return &Manager{
		registry: registry,
		executor: NewExecutor(registry),
		store:    store,
		taskChan: make(chan TaskNotification, 100),
	}
}

// SpawnLocalAgent spawns a new local agent task.
func (m *Manager) SpawnLocalAgent(ctx context.Context, prompt, agentType, description string, background bool) (*LocalAgentTaskState, error) {
	id := generateTaskID()
	task := CreateLocalAgentTask(id, prompt, agentType, description)
	task.IsBackgrounded = background

	if err := m.registry.Register(task); err != nil {
		return nil, err
	}

	m.store.Update(func(state *types.AppState) *types.AppState {
		state.Tasks[id] = types.TaskStateBase{
			Id:          id,
			Type:        types.TaskTypeLocalAgent,
			Status:      types.TaskStatusPending,
			Description: description,
			StartTime:   task.StartTime,
		}
		return state
	})

	m.notifyTaskCreated(task)
	return task, nil
}

// SpawnLocalShell spawns a new local shell task.
func (m *Manager) SpawnLocalShell(ctx context.Context, command, directory, description string, background bool) (*LocalShellTaskState, error) {
	id := generateTaskID()
	task := CreateLocalShellTask(id, command, directory, description)
	task.IsBackgrounded = background

	if err := m.registry.Register(task); err != nil {
		return nil, err
	}

	m.store.Update(func(state *types.AppState) *types.AppState {
		state.Tasks[id] = types.TaskStateBase{
			Id:          id,
			Type:        types.TaskTypeLocalBash,
			Status:      types.TaskStatusPending,
			Description: description,
			StartTime:   task.StartTime,
		}
		return state
	})

	m.notifyTaskCreated(task)
	return task, nil
}

// SpawnRemoteAgent spawns a new remote agent task.
func (m *Manager) SpawnRemoteAgent(ctx context.Context, command, sessionID, description string) (*RemoteAgentTaskState, error) {
	id := generateTaskID()
	task := CreateRemoteAgentTask(id, command, sessionID, description)

	if err := m.registry.Register(task); err != nil {
		return nil, err
	}

	m.store.Update(func(state *types.AppState) *types.AppState {
		state.Tasks[id] = types.TaskStateBase{
			Id:          id,
			Type:        types.TaskTypeRemoteAgent,
			Status:      types.TaskStatusPending,
			Description: description,
			StartTime:   task.StartTime,
		}
		return state
	})

	m.notifyTaskCreated(task)
	return task, nil
}

// GetTask retrieves a task by ID.
func (m *Manager) GetTask(id string) TaskState {
	return m.registry.Get(id)
}

// GetAllTasks returns all tasks.
func (m *Manager) GetAllTasks() map[string]TaskState {
	return m.registry.GetAll()
}

// GetBackgroundTasks returns all background tasks.
func (m *Manager) GetBackgroundTasks() []TaskState {
	tasks := m.registry.GetAll()
	result := make([]TaskState, 0)
	for _, task := range tasks {
		if IsBackgroundTask(task) {
			result = append(result, task)
		}
	}
	return result
}

// KillTask kills a task by ID.
func (m *Manager) KillTask(id string) error {
	task := m.registry.Get(id)
	if task == nil {
		return fmt.Errorf("task %s not found", id)
	}

	if err := m.executor.KillTask(id); err != nil {
		return err
	}

	m.store.Update(func(state *types.AppState) *types.AppState {
		if t, ok := state.Tasks[id]; ok {
			t.Status = types.TaskStatusKilled
			now := time.Now()
			t.EndTime = &now
			state.Tasks[id] = t
		}
		return state
	})

	m.notifyTaskKilled(task)
	return nil
}

// RetrieveTask retrieves a task result.
func (m *Manager) RetrieveTask(id string) (interface{}, error) {
	task := m.registry.Get(id)
	if task == nil {
		return nil, fmt.Errorf("task %s not found", id)
	}

	agentTask, ok := task.(*LocalAgentTaskState)
	if !ok {
		return nil, fmt.Errorf("task %s is not a local agent task", id)
	}

	if agentTask.Status != TaskStatusCompleted && agentTask.Status != TaskStatusFailed {
		return nil, fmt.Errorf("task %s is still running", id)
	}

	agentTask.Retrieved = true
	m.notifyTaskRetrieved(agentTask)

	return agentTask.Result, nil
}

// Notifications returns the notification channel.
func (m *Manager) Notifications() <-chan TaskNotification {
	return m.taskChan
}

// StartExecution starts executing a local agent task.
func (m *Manager) StartExecution(ctx context.Context, taskID string, handler AgentHandler) error {
	task := m.registry.Get(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	agentTask, ok := task.(*LocalAgentTaskState)
	if !ok {
		return fmt.Errorf("task %s is not a local agent task", taskID)
	}

	return m.executor.ExecuteLocalAgent(ctx, agentTask, handler)
}

// StartShellExecution starts executing a shell task.
func (m *Manager) StartShellExecution(ctx context.Context, taskID string, handler ShellHandler) error {
	task := m.registry.Get(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	shellTask, ok := task.(*LocalShellTaskState)
	if !ok {
		return fmt.Errorf("task %s is not a shell task", taskID)
	}

	return m.executor.ExecuteLocalShell(ctx, shellTask, handler)
}

// StartRemoteExecution starts executing a remote agent task.
func (m *Manager) StartRemoteExecution(ctx context.Context, taskID string, handler RemoteAgentHandler) error {
	task := m.registry.Get(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	remoteTask, ok := task.(*RemoteAgentTaskState)
	if !ok {
		return fmt.Errorf("task %s is not a remote agent task", taskID)
	}

	return m.executor.ExecuteRemoteAgent(ctx, remoteTask, handler)
}

// Cleanup removes completed/failed tasks older than the given duration.
func (m *Manager) Cleanup(olderThan time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks := m.registry.GetAll()
	cleaned := 0
	now := time.Now()

	for id, task := range tasks {
		base := task.GetBase()
		if base.Status == TaskStatusCompleted || base.Status == TaskStatusFailed || base.Status == TaskStatusKilled {
			if base.EndTime != nil && now.Sub(*base.EndTime) > olderThan {
				m.registry.Unregister(id)
				cleaned++
			}
		}
	}

	return cleaned
}

// notifyTaskCreated sends a task creation notification.
func (m *Manager) notifyTaskCreated(task TaskState) {
	notification := CreateNotification(task, "")
	select {
	case m.taskChan <- notification:
	default:
	}
}

// notifyTaskKilled sends a task killed notification.
func (m *Manager) notifyTaskKilled(task TaskState) {
	notification := CreateNotification(task, "Task killed")
	select {
	case m.taskChan <- notification:
	default:
	}
}

// notifyTaskRetrieved sends a task retrieved notification.
func (m *Manager) notifyTaskRetrieved(task *LocalAgentTaskState) {
	notification := TaskNotification{
		TaskID:      task.ID,
		Description: task.Description,
		Status:      string(task.Status),
		Timestamp:   time.Now(),
	}
	select {
	case m.taskChan <- notification:
	default:
	}
}

// =============================================================================
// Progress Tracking
// =============================================================================

// UpdateProgress updates the progress of a local agent task.
func (m *Manager) UpdateProgress(taskID string, progress AgentProgress) error {
	task := m.registry.Get(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	agentTask, ok := task.(*LocalAgentTaskState)
	if !ok {
		return fmt.Errorf("task %s is not a local agent task", taskID)
	}

	agentTask.Progress = &progress
	return nil
}

// AddToolActivity adds a tool activity to progress tracking.
func (m *Manager) AddToolActivity(taskID string, activity ToolActivity) error {
	task := m.registry.Get(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	agentTask, ok := task.(*LocalAgentTaskState)
	if !ok {
		return fmt.Errorf("task %s is not a local agent task", taskID)
	}

	if agentTask.Progress == nil {
		agentTask.Progress = &AgentProgress{}
	}

	agentTask.Progress.ToolUseCount++
	agentTask.Progress.RecentActivities = append(
		agentTask.Progress.RecentActivities,
		activity,
	)

	if len(agentTask.Progress.RecentActivities) > 10 {
		agentTask.Progress.RecentActivities = agentTask.Progress.RecentActivities[len(agentTask.Progress.RecentActivities)-10:]
	}

	return nil
}

// =============================================================================
// Task ID Generation
// =============================================================================

var taskIDCounter uint64
var taskIDMu sync.Mutex

func generateTaskID() string {
	taskIDMu.Lock()
	defer taskIDMu.Unlock()
	taskIDCounter++
	return fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), taskIDCounter)
}

// =============================================================================
// JSON Serialization Helpers
// =============================================================================

// SerializeTask serializes a task to JSON.
func SerializeTask(task TaskState) ([]byte, error) {
	return json.Marshal(task)
}

// DeserializeTask deserializes a task from JSON.
func DeserializeTask(data []byte) (TaskState, error) {
	var base TaskStateBase
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case TaskTypeLocalAgent:
		var task LocalAgentTaskState
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		return &task, nil
	case TaskTypeLocalShell:
		var task LocalShellTaskState
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		return &task, nil
	case TaskTypeRemoteAgent:
		var task RemoteAgentTaskState
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		return &task, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", base.Type)
	}
}
