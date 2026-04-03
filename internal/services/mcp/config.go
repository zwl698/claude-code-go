package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnterpriseMCPFileName is the filename for enterprise MCP config
const EnterpriseMCPFileName = "managed-mcp.json"

// McpJsonConfig represents the MCP JSON configuration file structure
type McpJsonConfig struct {
	MCPServers map[string]McpServerConfig `json:"mcpServers"`
}

// GetEnterpriseMcpFilePath returns the path to the managed MCP configuration file.
func GetEnterpriseMcpFilePath(managedFilePath string) string {
	return filepath.Join(managedFilePath, EnterpriseMCPFileName)
}

// AddScopeToServers adds scope information to server configs.
func AddScopeToServers(servers map[string]McpServerConfig, scope ConfigScope) map[string]ScopedMcpServerConfig {
	if servers == nil {
		return make(map[string]ScopedMcpServerConfig)
	}

	scopedServers := make(map[string]ScopedMcpServerConfig)
	for name, config := range servers {
		scopedServers[name] = ScopedMcpServerConfig{
			McpServerConfig: config,
			Scope:           scope,
		}
	}
	return scopedServers
}

// McpServerNameEntry represents a name-based MCP server entry
type McpServerNameEntry struct {
	ServerName string `json:"serverName"`
}

// McpServerCommandEntry represents a command-based MCP server entry
type McpServerCommandEntry struct {
	ServerCommand []string `json:"serverCommand"`
}

// McpServerUrlEntry represents a URL-based MCP server entry
type McpServerUrlEntry struct {
	ServerUrl string `json:"serverUrl"`
}

// McpServerAllowlistEntry is the union type for allowlist/denylist entries
type McpServerAllowlistEntry struct {
	ServerName    string   `json:"serverName,omitempty"`
	ServerCommand []string `json:"serverCommand,omitempty"`
	ServerUrl     string   `json:"serverUrl,omitempty"`
}

// IsMcpServerNameEntry checks if entry is a name-based entry
func IsMcpServerNameEntry(entry McpServerAllowlistEntry) bool {
	return entry.ServerName != ""
}

// IsMcpServerCommandEntry checks if entry is a command-based entry
func IsMcpServerCommandEntry(entry McpServerAllowlistEntry) bool {
	return len(entry.ServerCommand) > 0
}

// IsMcpServerUrlEntry checks if entry is a URL-based entry
func IsMcpServerUrlEntry(entry McpServerAllowlistEntry) bool {
	return entry.ServerUrl != ""
}

// PolicySettings represents policy-related settings
type PolicySettings struct {
	AllowManagedMcpServersOnly bool `json:"allowManagedMcpServersOnly,omitempty"`
}

