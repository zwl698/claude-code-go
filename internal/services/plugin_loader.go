package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/hooks"
	"claude-code-go/internal/services/mcp"
	"claude-code-go/internal/types"
)

// =============================================================================
// Advanced Plugin Loader
// =============================================================================

// PluginLoader handles loading plugins with full integration.
type PluginLoader struct {
	mu              sync.RWMutex
	pluginService   *PluginService
	mcpManager      *mcp.ConnectionManager
	hookRegistry    *hooks.Registry
	store           *types.AppStateStore
	pluginDirs      []string
	loadedManifests map[string]*types.PluginManifest
	errors          []types.PluginErrorDetail
}

// NewPluginLoader creates a new plugin loader.
func NewPluginLoader(store *types.AppStateStore, mcpManager *mcp.ConnectionManager, hookRegistry *hooks.Registry) *PluginLoader {
	return &PluginLoader{
		pluginService:   GetPluginService(),
		mcpManager:      mcpManager,
		hookRegistry:    hookRegistry,
		store:           store,
		pluginDirs:      getDefaultPluginDirs(),
		loadedManifests: make(map[string]*types.PluginManifest),
		errors:          make([]types.PluginErrorDetail, 0),
	}
}

// getDefaultPluginDirs returns default plugin directories.
func getDefaultPluginDirs() []string {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return []string{
		filepath.Join(home, ".claude", "plugins"),
		filepath.Join(cwd, ".claude", "plugins"),
		filepath.Join(cwd, "plugins"),
	}
}

// LoadAll loads all plugins from configured directories.
func (l *PluginLoader) LoadAll(ctx context.Context) *types.PluginLoadResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := &types.PluginLoadResult{
		Enabled:  make([]types.LoadedPlugin, 0),
		Disabled: make([]types.LoadedPlugin, 0),
		Errors:   make([]types.PluginErrorDetail, 0),
	}

	for _, dir := range l.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			result.Errors = append(result.Errors, types.PluginErrorDetail{
				Type:    "generic-error",
				Source:  dir,
				Error:   fmt.Sprintf("Failed to read directory: %v", err),
				Details: err.Error(),
			})
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginPath := filepath.Join(dir, entry.Name())
			pluginResult := l.loadPluginFromPath(ctx, pluginPath)

			if pluginResult.Error != nil {
				result.Errors = append(result.Errors, *pluginResult.Error)
			}

			if pluginResult.Plugin != nil {
				if pluginResult.Plugin.Enabled {
					result.Enabled = append(result.Enabled, *pluginResult.Plugin)
				} else {
					result.Disabled = append(result.Disabled, *pluginResult.Plugin)
				}
			}
		}
	}

	// Update app state
	if l.store != nil {
		l.store.Update(func(state *types.AppState) *types.AppState {
			state.Plugins.Enabled = result.Enabled
			state.Plugins.Disabled = result.Disabled
			// Convert PluginErrorDetail to PluginError
			for _, err := range result.Errors {
				state.Plugins.Errors = append(state.Plugins.Errors, types.PluginError{
					Name:    err.Plugin,
					Message: types.GetPluginErrorMessage(err),
					Type:    err.Type,
				})
			}
			return state
		})
	}

	return result
}

// pluginLoadResult represents the result of loading a single plugin.
type pluginLoadResult struct {
	Plugin *types.LoadedPlugin
	Error  *types.PluginErrorDetail
}

