package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"claude-code-go/internal/utils"
)

// ConnectionManager manages MCP server connections
type ConnectionManager struct {
	mu          sync.RWMutex
	clients     map[string]*Client
	connections map[string]*MCPServerConnection
	tools       map[string]SerializedTool
	configPath  string
}

// NewConnectionManager creates a new MCP connection manager
func NewConnectionManager() *ConnectionManager {
	home, _ := os.UserHomeDir()
	return &ConnectionManager{
		clients:     make(map[string]*Client),
		connections: make(map[string]*MCPServerConnection),
		tools:       make(map[string]SerializedTool),
		configPath:  filepath.Join(home, ".claude", "mcp.json"),
	}
}

// LoadConfig loads MCP configuration from file
func (m *ConnectionManager) LoadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file is OK
		}
		return err
	}

	var config McpJsonConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse MCP config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, serverConfig := range config.MCPServers {
		scoped := ScopedMcpServerConfig{
			McpServerConfig: serverConfig,
			Scope:           ConfigScopeUser,
		}

		m.connections[name] = &MCPServerConnection{
			Name:   name,
			Type:   "pending",
			Config: scoped,
		}
	}

	return nil
}

// ConnectAll connects to all configured MCP servers
func (m *ConnectionManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	m.mu.RUnlock()

	var errors []error
	for _, name := range names {
		if err := m.Connect(ctx, name); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to connect to some MCP servers: %v", errors)
	}
	return nil
}

// Connect connects to a specific MCP server
func (m *ConnectionManager) Connect(ctx context.Context, name string) error {
	m.mu.RLock()
	conn, ok := m.connections[name]
	if !ok {
		m.mu.RUnlock()
		return fmt.Errorf("unknown MCP server: %s", name)
	}
	m.mu.RUnlock()

	client := NewClient(name, conn.Config)

	if err := client.Connect(ctx); err != nil {
		m.mu.Lock()
		m.connections[name] = &MCPServerConnection{
			Name:   name,
			Type:   "failed",
			Config: conn.Config,
			Error:  err.Error(),
		}
		m.mu.Unlock()
		return err
	}

	// Get tools from server
	tools, err := client.ListTools(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to list tools: %w", err)
	}

	m.mu.Lock()
	m.clients[name] = client
	m.connections[name] = &MCPServerConnection{
		Name:         name,
		Type:         "connected",
		Config:       conn.Config,
		Capabilities: client.GetCapabilities(),
		ServerInfo:   client.GetServerInfo(),
	}

	// Register tools with mcp__ prefix
	for _, tool := range tools {
		mcpToolName := "mcp__" + name + "__" + tool.Name
		tool.OriginalToolName = tool.Name
		tool.Name = mcpToolName
		m.tools[mcpToolName] = tool
	}
	m.mu.Unlock()

	return nil
}

// Disconnect disconnects from a specific MCP server
func (m *ConnectionManager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return nil
	}

	if err := client.Close(); err != nil {
		return err
	}

	delete(m.clients, name)

	if conn, ok := m.connections[name]; ok {
		conn.Type = "disabled"
	}

	// Remove tools
	for toolName, tool := range m.tools {
		if tool.IsMcp && utils.Contains([]string{name}, extractServerFromToolName(toolName)) {
			delete(m.tools, toolName)
		}
	}

	return nil
}

// DisconnectAll disconnects from all MCP servers
func (m *ConnectionManager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[string]*Client)
	m.tools = make(map[string]SerializedTool)
}

// GetClient returns the client for a server
func (m *ConnectionManager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// GetConnections returns all connections
func (m *ConnectionManager) GetConnections() map[string]*MCPServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*MCPServerConnection)
	for k, v := range m.connections {
		result[k] = v
	}
	return result
}

// GetTools returns all available MCP tools
func (m *ConnectionManager) GetTools() []SerializedTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]SerializedTool, 0, len(m.tools))
	for _, tool := range m.tools {
		result = append(result, tool)
	}
	return result
}

// CallTool calls an MCP tool
func (m *ConnectionManager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	serverName := extractServerFromToolName(toolName)
	if serverName == "" {
		return nil, fmt.Errorf("invalid MCP tool name: %s", toolName)
	}

	m.mu.RLock()
	client, ok := m.clients[serverName]
	originalName := ""
	if tool, ok := m.tools[toolName]; ok {
		originalName = tool.OriginalToolName
	}
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("MCP server not connected: %s", serverName)
	}

	return client.CallTool(ctx, originalName, args)
}

// AddServer adds a new MCP server configuration
func (m *ConnectionManager) AddServer(name string, config McpServerConfig, scope ConfigScope) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	scoped := ScopedMcpServerConfig{
		McpServerConfig: config,
		Scope:           scope,
	}

	m.connections[name] = &MCPServerConnection{
		Name:   name,
		Type:   "pending",
		Config: scoped,
	}

	return m.saveConfig()
}

// RemoveServer removes an MCP server configuration
func (m *ConnectionManager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[name]; ok {
		client.Close()
		delete(m.clients, name)
	}

	delete(m.connections, name)

	// Remove tools
	for toolName, tool := range m.tools {
		if tool.IsMcp && utils.Contains([]string{name}, extractServerFromToolName(toolName)) {
			delete(m.tools, toolName)
		}
	}

	return m.saveConfig()
}

// saveConfig saves the current configuration to file
func (m *ConnectionManager) saveConfig() error {
	config := McpJsonConfig{
		MCPServers: make(map[string]McpServerConfig),
	}

	for name, conn := range m.connections {
		config.MCPServers[name] = conn.Config.McpServerConfig
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// extractServerFromToolName extracts the server name from an MCP tool name.
// Uses McpInfoFromString for correct parsing (handles server names with single underscores).
func extractServerFromToolName(toolName string) string {
	info := McpInfoFromString(toolName)
	if info == nil {
		return ""
	}
	return info.ServerName
}
