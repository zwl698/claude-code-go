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

// McpConfigLoader handles loading MCP configurations
type McpConfigLoader struct {
	homeDir     string
	cwd         string
	platform    string
	envProvider func(string) string
}

// NewMcpConfigLoader creates a new MCP config loader
func NewMcpConfigLoader(homeDir, cwd, platform string, envProvider func(string) string) *McpConfigLoader {
	return &McpConfigLoader{
		homeDir:     homeDir,
		cwd:         cwd,
		platform:    platform,
		envProvider: envProvider,
	}
}

// GetGlobalClaudeFile returns the path to the global Claude config file
func (l *McpConfigLoader) GetGlobalClaudeFile() string {
	return filepath.Join(l.homeDir, ".claude.json")
}

// GetProjectMcpJsonPath returns the path to .mcp.json in current directory
func (l *McpConfigLoader) GetProjectMcpJsonPath() string {
	return filepath.Join(l.cwd, ".mcp.json")
}

// LoadGlobalConfig loads the global Claude config
func (l *McpConfigLoader) LoadGlobalConfig() (map[string]interface{}, error) {
	data, err := os.ReadFile(l.GetGlobalClaudeFile())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	return config, nil
}

// LoadProjectMcpConfig loads the project-level .mcp.json
func (l *McpConfigLoader) LoadProjectMcpConfig() (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	return l.LoadMcpConfigFromPath(l.GetProjectMcpJsonPath(), ConfigScopeProject)
}

// LoadMcpConfigFromPath loads MCP config from a specific path
func (l *McpConfigLoader) LoadMcpConfigFromPath(path string, scope ConfigScope) (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]ScopedMcpServerConfig), nil, nil
		}
		return nil, nil, err
	}

	var rawConfig struct {
		McpServers map[string]McpServerConfig `json:"mcpServers"`
	}

	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, []ValidationError{{
			File:    path,
			Path:    "",
			Message: "MCP config is not a valid JSON",
			McpErrorMetadata: &McpErrorMetadata{
				Scope:    scope,
				Severity: "fatal",
			},
		}}, nil
	}

	// Expand environment variables and validate
	errors := make([]ValidationError, 0)
	servers := make(map[string]ScopedMcpServerConfig)

	for name, config := range rawConfig.McpServers {
		// Expand env vars
		expanded := l.expandEnvVars(config)

		// Check for Windows npx usage without cmd wrapper
		if l.platform == "windows" {
			if (config.Type == "" || config.Type == string(TransportStdio)) &&
				(config.Command == "npx" || strings.HasSuffix(config.Command, "\\npx") || strings.HasSuffix(config.Command, "/npx")) {
				errors = append(errors, ValidationError{
					File:       path,
					Path:       "mcpServers." + name,
					Message:    "Windows requires 'cmd /c' wrapper to execute npx",
					Suggestion: "Change command to \"cmd\" with args [\"/c\", \"npx\", ...]",
					McpErrorMetadata: &McpErrorMetadata{
						Scope:      scope,
						ServerName: name,
						Severity:   "warning",
					},
				})
			}
		}

		servers[name] = ScopedMcpServerConfig{
			McpServerConfig: expanded,
			Scope:           scope,
		}
	}

	return servers, errors, nil
}

// expandEnvVars expands environment variables in the config
func (l *McpConfigLoader) expandEnvVars(config McpServerConfig) McpServerConfig {
	result := config

	// Expand command
	result.Command = l.expandString(config.Command)

	// Expand args
	if config.Args != nil {
		result.Args = make([]string, len(config.Args))
		for i, arg := range config.Args {
			result.Args[i] = l.expandString(arg)
		}
	}

	// Expand env values
	if config.Env != nil {
		result.Env = make(map[string]string)
		for k, v := range config.Env {
			result.Env[k] = l.expandString(v)
		}
	}

	// Expand URL
	result.URL = l.expandString(config.URL)

	// Expand headers
	if config.Headers != nil {
		result.Headers = make(map[string]string)
		for k, v := range config.Headers {
			result.Headers[k] = l.expandString(v)
		}
	}

	return result
}