// loadPluginFromPath loads a plugin from a directory path.
func (l *PluginLoader) loadPluginFromPath(ctx context.Context, path string) pluginLoadResult {
	manifestPath := filepath.Join(path, "manifest.json")

	// Try manifest.json first, then plugin.json
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(path, "plugin.json")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return pluginLoadResult{
			Error: &types.PluginErrorDetail{
				Type:            "path-not-found",
				Source:          path,
				Path:            manifestPath,
				Component:       types.PluginComponentCommands,
				ManifestPath:    manifestPath,
				ValidationError: err.Error(),
			},
		}
	}

	var manifest types.PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return pluginLoadResult{
			Error: &types.PluginErrorDetail{
				Type:         "manifest-parse-error",
				Source:       path,
				ParseError:   err.Error(),
				ManifestPath: manifestPath,
			},
		}
	}

	// Validate manifest
	if validationErrors := validateManifest(&manifest); len(validationErrors) > 0 {
		return pluginLoadResult{
			Error: &types.PluginErrorDetail{
				Type:             "manifest-validation-error",
				Source:           path,
				ValidationErrors: validationErrors,
				ManifestPath:     manifestPath,
				ValidationError:  strings.Join(validationErrors, "; "),
			},
		}
	}

	// Create loaded plugin
	loadedPlugin := &types.LoadedPlugin{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Path:        path,
		Enabled:     true,
		Description: manifest.Description,
	}

	// Load MCP servers from plugin
	if manifest.McpServers != nil && l.mcpManager != nil {
		if err := l.loadMCPServers(ctx, manifest.Name, manifest.McpServers); err != nil {
			l.errors = append(l.errors, types.PluginErrorDetail{
				Type:       "mcp-config-invalid",
				Source:     path,
				Plugin:     manifest.Name,
				ServerName: "",
				Reason:     err.Error(),
				Error:      err.Error(),
			})
		}
	}

	// Load hooks from plugin
	if manifest.Hooks != nil && l.hookRegistry != nil {
		if err := l.loadHooks(manifest.Name, manifest.Hooks); err != nil {
			l.errors = append(l.errors, types.PluginErrorDetail{
				Type:   "hook-load-failed",
				Source: path,
				Plugin: manifest.Name,
				Reason: err.Error(),
				Error:  err.Error(),
			})
		}
	}

	// Store manifest
	l.loadedManifests[manifest.Name] = &manifest

	return pluginLoadResult{Plugin: loadedPlugin}
}

// validateManifest validates a plugin manifest.
func validateManifest(manifest *types.PluginManifest) []string {
	errors := make([]string, 0)

	if manifest.Name == "" {
		errors = append(errors, "plugin name is required")
	}

	if manifest.Version == "" {
		errors = append(errors, "plugin version is required")
	}

	// Validate MCP servers if present
	if manifest.McpServers != nil {
		for serverName, serverConfig := range manifest.McpServers {
			if serverConfig == nil {
				errors = append(errors, fmt.Sprintf("MCP server '%s' has nil config", serverName))
			}
		}
	}

	return errors
}

// loadMCPServers loads MCP servers from plugin configuration.
func (l *PluginLoader) loadMCPServers(ctx context.Context, pluginName string, servers map[string]interface{}) error {
	for serverName, serverConfig := range servers {
		configMap, ok := serverConfig.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid MCP server config for '%s'", serverName)
		}

		// Create MCP server config
		config := convertToMCPConfig(configMap)

		// Add to MCP manager
		if l.mcpManager != nil {
			if err := l.mcpManager.AddServer(serverName, config, mcp.ConfigScopeProject); err != nil {
				return fmt.Errorf("failed to add MCP server '%s': %w", serverName, err)
			}

			// Try to connect
			if err := l.mcpManager.Connect(ctx, serverName); err != nil {
				// Non-fatal: server might not be available
				continue
			}
		}
	}
	return nil
}

// convertToMCPConfig converts a map to an MCP server config.
func convertToMCPConfig(configMap map[string]interface{}) mcp.McpServerConfig {
	// Check for URL-based config
	if url, ok := configMap["url"].(string); ok {
		transport := mcp.TransportSSE
		if t, ok := configMap["type"].(string); ok {
			transport = mcp.Transport(t)
		}
		return &mcp.McpSSEServerConfig{
			Type:    transport,
			URL:     url,
			Headers: convertHeaders(configMap["headers"]),
		}
	}

	// Stdio config
	cmd, _ := configMap["command"].(string)
	args := convertArgs(configMap["args"])
	env := convertEnv(configMap["env"])

	return &mcp.McpStdioServerConfig{
		Type:    mcp.TransportStdio,
		Command: cmd,
		Args:    args,
		Env:     env,
	}
}