// SettingsJson represents the settings JSON structure
type SettingsJson struct {
	AllowedMcpServers []McpServerAllowlistEntry `json:"allowedMcpServers,omitempty"`
	DeniedMcpServers  []McpServerAllowlistEntry `json:"deniedMcpServers,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	File             string            `json:"file,omitempty"`
	Path             string            `json:"path"`
	Message          string            `json:"message"`
	Suggestion       string            `json:"suggestion,omitempty"`
	McpErrorMetadata *McpErrorMetadata `json:"mcpErrorMetadata,omitempty"`
}

// McpErrorMetadata represents MCP error metadata
type McpErrorMetadata struct {
	Scope      ConfigScope `json:"scope"`
	ServerName string      `json:"serverName,omitempty"`
	Severity   string      `json:"severity"`
}

// =============================================================================
// Config Loading
// =============================================================================

// McpConfigLoader handles loading MCP configurations
type McpConfigLoader struct {
	platform string
}

// NewMcpConfigLoader creates a new MCP config loader
func NewMcpConfigLoader() *McpConfigLoader {
	return &McpConfigLoader{
		platform: "unix",
	}
}

// LoadFromFile loads MCP config from a file
func (l *McpConfigLoader) LoadFromFile(path string, scope ConfigScope) (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return l.ParseConfig(data, path, scope)
}

// ParseConfig parses MCP configuration
func (l *McpConfigLoader) ParseConfig(data []byte, path string, scope ConfigScope) (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	var rawConfig struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}

	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	servers := make(map[string]ScopedMcpServerConfig)
	var errors []ValidationError

	for name, rawData := range rawConfig.MCPServers {
		config, err := ParseServerConfig(rawData)
		if err != nil {
			errors = append(errors, ValidationError{
				File:    path,
				Path:    "mcpServers." + name,
				Message: fmt.Sprintf("Failed to parse server config: %v", err),
				McpErrorMetadata: &McpErrorMetadata{
					Scope:      scope,
					ServerName: name,
					Severity:   "error",
				},
			})
			continue
		}

		servers[name] = ScopedMcpServerConfig{
			McpServerConfig: config,
			Scope:           scope,
		}
	}

	return servers, errors, nil
}

// ParseServerConfig parses raw JSON into a specific server config type
func ParseServerConfig(data json.RawMessage) (McpServerConfig, error) {
	var typeDetector struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeDetector); err != nil {
		return nil, err
	}

	transport := Transport(typeDetector.Type)

	switch transport {
	case TransportStdio, "":
		var config McpStdioServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		config.Type = TransportStdio
		return &config, nil

	case TransportSSE:
		var config McpSSEServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case "sse-ide":
		var config McpSSEIDEServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case TransportHTTP:
		var config McpHTTPServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case TransportWS:
		var config McpWebSocketServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case "ws-ide":
		var config McpWebSocketIDEServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case TransportSDK:
		var config McpSdkServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	case "claudeai-proxy":
		var config McpClaudeAIProxyServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil

	default:
		return nil, fmt.Errorf("unknown transport type: %s", transport)
	}
}

// ExpandEnvVars expands environment variables in a config
func ExpandEnvVars(config McpServerConfig) McpServerConfig {
	switch c := config.(type) {
	case *McpStdioServerConfig:
		result := *c
		result.Command = os.ExpandEnv(c.Command)
		if c.Args != nil {
			result.Args = make([]string, len(c.Args))
			for i, arg := range c.Args {
				result.Args[i] = os.ExpandEnv(arg)
			}
		}
		if c.Env != nil {
			result.Env = make(map[string]string)
			for k, v := range c.Env {
				result.Env[k] = os.ExpandEnv(v)
			}
		}
		return &result

	case *McpSSEServerConfig:
		result := *c
		result.URL = os.ExpandEnv(c.URL)
		return &result

	case *McpHTTPServerConfig:
		result := *c
		result.URL = os.ExpandEnv(c.URL)
		return &result

	case *McpWebSocketServerConfig:
		result := *c
		result.URL = os.ExpandEnv(c.URL)
		return &result

	default:
		return config
	}
}

// =============================================================================
// Config Validation
// =============================================================================

// ValidateConfig validates an MCP server config
func ValidateConfig(name string, config McpServerConfig) []ValidationError {
	var errors []ValidationError

	switch c := config.(type) {
	case *McpStdioServerConfig:
		if c.Command == "" {
			errors = append(errors, ValidationError{
				Path:    "mcpServers." + name + ".command",
				Message: "command is required for stdio transport",
			})
		}

		if strings.HasSuffix(c.Command, "npx") || strings.HasSuffix(c.Command, "\\npx") {
			// Warning for Windows users
		}

	case *McpSSEServerConfig:
		if c.URL == "" {
			errors = append(errors, ValidationError{
				Path:    "mcpServers." + name + ".url",
				Message: "url is required for SSE transport",
			})
		}

	case *McpHTTPServerConfig:
		if c.URL == "" {
			errors = append(errors, ValidationError{
				Path:    "mcpServers." + name + ".url",
				Message: "url is required for HTTP transport",
			})
		}

	case *McpWebSocketServerConfig:
		if c.URL == "" {
			errors = append(errors, ValidationError{
				Path:    "mcpServers." + name + ".url",
				Message: "url is required for WebSocket transport",
			})
		}
	}

	return errors
}

// =============================================================================
// Config Helpers
// =============================================================================

// GetConfigPath returns the default MCP config path
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "mcp.json")
}

// LoadMcpConfig loads MCP config from default locations
func LoadMcpConfig() (map[string]ScopedMcpServerConfig, error) {
	loader := NewMcpConfigLoader()
	allServers := make(map[string]ScopedMcpServerConfig)

	userConfig, _, err := loader.LoadFromFile(GetConfigPath(), ConfigScopeUser)
	if err != nil {
		return nil, err
	}
	for name, config := range userConfig {
		allServers[name] = config
	}

	wd, _ := os.Getwd()
	projectConfig, _, err := loader.LoadFromFile(filepath.Join(wd, ".mcp.json"), ConfigScopeProject)
	if err == nil {
		for name, config := range projectConfig {
			allServers[name] = config
		}
	}

	return allServers, nil
}

// GetServerCommand extracts the command from a stdio config
func GetServerCommand(config McpServerConfig) (string, []string) {
	if stdio, ok := config.(*McpStdioServerConfig); ok {
		return stdio.Command, stdio.Args
	}
	return "", nil
}

// GetServerURL extracts the URL from a URL-based config
func GetServerURL(config McpServerConfig) string {
	switch c := config.(type) {
	case *McpSSEServerConfig:
		return c.URL
	case *McpHTTPServerConfig:
		return c.URL
	case *McpWebSocketServerConfig:
		return c.URL
	case *McpSSEIDEServerConfig:
		return c.URL
	case *McpWebSocketIDEServerConfig:
		return c.URL
	default:
		return ""
	}
}

// IsStdioConfigType checks if config is stdio type
func IsStdioConfigType(config McpServerConfig) bool {
	_, ok := config.(*McpStdioServerConfig)
	return ok
}

// IsURLConfigType checks if config is URL-based
func IsURLConfigType(config McpServerConfig) bool {
	switch config.(type) {
	case *McpSSEServerConfig, *McpHTTPServerConfig, *McpWebSocketServerConfig:
		return true
	default:
		return false
	}
}
