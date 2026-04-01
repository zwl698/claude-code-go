package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Plugin Service - Plugin Management
// =============================================================================

// PluginState represents the state of a plugin
type PluginState string

const (
	PluginStateInstalled   PluginState = "installed"
	PluginStateEnabled     PluginState = "enabled"
	PluginStateDisabled    PluginState = "disabled"
	PluginStateError       PluginState = "error"
	PluginStateUninstalled PluginState = "uninstalled"
)

// Plugin represents a loaded plugin
type Plugin struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Path        string          `json:"path"`
	State       PluginState     `json:"state"`
	Config      json.RawMessage `json:"config,omitempty"`
	Commands    []PluginCommand `json:"commands,omitempty"`
	Tools       []PluginTool    `json:"tools,omitempty"`
	LoadedAt    time.Time       `json:"loadedAt"`
	Description string          `json:"description"`
}

// PluginCommand represents a command provided by a plugin
type PluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Handler     string `json:"handler"`
}

// PluginTool represents a tool provided by a plugin
type PluginTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// PluginManifest represents a plugin manifest file
type PluginManifest struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Main        string          `json:"main"`
	Commands    []PluginCommand `json:"commands,omitempty"`
	Tools       []PluginTool    `json:"tools,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
}

// PluginService manages plugins
type PluginService struct {
	mu           sync.RWMutex
	plugins      map[string]*Plugin
	pluginDirs   []string
	errorHandler func(pluginName, operation string, err error)
}

// NewPluginService creates a new plugin service
func NewPluginService() *PluginService {
	return &PluginService{
		plugins: make(map[string]*Plugin),
		pluginDirs: []string{
			"./plugins",
			"./.claude/plugins",
			filepath.Join(os.Getenv("HOME"), ".claude", "plugins"),
		},
	}
}

// LoadAllPlugins loads all plugins from plugin directories
func (s *PluginService) LoadAllPlugins() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, dir := range s.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginPath := filepath.Join(dir, entry.Name())
			_ = s.loadPluginLocked(pluginPath)
		}
	}

	return nil
}

// loadPluginLocked loads a single plugin (must hold lock)
func (s *PluginService) loadPluginLocked(pluginPath string) error {
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	plugin := &Plugin{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Path:        pluginPath,
		State:       PluginStateEnabled,
		Config:      manifest.Config,
		Commands:    manifest.Commands,
		Tools:       manifest.Tools,
		LoadedAt:    time.Now(),
		Description: manifest.Description,
	}

	s.plugins[manifest.Name] = plugin
	return nil
}

// GetPlugin gets a plugin by name
func (s *PluginService) GetPlugin(name string) (*Plugin, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plugin, ok := s.plugins[name]
	return plugin, ok
}

// GetAllPlugins returns all plugins
func (s *PluginService) GetAllPlugins() map[string]*Plugin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*Plugin)
	for k, v := range s.plugins {
		result[k] = v
	}
	return result
}

// EnablePlugin enables a plugin
func (s *PluginService) EnablePlugin(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plugin, ok := s.plugins[name]
	if !ok {
		return nil
	}

	plugin.State = PluginStateEnabled
	return nil
}

// DisablePlugin disables a plugin
func (s *PluginService) DisablePlugin(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plugin, ok := s.plugins[name]
	if !ok {
		return nil
	}

	plugin.State = PluginStateDisabled
	return nil
}

// InstallPlugin installs a plugin from a path or URL
func (s *PluginService) InstallPlugin(source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// For now, just load from the source path
	return s.loadPluginLocked(source)
}

// UninstallPlugin uninstalls a plugin
func (s *PluginService) UninstallPlugin(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.plugins, name)
	return nil
}

// RefreshPlugins refreshes the plugin list
func (s *PluginService) RefreshPlugins() error {
	return s.LoadAllPlugins()
}

// GetCommands returns all plugin commands
func (s *PluginService) GetCommands() []PluginCommand {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var commands []PluginCommand
	for _, plugin := range s.plugins {
		if plugin.State == PluginStateEnabled {
			commands = append(commands, plugin.Commands...)
		}
	}
	return commands
}

// GetTools returns all plugin tools
func (s *PluginService) GetTools() []PluginTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tools []PluginTool
	for _, plugin := range s.plugins {
		if plugin.State == PluginStateEnabled {
			tools = append(tools, plugin.Tools...)
		}
	}
	return tools
}

// AddPluginDir adds a plugin directory
func (s *PluginService) AddPluginDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pluginDirs = append(s.pluginDirs, dir)
}

// SetErrorHandler sets the error handler
func (s *PluginService) SetErrorHandler(handler func(pluginName, operation string, err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorHandler = handler
}

// FindPluginsByKeyword finds plugins by keyword
func (s *PluginService) FindPluginsByKeyword(keyword string) []*Plugin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyword = strings.ToLower(keyword)
	var results []*Plugin
	for _, plugin := range s.plugins {
		if strings.Contains(strings.ToLower(plugin.Name), keyword) ||
			strings.Contains(strings.ToLower(plugin.Description), keyword) {
			results = append(results, plugin)
		}
	}
	return results
}

// Global instance
var pluginServiceInstance *PluginService
var pluginOnce sync.Once

// GetPluginService returns the global plugin service instance
func GetPluginService() *PluginService {
	pluginOnce.Do(func() {
		pluginServiceInstance = NewPluginService()
	})
	return pluginServiceInstance
}