// convertHeaders converts headers from interface.
func convertHeaders(headers interface{}) map[string]string {
	if headers == nil {
		return nil
	}
	if h, ok := headers.(map[string]string); ok {
		return h
	}
	if h, ok := headers.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range h {
			if vs, ok := v.(string); ok {
				result[k] = vs
			}
		}
		return result
	}
	return nil
}

// convertArgs converts args from interface.
func convertArgs(args interface{}) []string {
	if args == nil {
		return nil
	}
	if a, ok := args.([]string); ok {
		return a
	}
	if a, ok := args.([]interface{}); ok {
		result := make([]string, 0, len(a))
		for _, v := range a {
			if vs, ok := v.(string); ok {
				result = append(result, vs)
			}
		}
		return result
	}
	return nil
}

// convertEnv converts env from interface.
func convertEnv(env interface{}) map[string]string {
	if env == nil {
		return nil
	}
	if e, ok := env.(map[string]string); ok {
		return e
	}
	if e, ok := env.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range e {
			if vs, ok := v.(string); ok {
				result[k] = vs
			}
		}
		return result
	}
	return nil
}

// loadHooks loads hooks from plugin configuration.
func (l *PluginLoader) loadHooks(pluginName string, hooksConfig map[string]interface{}) error {
	for eventName, hookConfig := range hooksConfig {
		if !hooks.IsHookEvent(eventName) {
			return fmt.Errorf("invalid hook event: %s", eventName)
		}

		configMap, ok := hookConfig.(map[string]interface{})
		if !ok {
			continue
		}

		command, _ := configMap["command"].(string)
		if command == "" {
			continue
		}

		// Create a hook handler
		handler := createHookHandler(pluginName, command, configMap)

		// Register the hook
		l.hookRegistry.Register(hooks.HookEvent(eventName), handler)
	}
	return nil
}

// createHookHandler creates a hook handler from plugin configuration.
func createHookHandler(pluginName, command string, config map[string]interface{}) hooks.HookHandler {
	return func(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
		// Execute the hook command
		// This is a simplified implementation - in production, this would
		// spawn a subprocess and capture its output
		output := hooks.HookOutput{
			Continue: true,
		}

		// For now, just log the hook execution
		fmt.Printf("[Plugin %s] Hook %s executed: %s\n", pluginName, input.EventName, command)

		return output, nil
	}
}

// GetPluginManifest returns the loaded manifest for a plugin.
func (l *PluginLoader) GetPluginManifest(name string) *types.PluginManifest {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loadedManifests[name]
}

// GetErrors returns all loading errors.
func (l *PluginLoader) GetErrors() []types.PluginErrorDetail {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.errors
}

// =============================================================================
// Plugin Marketplace
// =============================================================================

// MarketplaceClient handles plugin marketplace operations.
type MarketplaceClient struct {
	repositories map[string]types.PluginRepository
	cacheDir     string
}

// NewMarketplaceClient creates a new marketplace client.
func NewMarketplaceClient() *MarketplaceClient {
	home, _ := os.UserHomeDir()
	return &MarketplaceClient{
		repositories: make(map[string]types.PluginRepository),
		cacheDir:     filepath.Join(home, ".claude", "plugin-cache"),
	}
}

// AddRepository adds a plugin repository.
func (c *MarketplaceClient) AddRepository(name, url, branch string) {
	c.repositories[name] = types.PluginRepository{
		Url:    url,
		Branch: branch,
	}
}

// SearchPlugins searches for plugins in configured repositories.
func (c *MarketplaceClient) SearchPlugins(query string) ([]types.PluginManifest, error) {
	// Placeholder - in production this would query actual repositories
	return []types.PluginManifest{
		{
			Name:        "example-plugin",
			Version:     "1.0.0",
			Description: "An example plugin",
		},
	}, nil
}