// expandString expands environment variables in a string.
// Supports:
//   - ${VAR} - simple variable reference
//   - ${VAR:-default} - variable with default value if not set
//   - $VAR - simple variable reference (without braces)
func (l *McpConfigLoader) expandString(s string) string {
	if s == "" {
		return s
	}

	result := s

	// Replace ${VAR} and ${VAR:-default} patterns
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varContent := result[start+2 : end]
		varName := varContent
		defaultValue := ""

		// Check for :- syntax (default value)
		if idx := strings.Index(varContent, ":-"); idx != -1 {
			varName = varContent[:idx]
			defaultValue = varContent[idx+2:]
		}

		varValue := ""
		if l.envProvider != nil {
			varValue = l.envProvider(varName)
		}
		if varValue == "" {
			varValue = defaultValue
		}

		result = result[:start] + varValue + result[end+1:]
	}

	// Replace $VAR patterns (simple form without braces)
	for i := 0; i < len(result); i++ {
		if result[i] == '$' && i+1 < len(result) && result[i+1] != '{' {
			j := i + 1
			for j < len(result) && (result[j] >= 'a' && result[j] <= 'z' || result[j] >= 'A' && result[j] <= 'Z' || result[j] >= '0' && result[j] <= '9' || result[j] == '_') {
				j++
			}
			if j > i+1 {
				varName := result[i+1 : j]
				varValue := ""
				if l.envProvider != nil {
					varValue = l.envProvider(varName)
				}
				result = result[:i] + varValue + result[j:]
				i = i + len(varValue) - 1
			}
		}
	}

	return result
}

// GetMcpConfigsByScope gets all MCP configurations from a specific scope
func (l *McpConfigLoader) GetMcpConfigsByScope(scope ConfigScope) (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	switch scope {
	case ConfigScopeProject:
		return l.loadProjectConfigsRecursive()
	case ConfigScopeUser:
		return l.loadUserConfigs()
	case ConfigScopeLocal:
		return l.loadLocalConfigs()
	case ConfigScopeEnterprise:
		return l.loadEnterpriseConfigs()
	default:
		return make(map[string]ScopedMcpServerConfig), nil, nil
	}
}

// loadProjectConfigsRecursive loads project configs from current dir up to root
func (l *McpConfigLoader) loadProjectConfigsRecursive() (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	allServers := make(map[string]ScopedMcpServerConfig)
	allErrors := make([]ValidationError, 0)

	// Build list of directories to check
	dirs := make([]string, 0)
	currentDir := l.cwd

	for {
		dirs = append(dirs, currentDir)
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	// Process from root downward to CWD (closer files have higher priority)
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		mcpJsonPath := filepath.Join(dir, ".mcp.json")

		servers, errors, err := l.LoadMcpConfigFromPath(mcpJsonPath, ConfigScopeProject)
		if err != nil {
			continue
		}

		// Merge servers
		for name, config := range servers {
			allServers[name] = config
		}

		allErrors = append(allErrors, errors...)
	}

	return allServers, allErrors, nil
}

// loadUserConfigs loads user-scoped MCP configs
func (l *McpConfigLoader) loadUserConfigs() (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	config, err := l.LoadGlobalConfig()
	if err != nil {
		return nil, nil, err
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return make(map[string]ScopedMcpServerConfig), nil, nil
	}

	servers := make(map[string]ScopedMcpServerConfig)
	for name, serverConfig := range mcpServers {
		configBytes, err := json.Marshal(serverConfig)
		if err != nil {
			continue
		}

		var mcpConfig McpServerConfig
		if err := json.Unmarshal(configBytes, &mcpConfig); err != nil {
			continue
		}

		servers[name] = ScopedMcpServerConfig{
			McpServerConfig: l.expandEnvVars(mcpConfig),
			Scope:           ConfigScopeUser,
		}
	}

	return servers, nil, nil
}

