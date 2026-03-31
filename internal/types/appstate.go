package types

import (
	"sync"
	"time"
)

// CompletionBoundary represents a point where speculation can complete.
type CompletionBoundary struct {
	Type         string    `json:"type"` // complete | bash | edit | denied_tool
	CompletedAt  time.Time `json:"completedAt,omitempty"`
	OutputTokens int       `json:"outputTokens,omitempty"`
	Command      string    `json:"command,omitempty"`
	ToolName     string    `json:"toolName,omitempty"`
	FilePath     string    `json:"filePath,omitempty"`
	Detail       string    `json:"detail,omitempty"`
}

// SpeculationResult represents the result of speculation execution.
type SpeculationResult struct {
	Messages    []Message           `json:"messages"`
	Boundary    *CompletionBoundary `json:"boundary,omitempty"`
	TimeSavedMs int64               `json:"timeSavedMs"`
}

// SpeculationState represents the current state of speculation.
type SpeculationState struct {
	Status    string              `json:"status"` // idle | active
	Id        string              `json:"id,omitempty"`
	StartTime time.Time           `json:"startTime,omitempty"`
	Boundary  *CompletionBoundary `json:"boundary,omitempty"`
}

// FooterItem represents the items that can appear in the footer.
type FooterItem string

const (
	FooterItemTasks     FooterItem = "tasks"
	FooterItemTmux      FooterItem = "tmux"
	FooterItemBagel     FooterItem = "bagel"
	FooterItemTeams     FooterItem = "teams"
	FooterItemBridge    FooterItem = "bridge"
	FooterItemCompanion FooterItem = "companion"
)

