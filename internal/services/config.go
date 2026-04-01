package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"claude-code-go/internal/types"
)

// =============================================================================
// Config Manager
// =============================================================================

// ConfigManager manages application configuration.
type ConfigManager struct {
	mu         sync.RWMutex
	config     *Config
	filePath   string
	projectCfg *ProjectConfig // Project-level config
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

	// Provider settings
	Provider string `json:"provider,omitempty"` // anthropic, bedrock, vertex, foundry

	// AWS Bedrock settings
	AWSRegion          string `json:"awsRegion,omitempty"`
	AWSProfile         string `json:"awsProfile,omitempty"`
	AWSAccessKeyID     string `json:"awsAccessKeyId,omitempty"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`

	// Google Vertex AI settings
	GCPProjectID string `json:"gcpProjectId,omitempty"`
	GCPRegion    string `json:"gcpRegion,omitempty"`
	GCPCredsFile string `json:"gcpCredsFile,omitempty"`

	// Azure Foundry settings
	AzureResourceGroup string `json:"azureResourceGroup,omitempty"`
	AzureSubscription  string `json:"azureSubscription,omitempty"`
	AzureTenant        string `json:"azureTenant,omitempty"`

	// History settings
	MaxHistorySize int  `json:"maxHistorySize,omitempty"`
	SaveHistory    bool `json:"saveHistory"`

	// Editor settings
	DefaultEditor string `json:"defaultEditor,omitempty"`
}

// ProjectConfig represents project-level configuration.
type ProjectConfig struct {
	// Project identification
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`

	// Project-specific model settings
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`

	// Project-specific permissions
	PermissionMode string   `json:"permissionMode,omitempty"`
	AllowedTools   []string `json:"allowedTools,omitempty"`
	DeniedTools    []string `json:"deniedTools,omitempty"`

	// Project-specific MCP servers
	MCPServers map[string]MCPServerConfig `json:"mcpServers,omitempty"`

	// Project context
	ContextPaths   []string `json:"contextPaths,omitempty"`
	IgnorePatterns []string `json:"ignorePatterns,omitempty"`

	// Custom system prompt
	SystemPrompt string `json:"systemPrompt,omitempty"`
}

// MCPServerConfig represents MCP server configuration.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
	// Optional URL for SSE transport
	URL string `json:"url,omitempty"`
	// Tool filtering
	AllowedTools []string `json:"allowedTools,omitempty"`
	DeniedTools  []string `json:"deniedTools,omitempty"`
}

// HookConfig represents hook configuration.
type HookConfig struct {
	Event   string `json:"event"`
	Command string `json:"command"`
	Enabled bool   `json:"enabled"`
	// Optional timeout for hook execution
	TimeoutMs int `json:"timeoutMs,omitempty"`
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
		SaveHistory:        true,
		MaxHistorySize:     1000,
		MCPServers:         make(map[string]MCPServerConfig),
		Hooks:              make(map[string]HookConfig),
		Provider:           "anthropic",
		DefaultEditor:      "vim",
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

	// Load project config if available
	cm.loadProjectConfig()

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

	// Load provider settings
	if os.Getenv("CLAUDE_CODE_USE_BEDROCK") == "true" {
		cm.config.Provider = "bedrock"
	}
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "true" {
		cm.config.Provider = "vertex"
	}
	if os.Getenv("CLAUDE_CODE_USE_FOUNDRY") == "true" {
		cm.config.Provider = "foundry"
	}

	// AWS settings
	if region := os.Getenv("AWS_REGION"); region != "" {
		cm.config.AWSRegion = region
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		cm.config.AWSProfile = profile
	}

	// GCP settings
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		cm.config.GCPProjectID = project
	}
	if region := os.Getenv("GOOGLE_CLOUD_REGION"); region != "" {
		cm.config.GCPRegion = region
	}
	if creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); creds != "" {
		cm.config.GCPCredsFile = creds
	}
}

// loadProjectConfig loads project-level configuration.
func (cm *ConfigManager) loadProjectConfig() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	projectCfgPath := filepath.Join(cwd, ".claude", "project.json")
	if _, err := os.Stat(projectCfgPath); err != nil {
		return
	}

	data, err := os.ReadFile(projectCfgPath)
	if err != nil {
		return
	}

	var projectCfg ProjectConfig
	if err := json.Unmarshal(data, &projectCfg); err != nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.projectCfg = &projectCfg

	// Apply project-level overrides
	if projectCfg.Model != "" {
		cm.config.Model = projectCfg.Model
	}
	if projectCfg.PermissionMode != "" {
		cm.config.PermissionMode = projectCfg.PermissionMode
	}
	if projectCfg.Temperature > 0 {
		cm.config.Temperature = projectCfg.Temperature
	}

	// Merge MCP servers
	for name, server := range projectCfg.MCPServers {
		if cm.config.MCPServers == nil {
			cm.config.MCPServers = make(map[string]MCPServerConfig)
		}
		cm.config.MCPServers[name] = server
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

// GetProvider returns the current provider.
func (cm *ConfigManager) GetProvider() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Provider
}

// SetProvider sets the provider.
func (cm *ConfigManager) SetProvider(provider string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Provider = provider
}

// GetProjectConfig returns the project configuration.
func (cm *ConfigManager) GetProjectConfig() *ProjectConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.projectCfg
}

// IsToolAllowed checks if a tool is allowed by project config.
func (cm *ConfigManager) IsToolAllowed(toolName string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.projectCfg == nil {
		return true
	}

	// Check denied tools first
	for _, denied := range cm.projectCfg.DeniedTools {
		if toolName == denied {
			return false
		}
	}

	// If allowed tools specified, check against it
	if len(cm.projectCfg.AllowedTools) > 0 {
		for _, allowed := range cm.projectCfg.AllowedTools {
			if toolName == allowed || allowed == "*" {
				return true
			}
		}
		return false
	}

	return true
}

// GetSystemPrompt returns the custom system prompt if set.
func (cm *ConfigManager) GetSystemPrompt() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.projectCfg != nil && cm.projectCfg.SystemPrompt != "" {
		return cm.projectCfg.SystemPrompt
	}
	return ""
}

// =============================================================================
// Settings File Helpers
// =============================================================================

// GetSettingsPath returns the path to a settings file.
func GetSettingsPath(filename string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, filename), nil
}

// LoadSettings loads a settings file into a struct.
func LoadSettings(filename string, v interface{}) error {
	path, err := GetSettingsPath(filename)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// SaveSettings saves a struct to a settings file.
func SaveSettings(filename string, v interface{}) error {
	path, err := GetSettingsPath(filename)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ExpandEnvVars expands environment variables in strings.
// Supports ${VAR} and ${VAR:-default} syntax.
func ExpandEnvVars(s string) string {
	// Handle ${VAR:-default} syntax
	defaultPattern := "${"
	for {
		start := strings.Index(s, defaultPattern)
		if start == -1 {
			break
		}

		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}

		expr := s[start+2 : start+end]
		var value string

		if idx := strings.Index(expr, ":-"); idx != -1 {
			varName := expr[:idx]
			defaultVal := expr[idx+2:]
			value = os.Getenv(varName)
			if value == "" {
				value = defaultVal
			}
		} else {
			value = os.Getenv(expr)
		}

		s = s[:start] + value + s[start+end+1:]
	}

	// Handle ${VAR} syntax
	return os.ExpandEnv(s)
}
