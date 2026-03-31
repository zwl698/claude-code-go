// Package types contains core type definitions for the claude-code CLI.
// This file contains plugin-related types translated from TypeScript.
package types

// PluginAuthor represents the author of a plugin.
type PluginAuthor struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Url   string `json:"url,omitempty"`
}

// CommandMetadata represents metadata for a plugin command.
type CommandMetadata struct {
	Description string `json:"description,omitempty"`
	WhenToUse   string `json:"whenToUse,omitempty"`
}

// PluginManifest represents the manifest of a plugin.
type PluginManifest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Author       *PluginAuthor          `json:"author,omitempty"`
	Commands     []string               `json:"commands,omitempty"`
	Agents       []string               `json:"agents,omitempty"`
	Skills       []string               `json:"skills,omitempty"`
	Hooks        map[string]interface{} `json:"hooks,omitempty"`
	McpServers   map[string]interface{} `json:"mcpServers,omitempty"`
	LspServers   map[string]interface{} `json:"lspServers,omitempty"`
	CommandPaths []string               `json:"commandPaths,omitempty"`
	AgentPaths   []string               `json:"agentPaths,omitempty"`
	SkillPaths   []string               `json:"skillPaths,omitempty"`
}

// BuiltinPluginDefinition represents a built-in plugin that ships with the CLI.
type BuiltinPluginDefinition struct {
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Version        string                   `json:"version,omitempty"`
	Skills         []BundledSkillDefinition `json:"skills,omitempty"`
	Hooks          map[string]interface{}   `json:"hooks,omitempty"`
	McpServers     map[string]interface{}   `json:"mcpServers,omitempty"`
	IsAvailable    func() bool              `json:"-"`
	DefaultEnabled bool                     `json:"defaultEnabled,omitempty"`
}

// BundledSkillDefinition represents a bundled skill definition.
type BundledSkillDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
	Tools       []string `json:"tools,omitempty"`
	Model       string   `json:"model,omitempty"`
}

// PluginRepository represents a plugin repository.
type PluginRepository struct {
	Url         string `json:"url"`
	Branch      string `json:"branch"`
	LastUpdated string `json:"lastUpdated,omitempty"`
	CommitSha   string `json:"commitSha,omitempty"`
}

// PluginConfig represents the configuration for plugins.
type PluginConfig struct {
	Repositories map[string]PluginRepository `json:"repositories"`
}

// PluginComponent represents a component of a plugin.
type PluginComponent string

const (
	PluginComponentCommands     PluginComponent = "commands"
	PluginComponentAgents       PluginComponent = "agents"
	PluginComponentSkills       PluginComponent = "skills"
	PluginComponentHooks        PluginComponent = "hooks"
	PluginComponentOutputStyles PluginComponent = "output-styles"
)

// PluginError represents a discriminated union of plugin error types.
type PluginErrorDetail struct {
	Type string `json:"type"`

	// Common fields
	Source string `json:"source"`
	Plugin string `json:"plugin,omitempty"`

	// Path-not-found fields
	Path      string          `json:"path,omitempty"`
	Component PluginComponent `json:"component,omitempty"`

	// Git-related fields
	GitUrl    string `json:"gitUrl,omitempty"`
	AuthType  string `json:"authType,omitempty"`  // ssh | https
	Operation string `json:"operation,omitempty"` // clone | pull

	// Network-related fields
	Url     string `json:"url,omitempty"`
	Details string `json:"details,omitempty"`

	// Manifest-related fields
	ManifestPath     string   `json:"manifestPath,omitempty"`
	ParseError       string   `json:"parseError,omitempty"`
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// Plugin-not-found fields
	PluginId    string `json:"pluginId,omitempty"`
	Marketplace string `json:"marketplace,omitempty"`

	// Marketplace-related fields
	AvailableMarketplaces []string `json:"availableMarketplaces,omitempty"`
	Reason                string   `json:"reason,omitempty"`

	// MCP-related fields
	ServerName      string `json:"serverName,omitempty"`
	ValidationError string `json:"validationError,omitempty"`
	DuplicateOf     string `json:"duplicateOf,omitempty"`

	// Hook-related fields
	HookPath string `json:"hookPath,omitempty"`

	// MCPB-related fields
	McpbPath string `json:"mcpbPath,omitempty"`

	// LSP-related fields
	ExitCode  int    `json:"exitCode,omitempty"`
	Signal    string `json:"signal,omitempty"`
	Method    string `json:"method,omitempty"`
	TimeoutMs int64  `json:"timeoutMs,omitempty"`

	// Policy-related fields
	BlockedByBlocklist bool     `json:"blockedByBlocklist,omitempty"`
	AllowedSources     []string `json:"allowedSources,omitempty"`

	// Dependency-related fields
	Dependency string `json:"dependency,omitempty"`
	DepReason  string `json:"depReason,omitempty"` // not-enabled | not-found

	// Cache-related fields
	InstallPath string `json:"installPath,omitempty"`

	// Generic error field
	Error string `json:"error,omitempty"`
}

