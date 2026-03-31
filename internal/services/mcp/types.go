package mcp

import (
	"encoding/json"
)

// ConfigScope represents where the configuration is defined
type ConfigScope string

const (
	ConfigScopeLocal      ConfigScope = "local"
	ConfigScopeUser       ConfigScope = "user"
	ConfigScopeProject    ConfigScope = "project"
	ConfigScopeDynamic    ConfigScope = "dynamic"
	ConfigScopeEnterprise ConfigScope = "enterprise"
	ConfigScopeClaudeAI   ConfigScope = "claudeai"
	ConfigScopeManaged    ConfigScope = "managed"
)

// Transport represents the MCP transport type
type Transport string

const (
	TransportStdio  Transport = "stdio"
	TransportSSE    Transport = "sse"
	TransportSSEIDE Transport = "sse-ide"
	TransportHTTP   Transport = "http"
	TransportWS     Transport = "ws"
	TransportSDK    Transport = "sdk"
)

// McpOAuthConfig represents OAuth configuration for MCP servers
type McpOAuthConfig struct {
	ClientID              string `json:"clientId,omitempty"`
	CallbackPort          int    `json:"callbackPort,omitempty"`
	AuthServerMetadataURL string `json:"authServerMetadataUrl,omitempty"`
	XAA                   bool   `json:"xaa,omitempty"`
}

// McpStdioServerConfig represents stdio MCP server configuration
type McpStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // Optional for backwards compatibility
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig represents SSE MCP server configuration
type McpSSEServerConfig struct {
	Type          string            `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

// McpSSEIDEServerConfig represents SSE IDE MCP server configuration
type McpSSEIDEServerConfig struct {
	Type                string `json:"type"`
	URL                 string `json:"url"`
	IDEName             string `json:"ideName"`
	IDERunningInWindows bool   `json:"ideRunningInWindows,omitempty"`
}

// McpWebSocketIDEServerConfig represents WebSocket IDE MCP server configuration
type McpWebSocketIDEServerConfig struct {
	Type                string `json:"type"`
	URL                 string `json:"url"`
	IDEName             string `json:"ideName"`
	AuthToken           string `json:"authToken,omitempty"`
	IDERunningInWindows bool   `json:"ideRunningInWindows,omitempty"`
}

// McpHTTPServerConfig represents HTTP MCP server configuration
type McpHTTPServerConfig struct {
	Type          string            `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

// McpWebSocketServerConfig represents WebSocket MCP server configuration
type McpWebSocketServerConfig struct {
	Type          string            `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
}

// McpSdkServerConfig represents SDK MCP server configuration
type McpSdkServerConfig struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// McpClaudeAIProxyServerConfig represents Claude.ai proxy MCP server configuration
type McpClaudeAIProxyServerConfig struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	ID   string `json:"id"`
}

// McpServerConfig is the union type for all MCP server configurations
type McpServerConfig struct {
	// Common fields
	Type string `json:"type"`

	// Stdio fields
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// SSE/HTTP/WS fields
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`

	// IDE fields
	IDEName             string `json:"ideName,omitempty"`
	AuthToken           string `json:"authToken,omitempty"`
	IDERunningInWindows bool   `json:"ideRunningInWindows,omitempty"`

	// SDK fields
	Name string `json:"name,omitempty"`

	// Claude.ai proxy fields
	ID string `json:"id,omitempty"`
}

// ScopedMcpServerConfig is an MCP server config with scope information
type ScopedMcpServerConfig struct {
	McpServerConfig
	Scope        ConfigScope `json:"scope"`
	PluginSource string      `json:"pluginSource,omitempty"`
}

// ServerInfo contains server identification information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities represents MCP server capabilities
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability represents tools capability
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability represents resources capability
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability represents prompts capability
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ConnectedMCPServer represents a connected MCP server
type ConnectedMCPServer struct {
	Name         string                `json:"name"`
	Type         string                `json:"type"`
	Capabilities *ServerCapabilities   `json:"capabilities,omitempty"`
	ServerInfo   *ServerInfo           `json:"serverInfo,omitempty"`
	Instructions string                `json:"instructions,omitempty"`
	Config       ScopedMcpServerConfig `json:"config"`
}

// FailedMCPServer represents a failed MCP server connection
type FailedMCPServer struct {
	Name   string                `json:"name"`
	Type   string                `json:"type"`
	Config ScopedMcpServerConfig `json:"config"`
	Error  string                `json:"error,omitempty"`
}

// NeedsAuthMCPServer represents an MCP server that needs authentication
type NeedsAuthMCPServer struct {
	Name   string                `json:"name"`
	Type   string                `json:"type"`
	Config ScopedMcpServerConfig `json:"config"`
}

// PendingMCPServer represents a pending MCP server connection
type PendingMCPServer struct {
	Name                 string                `json:"name"`
	Type                 string                `json:"type"`
	Config               ScopedMcpServerConfig `json:"config"`
	ReconnectAttempt     int                   `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts int                   `json:"maxReconnectAttempts,omitempty"`
}

// DisabledMCPServer represents a disabled MCP server
type DisabledMCPServer struct {
	Name   string                `json:"name"`
	Type   string                `json:"type"`
	Config ScopedMcpServerConfig `json:"config"`
}

// MCPServerConnection is the union type for all MCP server states
type MCPServerConnection struct {
	Name                 string                `json:"name"`
	Type                 string                `json:"type"` // connected, failed, needs-auth, pending, disabled
	Config               ScopedMcpServerConfig `json:"config"`
	Capabilities         *ServerCapabilities   `json:"capabilities,omitempty"`
	ServerInfo           *ServerInfo           `json:"serverInfo,omitempty"`
	Instructions         string                `json:"instructions,omitempty"`
	Error                string                `json:"error,omitempty"`
	ReconnectAttempt     int                   `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts int                   `json:"maxReconnectAttempts,omitempty"`
}

// SerializedTool represents a serialized MCP tool
type SerializedTool struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	InputJSONSchema  map[string]interface{} `json:"inputJSONSchema,omitempty"`
	IsMcp            bool                   `json:"isMcp,omitempty"`
	OriginalToolName string                 `json:"originalToolName,omitempty"`
}

// SerializedClient represents a serialized MCP client state
type SerializedClient struct {
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`
}

// ServerResource represents a resource from an MCP server
type ServerResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Server      string `json:"server"`
}

// MCPCliState represents the CLI state for MCP
type MCPCliState struct {
	Clients         []SerializedClient               `json:"clients"`
	Configs         map[string]ScopedMcpServerConfig `json:"configs"`
	Tools           []SerializedTool                 `json:"tools"`
	Resources       map[string][]ServerResource      `json:"resources"`
	NormalizedNames map[string]string                `json:"normalizedNames,omitempty"`
}

// McpJsonConfig represents the MCP configuration file format
type McpJsonConfig struct {
	MCPServers map[string]McpServerConfig `json:"mcpServers"`
}

// ParseMcpServerConfig parses a raw JSON message into an McpServerConfig
func ParseMcpServerConfig(data json.RawMessage) (*McpServerConfig, error) {
	var config McpServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