// AppState represents the global application state.
type AppState struct {
	mu sync.RWMutex

	// Core settings
	Settings                SettingsJson `json:"settings"`
	Verbose                 bool         `json:"verbose"`
	MainLoopModel           string       `json:"mainLoopModel"`
	MainLoopModelForSession string       `json:"mainLoopModelForSession"`
	StatusLineText          string       `json:"statusLineText,omitempty"`
	ExpandedView            string       `json:"expandedView"` // none | tasks | teammates
	IsBriefOnly             bool         `json:"isBriefOnly"`

	// Agent and teammate state
	ShowTeammateMessagePreview bool        `json:"showTeammateMessagePreview,omitempty"`
	SelectedIPAgentIndex       int         `json:"selectedIPAgentIndex"`
	CoordinatorTaskIndex       int         `json:"coordinatorTaskIndex"`
	ViewSelectionMode          string      `json:"viewSelectionMode"` // none | selecting-agent | viewing-agent
	FooterSelection            *FooterItem `json:"footerSelection,omitempty"`

	// Permission context
	ToolPermissionContext ToolPermissionContext `json:"toolPermissionContext"`
	SpinnerTip            string                `json:"spinnerTip,omitempty"`

	// Agent configuration
	Agent         string `json:"agent,omitempty"`
	KairosEnabled bool   `json:"kairosEnabled"`

	// Remote session state
	RemoteSessionUrl          string `json:"remoteSessionUrl,omitempty"`
	RemoteConnectionStatus    string `json:"remoteConnectionStatus"` // connecting | connected | reconnecting | disconnected
	RemoteBackgroundTaskCount int    `json:"remoteBackgroundTaskCount"`

	// Bridge state (always-on bridge)
	ReplBridgeEnabled       bool   `json:"replBridgeEnabled"`
	ReplBridgeExplicit      bool   `json:"replBridgeExplicit"`
	ReplBridgeOutboundOnly  bool   `json:"replBridgeOutboundOnly"`
	ReplBridgeConnected     bool   `json:"replBridgeConnected"`
	ReplBridgeSessionActive bool   `json:"replBridgeSessionActive"`
	ReplBridgeReconnecting  bool   `json:"replBridgeReconnecting"`
	ReplBridgeConnectUrl    string `json:"replBridgeConnectUrl,omitempty"`
	ReplBridgeSessionUrl    string `json:"replBridgeSessionUrl,omitempty"`
	ReplBridgeEnvironmentId string `json:"replBridgeEnvironmentId,omitempty"`
	ReplBridgeSessionId     string `json:"replBridgeSessionId,omitempty"`
	ReplBridgeError         string `json:"replBridgeError,omitempty"`
	ReplBridgeInitialName   string `json:"replBridgeInitialName,omitempty"`
	ShowRemoteCallout       bool   `json:"showRemoteCallout"`

	// Task management
	Tasks              map[string]TaskStateBase `json:"tasks"`
	AgentNameRegistry  map[string]AgentId       `json:"agentNameRegistry"`
	ForegroundedTaskId string                   `json:"foregroundedTaskId,omitempty"`
	ViewingAgentTaskId string                   `json:"viewingAgentTaskId,omitempty"`

	// Companion state
	CompanionReaction string    `json:"companionReaction,omitempty"`
	CompanionPetAt    time.Time `json:"companionPetAt,omitempty"`

	// MCP state
	MCP MCPState `json:"mcp"`

	// Plugin state
	Plugins PluginState `json:"plugins"`

	// Agent definitions
	AgentDefinitions AgentDefinitionsResult `json:"agentDefinitions"`

	// File history and attribution
	FileHistory FileHistoryState    `json:"fileHistory"`
	Attribution AttributionState    `json:"attribution"`
	Todos       map[string]TodoList `json:"todos"`

	// Suggestions and notifications
	RemoteAgentTaskSuggestions []TaskSuggestion  `json:"remoteAgentTaskSuggestions,omitempty"`
	Notifications              NotificationState `json:"notifications"`
	Elicitation                ElicitationState  `json:"elicitation"`

	// Thinking and prompt suggestion
	ThinkingEnabled         bool `json:"thinkingEnabled,omitempty"`
	PromptSuggestionEnabled bool `json:"promptSuggestionEnabled"`

	// Session hooks
	SessionHooks SessionHooksState `json:"sessionHooks"`

	// Tungsten/tmux state
	TungstenActiveSession    *TungstenSession `json:"tungstenActiveSession,omitempty"`
	TungstenLastCapturedTime time.Time        `json:"tungstenLastCapturedTime,omitempty"`
	TungstenLastCommand      *TungstenCommand `json:"tungstenLastCommand,omitempty"`
	TungstenPanelVisible     bool             `json:"tungstenPanelVisible,omitempty"`
	TungstenPanelAutoHidden  bool             `json:"tungstenPanelAutoHidden,omitempty"`

	// WebBrowser (bagel) state
	BagelActive  bool   `json:"bagelActive,omitempty"`
	BagelPageUrl string `json:"bagelPageUrl,omitempty"`
}

// SettingsJson represents the settings configuration.
type SettingsJson struct {
	Theme              string `json:"theme,omitempty"`
	PermissionMode     string `json:"permissionMode,omitempty"`
	AutoUpdaterEnabled bool   `json:"autoUpdaterEnabled,omitempty"`
	// Add other settings as needed
}

// MCPState contains MCP-related state.
type MCPState struct {
	Clients            []MCPServerConnection       `json:"clients"`
	Tools              []Tool                      `json:"tools"`
	Commands           []Command                   `json:"commands"`
	Resources          map[string][]ServerResource `json:"resources"`
	PluginReconnectKey int                         `json:"pluginReconnectKey"`
}

// PluginState contains plugin-related state.
type PluginState struct {
	Enabled            []LoadedPlugin     `json:"enabled"`
	Disabled           []LoadedPlugin     `json:"disabled"`
	Commands           []Command          `json:"commands"`
	Errors             []PluginError      `json:"errors"`
	InstallationStatus InstallationStatus `json:"installationStatus"`
	NeedsRefresh       bool               `json:"needsRefresh"`
}

// LoadedPlugin represents a loaded plugin.
type LoadedPlugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

// PluginError represents an error from plugin loading.
type PluginError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// InstallationStatus represents the status of plugin/marketplace installation.
type InstallationStatus struct {
	Marketplaces []MarketplaceStatus   `json:"marketplaces"`
	Plugins      []PluginInstallStatus `json:"plugins"`
}