// GetPluginErrorMessage returns a human-readable message for a plugin error.
func GetPluginErrorMessage(err PluginErrorDetail) string {
	switch err.Type {
	case "generic-error":
		return err.Error
	case "path-not-found":
		return "Path not found: " + err.Path + " (" + string(err.Component) + ")"
	case "git-auth-failed":
		return "Git authentication failed (" + err.AuthType + "): " + err.GitUrl
	case "git-timeout":
		return "Git " + err.Operation + " timeout: " + err.GitUrl
	case "network-error":
		msg := "Network error: " + err.Url
		if err.Details != "" {
			msg += " - " + err.Details
		}
		return msg
	case "manifest-parse-error":
		return "Manifest parse error: " + err.ParseError
	case "manifest-validation-error":
		return "Manifest validation failed: " + joinErrors(err.ValidationErrors)
	case "plugin-not-found":
		return "Plugin " + err.PluginId + " not found in marketplace " + err.Marketplace
	case "marketplace-not-found":
		return "Marketplace " + err.Marketplace + " not found"
	case "marketplace-load-failed":
		return "Marketplace " + err.Marketplace + " failed to load: " + err.Reason
	case "mcp-config-invalid":
		return "MCP server " + err.ServerName + " invalid: " + err.ValidationError
	case "mcp-server-suppressed-duplicate":
		return "MCP server \"" + err.ServerName + "\" skipped — same command/URL as " + err.DuplicateOf
	case "hook-load-failed":
		return "Hook load failed: " + err.Reason
	case "component-load-failed":
		return string(err.Component) + " load failed from " + err.Path + ": " + err.Reason
	case "mcpb-download-failed":
		return "Failed to download MCPB from " + err.Url + ": " + err.Reason
	case "mcpb-extract-failed":
		return "Failed to extract MCPB " + err.McpbPath + ": " + err.Reason
	case "mcpb-invalid-manifest":
		return "MCPB manifest invalid at " + err.McpbPath + ": " + err.ValidationError
	case "lsp-config-invalid":
		return "Plugin \"" + err.Plugin + "\" has invalid LSP server config for \"" + err.ServerName + "\": " + err.ValidationError
	case "lsp-server-start-failed":
		return "Plugin \"" + err.Plugin + "\" failed to start LSP server \"" + err.ServerName + "\": " + err.Reason
	case "lsp-server-crashed":
		if err.Signal != "" {
			return "Plugin \"" + err.Plugin + "\" LSP server \"" + err.ServerName + "\" crashed with signal " + err.Signal
		}
		return "Plugin \"" + err.Plugin + "\" LSP server \"" + err.ServerName + "\" crashed with exit code"
	case "lsp-request-timeout":
		return "Plugin \"" + err.Plugin + "\" LSP server \"" + err.ServerName + "\" timed out on " + err.Method + " request after"
	case "lsp-request-failed":
		return "Plugin \"" + err.Plugin + "\" LSP server \"" + err.ServerName + "\" " + err.Method + " request failed: " + err.Error
	case "marketplace-blocked-by-policy":
		if err.BlockedByBlocklist {
			return "Marketplace '" + err.Marketplace + "' is blocked by enterprise policy"
		}
		return "Marketplace '" + err.Marketplace + "' is not in the allowed marketplace list"
	case "dependency-unsatisfied":
		hint := "disabled — enable it or remove the dependency"
		if err.DepReason == "not-found" {
			hint = "not found in any configured marketplace"
		}
		return "Dependency \"" + err.Dependency + "\" is " + hint
	case "plugin-cache-miss":
		return "Plugin \"" + err.Plugin + "\" not cached at " + err.InstallPath + " — run /plugins to refresh"
	default:
		return err.Error
	}
}

// joinErrors joins multiple error messages.
func joinErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	result := errors[0]
	for i := 1; i < len(errors); i++ {
		result += ", " + errors[i]
	}
	return result
}

// PluginLoadResult represents the result of loading plugins.
type PluginLoadResult struct {
	Enabled  []LoadedPlugin      `json:"enabled"`
	Disabled []LoadedPlugin      `json:"disabled"`
	Errors   []PluginErrorDetail `json:"errors"`
}