// InstallPlugin installs a plugin from a marketplace.
func (c *MarketplaceClient) InstallPlugin(ctx context.Context, pluginID string) error {
	// Placeholder - in production this would download and install the plugin
	return nil
}

// =============================================================================
// Plugin Tools Integration
// =============================================================================

// PluginToolWrapper wraps a plugin tool for use in the tool system.
type PluginToolWrapper struct {
	name        string
	description string
	inputSchema types.ToolInputJSONSchema
	pluginName  string
	handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// NewPluginToolWrapper creates a new plugin tool wrapper.
func NewPluginToolWrapper(pluginName, toolName, description string, schema types.ToolInputJSONSchema, handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)) *PluginToolWrapper {
	return &PluginToolWrapper{
		name:        fmt.Sprintf("plugin__%s__%s", pluginName, toolName),
		description: description,
		inputSchema: schema,
		pluginName:  pluginName,
		handler:     handler,
	}
}

// GetName returns the tool name.
func (t *PluginToolWrapper) GetName() string {
	return t.name
}

// GetDescription returns the tool description.
func (t *PluginToolWrapper) GetDescription() string {
	return t.description
}

// GetInputSchema returns the input schema.
func (t *PluginToolWrapper) GetInputSchema() types.ToolInputJSONSchema {
	return t.inputSchema
}

// Call executes the plugin tool.
func (t *PluginToolWrapper) Call(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if t.handler == nil {
		return nil, fmt.Errorf("plugin tool '%s' has no handler", t.name)
	}
	return t.handler(ctx, args)
}

// =============================================================================
// Plugin Commands Integration
// =============================================================================

// PluginCommandWrapper wraps a plugin command for use in the command system.
type PluginCommandWrapper struct {
	name        string
	description string
	pluginName  string
	handler     string // Path to handler script
}

// NewPluginCommandWrapper creates a new plugin command wrapper.
func NewPluginCommandWrapper(pluginName, cmdName, description, handler string) *PluginCommandWrapper {
	return &PluginCommandWrapper{
		name:        fmt.Sprintf("plugin:%s:%s", pluginName, cmdName),
		description: description,
		pluginName:  pluginName,
		handler:     handler,
	}
}

// Execute executes the plugin command.
func (c *PluginCommandWrapper) Execute(ctx context.Context, args []string) error {
	// Placeholder - in production this would execute the handler script
	fmt.Printf("[Plugin %s] Command %s executed with args: %v\n", c.pluginName, c.name, args)
	return nil
}

// =============================================================================
// Built-in Plugins
// =============================================================================

// GetBuiltinPlugins returns the list of built-in plugins.
func GetBuiltinPlugins() []types.BuiltinPluginDefinition {
	return []types.BuiltinPluginDefinition{
		{
			Name:        "git",
			Description: "Git integration plugin",
			Version:     "1.0.0",
			Skills: []types.BundledSkillDefinition{
				{
					Name:        "commit",
					Description: "Create a git commit with changes",
					Prompt:      "Create a git commit with all staged changes",
					Tools:       []string{"Bash"},
				},
				{
					Name:        "branch",
					Description: "Create and switch to a new branch",
					Prompt:      "Create and switch to a new git branch",
					Tools:       []string{"Bash"},
				},
			},
			DefaultEnabled: true,
		},
		{
			Name:        "test",
			Description: "Test runner plugin",
			Version:     "1.0.0",
			Skills: []types.BundledSkillDefinition{
				{
					Name:        "run",
					Description: "Run tests",
					Prompt:      "Run the test suite",
					Tools:       []string{"Bash"},
				},
				{
					Name:        "coverage",
					Description: "Run tests with coverage",
					Prompt:      "Run tests with coverage report",
					Tools:       []string{"Bash"},
				},
			},
			DefaultEnabled: true,
		},
	}
}