// loadLocalConfigs loads local-scoped MCP configs
func (l *McpConfigLoader) loadLocalConfigs() (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	// Local configs are stored in .claude/settings.json in the project
	localSettingsPath := filepath.Join(l.cwd, ".claude", "settings.json")

	data, err := os.ReadFile(localSettingsPath)
	if err != nil {
		return make(map[string]ScopedMcpServerConfig), nil, nil
	}

	var config struct {
		McpServers map[string]McpServerConfig `json:"mcpServers,omitempty"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, []ValidationError{{
			File:    localSettingsPath,
			Path:    "",
			Message: "Local settings is not a valid JSON",
			McpErrorMetadata: &McpErrorMetadata{
				Scope:    ConfigScopeLocal,
				Severity: "fatal",
			},
		}}, nil
	}

	return AddScopeToServers(config.McpServers, ConfigScopeLocal), nil, nil
}

// loadEnterpriseConfigs loads enterprise-scoped MCP configs
func (l *McpConfigLoader) loadEnterpriseConfigs() (map[string]ScopedMcpServerConfig, []ValidationError, error) {
	// Enterprise configs are in managed path
	managedPath := filepath.Join(l.homeDir, ".claude", "managed")
	enterprisePath := GetEnterpriseMcpFilePath(managedPath)

	return l.LoadMcpConfigFromPath(enterprisePath, ConfigScopeEnterprise)
}

// ValidateServerName validates an MCP server name
func ValidateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("invalid name %q: names can only contain letters, numbers, hyphens, and underscores", name)
		}
	}

	return nil
}

// ReservedServerNames contains reserved server names that cannot be added
var ReservedServerNames = map[string]bool{
	"claude-in-chrome": true,
}

// IsReservedServerName checks if a server name is reserved
func IsReservedServerName(name string) bool {
	return ReservedServerNames[name]
}

// DedupPluginMcpServers filters plugin MCP servers, dropping any whose signature matches
// a manually-configured server or an earlier-loaded plugin server.
// Manual wins over plugin; between plugins, first-loaded wins.
func DedupPluginMcpServers(
	pluginServers map[string]ScopedMcpServerConfig,
	manualServers map[string]ScopedMcpServerConfig,
) (servers map[string]ScopedMcpServerConfig, suppressed []DedupResult) {
	// Map signature -> server name
	manualSigs := make(map[string]string)
	for name, config := range manualServers {
		sig := GetMcpServerSignature(config.McpServerConfig)
		if sig != "" {
			if _, exists := manualSigs[sig]; !exists {
				manualSigs[sig] = name
			}
		}
	}

	servers = make(map[string]ScopedMcpServerConfig)
	seenPluginSigs := make(map[string]string)

	for name, config := range pluginServers {
		sig := GetMcpServerSignature(config.McpServerConfig)

		if sig == "" {
			servers[name] = config
			continue
		}

		if manualDup, exists := manualSigs[sig]; exists {
			suppressed = append(suppressed, DedupResult{
				Name:        name,
				DuplicateOf: manualDup,
			})
			continue
		}

		if pluginDup, exists := seenPluginSigs[sig]; exists {
			suppressed = append(suppressed, DedupResult{
				Name:        name,
				DuplicateOf: pluginDup,
			})
			continue
		}

		seenPluginSigs[sig] = name
		servers[name] = config
	}

	return servers, suppressed
}

// DedupClaudeAiMcpServers filters claude.ai connectors, dropping any whose signature matches
// an enabled manually-configured server. Manual wins.
func DedupClaudeAiMcpServers(
	claudeAiServers map[string]ScopedMcpServerConfig,
	manualServers map[string]ScopedMcpServerConfig,
	isDisabled func(string) bool,
) (servers map[string]ScopedMcpServerConfig, suppressed []DedupResult) {
	manualSigs := make(map[string]string)
	for name, config := range manualServers {
		if isDisabled(name) {
			continue
		}
		sig := GetMcpServerSignature(config.McpServerConfig)
		if sig != "" {
			if _, exists := manualSigs[sig]; !exists {
				manualSigs[sig] = name
			}
		}
	}

	servers = make(map[string]ScopedMcpServerConfig)
	for name, config := range claudeAiServers {
		sig := GetMcpServerSignature(config.McpServerConfig)

		if manualDup, exists := manualSigs[sig]; exists {
			suppressed = append(suppressed, DedupResult{
				Name:        name,
				DuplicateOf: manualDup,
			})
			continue
		}

		servers[name] = config
	}

	return servers, suppressed
}

// DedupResult represents a deduplication result
type DedupResult struct {
	Name        string
	DuplicateOf string
}

// FilterMcpServersByPolicy filters MCP servers by policy (allowedMcpServers/deniedMcpServers)
func FilterMcpServersByPolicy(
	configs map[string]ScopedMcpServerConfig,
	settings SettingsJson,
	isDisabled func(string) bool,
) (allowed map[string]ScopedMcpServerConfig, blocked []string) {
	allowed = make(map[string]ScopedMcpServerConfig)

	for name, config := range configs {
		// SDK servers are exempt from policy filtering
		if config.Type == string(TransportSDK) {
			allowed[name] = config
			continue
		}

		if !isMcpServerAllowedByPolicy(name, config.McpServerConfig, settings, isDisabled) {
			blocked = append(blocked, name)
			continue
		}

		allowed[name] = config
	}

	return allowed, blocked
}

// isMcpServerAllowedByPolicy checks if an MCP server is allowed by enterprise policy
func isMcpServerAllowedByPolicy(
	serverName string,
	config McpServerConfig,
	settings SettingsJson,
	isDisabled func(string) bool,
) bool {
	// Denylist takes absolute precedence
	if isMcpServerDenied(serverName, config, settings) {
		return false
	}

	// No allowlist restrictions
	if settings.AllowedMcpServers == nil {
		return true
	}

	// Empty allowlist means block all servers
	if len(settings.AllowedMcpServers) == 0 {
		return false
	}

	// Check command-based allowance for stdio servers
	serverCommand := GetServerCommandArray(config)
	if serverCommand != nil {
		hasCommandEntries := false
		for _, entry := range settings.AllowedMcpServers {
			if IsMcpServerCommandEntry(entry) {
				hasCommandEntries = true
				if CommandArraysMatch(entry.ServerCommand, serverCommand) {
					return true
				}
			}
		}
		if hasCommandEntries {
			return false
		}
	}

	// Check URL-based allowance for remote servers
	serverUrl := config.URL
	if serverUrl != "" {
		hasUrlEntries := false
		for _, entry := range settings.AllowedMcpServers {
			if IsMcpServerUrlEntry(entry) {
				hasUrlEntries = true
				if UrlMatchesPattern(serverUrl, entry.ServerUrl) {
					return true
				}
			}
		}
		if hasUrlEntries {
			return false
		}
	}

	// Check name-based allowance
	for _, entry := range settings.AllowedMcpServers {
		if IsMcpServerNameEntry(entry) && entry.ServerName == serverName {
			return true
		}
	}

	return false
}

// isMcpServerDenied checks if an MCP server is denied by enterprise policy
func isMcpServerDenied(serverName string, config McpServerConfig, settings SettingsJson) bool {
	if settings.DeniedMcpServers == nil {
		return false
	}

	// Check name-based denial
	for _, entry := range settings.DeniedMcpServers {
		if IsMcpServerNameEntry(entry) && entry.ServerName == serverName {
			return true
		}
	}

	// Check command-based denial (stdio servers)
	serverCommand := GetServerCommandArray(config)
	if serverCommand != nil {
		for _, entry := range settings.DeniedMcpServers {
			if IsMcpServerCommandEntry(entry) && CommandArraysMatch(entry.ServerCommand, serverCommand) {
				return true
			}
		}
	}

	// Check URL-based denial (remote servers)
	serverUrl := config.URL
	if serverUrl != "" {
		for _, entry := range settings.DeniedMcpServers {
			if IsMcpServerUrlEntry(entry) && UrlMatchesPattern(serverUrl, entry.ServerUrl) {
				return true
			}
		}
	}

	return false
}
