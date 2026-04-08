// Package state provides state management for the claude-code CLI.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"claude-code-go/internal/utils"
)

// ========================================
// State Types
// ========================================

// AppState represents the global application state.
type AppState struct {
	mu sync.RWMutex `json:"-"`

	// Session info
	SessionID    string `json:"session_id"`
	SessionStart int64  `json:"session_start"`

	// User info
	UserType         string `json:"user_type"`
	SubscriptionType string `json:"subscription_type"`

	// Model settings
	CurrentModel string `json:"current_model"`

	// Conversation state
	ConversationID string `json:"conversation_id,omitempty"`
	MessageCount   int    `json:"message_count"`

	// Working directory
	Cwd string `json:"cwd"`

	// UI state
	IsInteractive  bool   `json:"is_interactive"`
	PermissionMode string `json:"permission_mode,omitempty"`

	// Feature flags
	Features map[string]bool `json:"features,omitempty"`

	// Custom data
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// StateManager manages application state.
type StateManager struct {
	mu    sync.RWMutex
	state *AppState
	path  string
}

// ========================================
// State Manager
// ========================================

// NewStateManager creates a new state manager.
func NewStateManager() *StateManager {
	return &StateManager{
		state: &AppState{
			SessionStart: 0, // Will be set on init
			Features:     make(map[string]bool),
			Custom:       make(map[string]interface{}),
		},
	}
}

// Initialize initializes the state manager.
func (sm *StateManager) Initialize() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Set state file path
	cacheDir := utils.GetClaudeCacheHome()
	sm.path = filepath.Join(cacheDir, "state.json")

	// Initialize session
	if sm.state.SessionID == "" {
		sm.state.SessionID = utils.GenerateUUID()
	}
	if sm.state.SessionStart == 0 {
		sm.state.SessionStart = 0 // TODO: Use time.Now().Unix()
	}

	// Load existing state if available
	if err := sm.load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load state: %w", err)
	}

	return nil
}

// load loads state from disk.
func (sm *StateManager) load() error {
	data, err := os.ReadFile(sm.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, sm.state)
}

// save saves state to disk.
func (sm *StateManager) save() error {
	// Ensure directory exists
	dir := filepath.Dir(sm.path)
	if err := utils.EnsureDir(dir); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.path, data, 0600)
}

// ========================================
// Getters
// ========================================

// GetState returns the current state (thread-safe copy).
func (sm *StateManager) GetState() AppState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy
	state := *sm.state
	return state
}

// GetSessionID returns the current session ID.
func (sm *StateManager) GetSessionID() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.SessionID
}

// GetCurrentModel returns the current model.
func (sm *StateManager) GetCurrentModel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.CurrentModel
}

// GetCwd returns the current working directory.
func (sm *StateManager) GetCwd() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.Cwd
}

// GetFeature returns a feature flag.
func (sm *StateManager) GetFeature(name string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.Features[name]
}

// GetCustom returns a custom value.
func (sm *StateManager) GetCustom(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.state.Custom[key]
	return val, ok
}

// ========================================
// Setters
// ========================================

// SetCurrentModel sets the current model.
func (sm *StateManager) SetCurrentModel(model string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.CurrentModel = model
	return sm.save()
}

// SetCwd sets the current working directory.
func (sm *StateManager) SetCwd(cwd string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.Cwd = cwd
	return sm.save()
}

// SetConversationID sets the conversation ID.
func (sm *StateManager) SetConversationID(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.ConversationID = id
	return sm.save()
}

// SetPermissionMode sets the permission mode.
func (sm *StateManager) SetPermissionMode(mode string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.PermissionMode = mode
	return sm.save()
}

// SetFeature sets a feature flag.
func (sm *StateManager) SetFeature(name string, enabled bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state.Features == nil {
		sm.state.Features = make(map[string]bool)
	}
	sm.state.Features[name] = enabled
	return sm.save()
}

// SetCustom sets a custom value.
func (sm *StateManager) SetCustom(key string, value interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state.Custom == nil {
		sm.state.Custom = make(map[string]interface{})
	}
	sm.state.Custom[key] = value
	return sm.save()
}

// IncrementMessageCount increments the message count.
func (sm *StateManager) IncrementMessageCount() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.MessageCount++
	return sm.save()
}

// ========================================
// State Updates
// ========================================

// UpdateState updates multiple state values atomically.
func (sm *StateManager) UpdateState(fn func(*AppState)) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fn(sm.state)
	return sm.save()
}

// Reset resets the state (for new sessions).
func (sm *StateManager) Reset() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state = &AppState{
		SessionID:        utils.GenerateUUID(),
		SessionStart:     0, // TODO: time.Now().Unix()
		UserType:         string(utils.GetUserType()),
		SubscriptionType: string(utils.GetSubscriptionType()),
		Features:         make(map[string]bool),
		Custom:           make(map[string]interface{}),
	}

	return sm.save()
}

// Clear clears all state and removes the state file.
func (sm *StateManager) Clear() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state = &AppState{
		Features: make(map[string]bool),
		Custom:   make(map[string]interface{}),
	}

	if sm.path != "" {
		if err := os.Remove(sm.path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// ========================================
// Snapshot and Restore
// ========================================

// Snapshot creates a snapshot of the current state.
func (sm *StateManager) Snapshot() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return json.Marshal(sm.state)
}

// Restore restores state from a snapshot.
func (sm *StateManager) Restore(data []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return json.Unmarshal(data, sm.state)
}

// ========================================
// Global State Manager
// ========================================

var (
	globalStateManager *StateManager
	globalStateOnce    sync.Once
)

// GetGlobalStateManager returns the global state manager.
func GetGlobalStateManager() *StateManager {
	globalStateOnce.Do(func() {
		globalStateManager = NewStateManager()
	})
	return globalStateManager
}