// LoadBuiltinPlugins loads built-in plugins.
func (l *PluginLoader) LoadBuiltinPlugins(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, def := range GetBuiltinPlugins() {
		if def.IsAvailable != nil && !def.IsAvailable() {
			continue
		}

		loadedPlugin := types.LoadedPlugin{
			Name:        def.Name,
			Version:     def.Version,
			Path:        "builtin:" + def.Name,
			Enabled:     def.DefaultEnabled,
			Description: def.Description,
		}

		// Store manifest
		l.loadedManifests[def.Name] = &types.PluginManifest{
			Name:        def.Name,
			Version:     def.Version,
			Description: def.Description,
			Skills:      convertSkills(def.Skills),
		}

		// Update app state
		if l.store != nil {
			l.store.Update(func(state *types.AppState) *types.AppState {
				if def.DefaultEnabled {
					state.Plugins.Enabled = append(state.Plugins.Enabled, loadedPlugin)
				} else {
					state.Plugins.Disabled = append(state.Plugins.Disabled, loadedPlugin)
				}
				return state
			})
		}
	}

	return nil
}

// convertSkills converts bundled skills to skill paths.
func convertSkills(skills []types.BundledSkillDefinition) []string {
	paths := make([]string, len(skills))
	for i, skill := range skills {
		paths[i] = skill.Name
	}
	return paths
}

// =============================================================================
// Global Plugin Loader Instance
// =============================================================================

var globalPluginLoader *PluginLoader
var pluginLoaderOnce sync.Once

// GetPluginLoader returns the global plugin loader instance.
func GetPluginLoader(store *types.AppStateStore, mcpManager *mcp.ConnectionManager, hookRegistry *hooks.Registry) *PluginLoader {
	pluginLoaderOnce.Do(func() {
		globalPluginLoader = NewPluginLoader(store, mcpManager, hookRegistry)
	})
	return globalPluginLoader
}

// InitializePlugins initializes all plugins.
func InitializePlugins(ctx context.Context, store *types.AppStateStore, mcpManager *mcp.ConnectionManager, hookRegistry *hooks.Registry) (*types.PluginLoadResult, error) {
	loader := GetPluginLoader(store, mcpManager, hookRegistry)

	// Load built-in plugins first
	if err := loader.LoadBuiltinPlugins(ctx); err != nil {
		return nil, fmt.Errorf("failed to load built-in plugins: %w", err)
	}

	// Load external plugins
	result := loader.LoadAll(ctx)

	return result, nil
}

// =============================================================================
// Plugin Refresh
// =============================================================================

// RefreshPlugins refreshes the plugin list.
func RefreshPlugins(ctx context.Context, store *types.AppStateStore, mcpManager *mcp.ConnectionManager, hookRegistry *hooks.Registry) (*types.PluginLoadResult, error) {
	// Reset the loader
	globalPluginLoader = nil
	pluginLoaderOnce = sync.Once{}

	// Reinitialize
	return InitializePlugins(ctx, store, mcpManager, hookRegistry)
}

// =============================================================================
// Plugin Installation Status
// =============================================================================

// InstallPluginFromSource installs a plugin from a source path or URL.
func InstallPluginFromSource(ctx context.Context, source string, store *types.AppStateStore, mcpManager *mcp.ConnectionManager, hookRegistry *hooks.Registry) error {
	loader := GetPluginLoader(store, mcpManager, hookRegistry)

	// For URL sources, download first
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// TODO: Implement download
		return fmt.Errorf("URL installation not yet implemented")
	}

	// For git sources, clone first
	if strings.HasPrefix(source, "git@") || strings.HasPrefix(source, "https://github.com/") {
		// TODO: Implement git clone
		return fmt.Errorf("git installation not yet implemented")
	}

	// For local paths, load directly
	result := loader.loadPluginFromPath(ctx, source)
	if result.Error != nil {
		return fmt.Errorf("failed to load plugin: %v", result.Error)
	}

	return nil
}

// =============================================================================
// Plugin Cleanup
// =============================================================================

// CleanupPluginCache cleans up the plugin cache.
func CleanupPluginCache() error {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".claude", "plugin-cache")

	// Remove old cached plugins (older than 30 days)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -30)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.RemoveAll(filepath.Join(cacheDir, entry.Name()))
		}
	}

	return nil
}