// MarketplaceStatus represents the installation status of a marketplace.
type MarketplaceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pending | installing | installed | failed
	Error  string `json:"error,omitempty"`
}

// PluginInstallStatus represents the installation status of a plugin.
type PluginInstallStatus struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"` // pending | installing | installed | failed
	Error  string `json:"error,omitempty"`
}

// FileHistoryState tracks file edit history.
type FileHistoryState struct {
	Files map[string]FileHistoryEntry `json:"files"`
}

// FileHistoryEntry represents the history of edits to a file.
type FileHistoryEntry struct {
	Path     string    `json:"path"`
	Original string    `json:"original,omitempty"`
	Current  string    `json:"current,omitempty"`
	Modified time.Time `json:"modified"`
}

// AttributionState tracks commit attribution.
type AttributionState struct {
	Enabled     bool   `json:"enabled"`
	AuthorName  string `json:"authorName,omitempty"`
	AuthorEmail string `json:"authorEmail,omitempty"`
}

// TodoList represents a list of todos for an agent.
type TodoList struct {
	AgentId string     `json:"agentId"`
	Items   []TodoItem `json:"items"`
}

// TodoItem represents a single todo item.
type TodoItem struct {
	Id      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"` // pending | in_progress | completed | cancelled
}

// TaskSuggestion represents a suggested task.
type TaskSuggestion struct {
	Summary string `json:"summary"`
	Task    string `json:"task"`
}

// NotificationState manages the notification queue.
type NotificationState struct {
	Current *Notification  `json:"current,omitempty"`
	Queue   []Notification `json:"queue"`
}

// Notification represents a user notification.
type Notification struct {
	Id        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title,omitempty"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ElicitationState manages elicitation requests.
type ElicitationState struct {
	Queue []ElicitationRequest `json:"queue"`
}

// ElicitationRequest represents an elicitation request from an MCP server.
type ElicitationRequest struct {
	Id         string                 `json:"id"`
	ServerName string                 `json:"serverName"`
	Params     map[string]interface{} `json:"params"`
}

// SessionHooksState tracks session hooks.
type SessionHooksState struct {
	Registered map[string]HookConfig `json:"registered"`
}

// HookConfig represents the configuration for a hook.
type HookConfig struct {
	Name    string `json:"name"`
	Event   string `json:"event"`
	Command string `json:"command"`
	Enabled bool   `json:"enabled"`
	Timeout int    `json:"timeout,omitempty"`
}

// TungstenSession represents an active tungsten/tmux session.
type TungstenSession struct {
	SessionName string `json:"sessionName"`
	SocketName  string `json:"socketName"`
	Target      string `json:"target"`
}

// TungstenCommand represents the last command sent to tungsten.
type TungstenCommand struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

// AppStateStore manages the application state with reactive updates.
type AppStateStore struct {
	state     *AppState
	listeners []func(*AppState)
	mu        sync.RWMutex
}

// NewAppStateStore creates a new state store.
func NewAppStateStore() *AppStateStore {
	return &AppStateStore{
		state: &AppState{
			Tasks:             make(map[string]TaskStateBase),
			AgentNameRegistry: make(map[string]AgentId),
			Todos:             make(map[string]TodoList),
			Settings:          SettingsJson{},
		},
		listeners: make([]func(*AppState), 0),
	}
}

// Get returns the current state.
func (s *AppStateStore) Get() *AppState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set updates the state and notifies listeners.
func (s *AppStateStore) Set(newState *AppState) {
	s.mu.Lock()
	s.state = newState
	listeners := make([]func(*AppState), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()

	for _, listener := range listeners {
		listener(newState)
	}
}

// Update applies a function to update the state.
func (s *AppStateStore) Update(updater func(*AppState) *AppState) {
	s.mu.Lock()
	s.state = updater(s.state)
	listeners := make([]func(*AppState), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()

	for _, listener := range listeners {
		listener(s.state)
	}
}

// Subscribe adds a listener for state changes.
func (s *AppStateStore) Subscribe(listener func(*AppState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

// Unsubscribe removes a listener.
func (s *AppStateStore) Unsubscribe(listener func(*AppState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, l := range s.listeners {
		if &l == &listener {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}
