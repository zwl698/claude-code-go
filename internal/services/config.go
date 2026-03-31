package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"claude-code-go/internal/types"
)

// ConfigManager manages application configuration.
type ConfigManager struct {
	mu       sync.RWMutex
	config   *Config
	filePath string
}

// Config represents the application configuration.
type Config struct {
	// API settings
	APIKey       string `json:"apiKey,omitempty"`
	APIKeySource string `json:"apiKeySource,omitempty"`
	BaseURL      string `json:"baseUrl,omitempty"`

	// Model settings
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`

	// Permission settings
	PermissionMode string `json:"permissionMode,omitempty"`

	// Feature flags
	AutoUpdaterEnabled bool `json:"autoUpdaterEnabled"`
	ThinkingEnabled    bool `json:"thinkingEnabled"`

	// UI settings
	Theme    string `json:"theme,omitempty"`
	Verbose  bool   `json:"verbose"`
	Debug    bool   `json:"debug"`
	ShowTips bool   `json:"showTips"`

	// MCP settings
	MCPServers map[string]MCPServerConfig `json:"mcpServers,omitempty"`

	// Hooks
	Hooks map[string]HookConfig `json:"hooks,omitempty"`

	// Other settings
	AdditionalDirectories []string `json:"additionalDirectories,omitempty"`
}

// MCPServerConfig represents MCP server configuration.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
}

// HookConfig represents hook configuration.
type HookConfig struct {
	Event   string `json:"event"`
	Command string `json:"command"`
	Enabled bool   `json:"enabled"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Model:              "claude-sonnet-4-20250514",
		Temperature:        1.0,
		MaxTokens:          4096,
		PermissionMode:     string(types.PermissionModeDefault),
		AutoUpdaterEnabled: true,
		ThinkingEnabled:    false,
		Theme:              "dark",
		Verbose:            false,
		Debug:              false,
		ShowTips:           true,
		MCPServers:         make(map[string]MCPServerConfig),
		Hooks:              make(map[string]HookConfig),
	}
}

// NewConfigManager creates a new config manager.
func NewConfigManager() (*ConfigManager, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	cm := &ConfigManager{
		config:   DefaultConfig(),
		filePath: filepath.Join(configDir, "config.json"),
	}

	// Load existing config
	if _, err := os.Stat(cm.filePath); err == nil {
		if err := cm.Load(); err != nil {
			// Non-fatal error, use defaults
			fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		}
	}

	// Load from environment variables
	cm.loadFromEnv()

	return cm, nil
}

// GetConfigDir returns the config directory.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// loadFromEnv loads configuration from environment variables.
func (cm *ConfigManager) loadFromEnv() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cm.config.APIKey = apiKey
		cm.config.APIKeySource = "environment"
	}

	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		cm.config.BaseURL = baseURL
	}

	if model := os.Getenv("CLAUDE_MODEL"); model != "" {
		cm.config.Model = model
	}

	if permMode := os.Getenv("CLAUDE_PERMISSION_MODE"); permMode != "" {
		cm.config.PermissionMode = permMode
	}

	if os.Getenv("CLAUDE_VERBOSE") == "true" {
		cm.config.Verbose = true
	}

	if os.Getenv("CLAUDE_DEBUG") == "true" {
		cm.config.Debug = true
	}
}

// Load loads the configuration from file.
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cm.config)
}

// Save saves the configuration to file.
func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.filePath, data, 0644)
}

// Get returns the current configuration.
func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// Set updates the configuration.
func (cm *ConfigManager) Set(config *Config) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = config
}

// Update applies a function to update the configuration.
func (cm *ConfigManager) Update(fn func(*Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	fn(cm.config)
}

// GetAPIKey returns the API key.
func (cm *ConfigManager) GetAPIKey() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.APIKey
}

// SetAPIKey sets the API key.
func (cm *ConfigManager) SetAPIKey(apiKey string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.APIKey = apiKey
	cm.config.APIKeySource = "user"
}

// GetModel returns the current model.
func (cm *ConfigManager) GetModel() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Model
}

// SetModel sets the model.
func (cm *ConfigManager) SetModel(model string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Model = model
}

// GetPermissionMode returns the permission mode.
func (cm *ConfigManager) GetPermissionMode() types.PermissionMode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return types.PermissionMode(cm.config.PermissionMode)
}

// SetPermissionMode sets the permission mode.
func (cm *ConfigManager) SetPermissionMode(mode types.PermissionMode) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.PermissionMode = string(mode)
}

// GetMCPServers returns MCP server configurations.
func (cm *ConfigManager) GetMCPServers() map[string]MCPServerConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make(map[string]MCPServerConfig)
	for k, v := range cm.config.MCPServers {
		result[k] = v
	}
	return result
}

// AddMCPServer adds an MCP server configuration.
func (cm *ConfigManager) AddMCPServer(name string, config MCPServerConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.config.MCPServers == nil {
		cm.config.MCPServers = make(map[string]MCPServerConfig)
	}
	cm.config.MCPServers[name] = config
}

// RemoveMCPServer removes an MCP server configuration.
func (cm *ConfigManager) RemoveMCPServer(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.config.MCPServers, name)
}
