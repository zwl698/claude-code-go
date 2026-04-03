package mcp

// =============================================================================
// MCP Types
// =============================================================================

// ConfigScope represents the scope of an MCP configuration.
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

// Transport represents the transport type for MCP.
type Transport string

const (
	TransportStdio  Transport = "stdio"
	TransportSSE    Transport = "sse"
	TransportSSEIDE Transport = "sse-ide"
	TransportHTTP   Transport = "http"
	TransportWS     Transport = "ws"
	TransportSDK    Transport = "sdk"
)

// =============================================================================
// Server Configuration Types
// =============================================================================

// McpServerConfig represents the base server configuration.
type McpServerConfig interface {
	GetType() Transport
}

// McpStdioServerConfig represents a stdio-based MCP server configuration.
type McpStdioServerConfig struct {
	Type    Transport         `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (c *McpStdioServerConfig) GetType() Transport { return TransportStdio }

// McpSSEServerConfig represents an SSE-based MCP server configuration.
type McpSSEServerConfig struct {
	Type          Transport         `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

func (c *McpSSEServerConfig) GetType() Transport { return TransportSSE }

// McpSSEIDEServerConfig represents an SSE IDE MCP server configuration.
type McpSSEIDEServerConfig struct {
	Type                Transport `json:"type"`
	URL                 string    `json:"url"`
	IDEName             string    `json:"ideName"`
	IDERunningInWindows bool      `json:"ideRunningInWindows,omitempty"`
}

func (c *McpSSEIDEServerConfig) GetType() Transport { return TransportSSEIDE }

// McpWebSocketIDEServerConfig represents a WebSocket IDE MCP server configuration.
type McpWebSocketIDEServerConfig struct {
	Type                Transport `json:"type"`
	URL                 string    `json:"url"`
	IDEName             string    `json:"ideName"`
	AuthToken           string    `json:"authToken,omitempty"`
	IDERunningInWindows bool      `json:"ideRunningInWindows,omitempty"`
}

func (c *McpWebSocketIDEServerConfig) GetType() Transport { return TransportWS }

// McpHTTPServerConfig represents an HTTP-based MCP server configuration.
type McpHTTPServerConfig struct {
	Type          Transport         `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

func (c *McpHTTPServerConfig) GetType() Transport { return TransportHTTP }

// McpWebSocketServerConfig represents a WebSocket-based MCP server configuration.
type McpWebSocketServerConfig struct {
	Type          Transport         `json:"type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
}

func (c *McpWebSocketServerConfig) GetType() Transport { return TransportWS }

// McpSdkServerConfig represents an SDK-based MCP server configuration.
type McpSdkServerConfig struct {
	Type Transport `json:"type"`
	Name string    `json:"name"`
}

func (c *McpSdkServerConfig) GetType() Transport { return TransportSDK }

// McpClaudeAIProxyServerConfig represents a Claude.ai proxy MCP server configuration.
type McpClaudeAIProxyServerConfig struct {
	Type Transport `json:"type"`
	URL  string    `json:"url"`
	ID   string    `json:"id"`
}

func (c *McpClaudeAIProxyServerConfig) GetType() Transport { return "claudeai-proxy" }

// McpOAuthConfig represents OAuth configuration for MCP.
type McpOAuthConfig struct {
	ClientID              string `json:"clientId,omitempty"`
	CallbackPort          int    `json:"callbackPort,omitempty"`
	AuthServerMetadataURL string `json:"authServerMetadataUrl,omitempty"`
	XAA                   bool   `json:"xaa,omitempty"`
}

// ScopedMcpServerConfig represents a scoped MCP server configuration.
type ScopedMcpServerConfig struct {
	McpServerConfig
	Scope        ConfigScope `json:"scope"`
	PluginSource string      `json:"pluginSource,omitempty"`
}

// =============================================================================
// Server Connection Types
// =============================================================================

// ServerConnectionType represents the connection state of a server.
type ServerConnectionType string

const (
	ServerConnectionConnected ServerConnectionType = "connected"
	ServerConnectionFailed    ServerConnectionType = "failed"
	ServerConnectionNeedsAuth ServerConnectionType = "needs-auth"
	ServerConnectionPending   ServerConnectionType = "pending"
	ServerConnectionDisabled  ServerConnectionType = "disabled"
)

// ConnectedMCPServer represents a connected MCP server.
type ConnectedMCPServer struct {
	Type         ServerConnectionType  `json:"type"`
	Name         string                `json:"name"`
	Capabilities ServerCapabilities    `json:"capabilities"`
	ServerInfo   *ServerInfo           `json:"serverInfo,omitempty"`
	Instructions string                `json:"instructions,omitempty"`
	Config       ScopedMcpServerConfig `json:"config"`
}

// FailedMCPServer represents a failed MCP server.
type FailedMCPServer struct {
	Type   ServerConnectionType  `json:"type"`
	Name   string                `json:"name"`
	Config ScopedMcpServerConfig `json:"config"`
	Error  string                `json:"error,omitempty"`
}

// NeedsAuthMCPServer represents an MCP server that needs authentication.
type NeedsAuthMCPServer struct {
	Type   ServerConnectionType  `json:"type"`
	Name   string                `json:"name"`
	Config ScopedMcpServerConfig `json:"config"`
}

// PendingMCPServer represents a pending MCP server.
type PendingMCPServer struct {
	Type                 ServerConnectionType  `json:"type"`
	Name                 string                `json:"name"`
	Config               ScopedMcpServerConfig `json:"config"`
	ReconnectAttempt     int                   `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts int                   `json:"maxReconnectAttempts,omitempty"`
}

// DisabledMCPServer represents a disabled MCP server.
type DisabledMCPServer struct {
	Type   ServerConnectionType  `json:"type"`
	Name   string                `json:"name"`
	Config ScopedMcpServerConfig `json:"config"`
}

// =============================================================================
// Capability Types
// =============================================================================

// ServerCapabilities represents the capabilities of an MCP server.
type ServerCapabilities struct {
	Tools     *ToolsCapabilities     `json:"tools,omitempty"`
	Resources *ResourcesCapabilities `json:"resources,omitempty"`
	Prompts   *PromptsCapabilities   `json:"prompts,omitempty"`
}

// ToolsCapabilities represents tool capabilities.
type ToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapabilities represents resource capabilities.
type ResourcesCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapabilities represents prompt capabilities.
type PromptsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo represents server information.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// =============================================================================
// Tool Types
// =============================================================================

// SerializedTool represents a serialized MCP tool.
type SerializedTool struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	InputJSONSchema  map[string]interface{} `json:"inputJSONSchema,omitempty"`
	IsMCP            bool                   `json:"isMcp,omitempty"`
	OriginalToolName string                 `json:"originalToolName,omitempty"`
}

// SerializedClient represents a serialized MCP client.
type SerializedClient struct {
	Name         string               `json:"name"`
	Type         ServerConnectionType `json:"type"`
	Capabilities *ServerCapabilities  `json:"capabilities,omitempty"`
}

// ServerResource represents a server resource.
type ServerResource struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
	Name   string `json:"name"`
}

// MCPCliState represents the state of the MCP CLI.
type MCPCliState struct {
	Clients         []SerializedClient               `json:"clients"`
	Configs         map[string]ScopedMcpServerConfig `json:"configs"`
	Tools           []SerializedTool                 `json:"tools"`
	Resources       map[string][]ServerResource      `json:"resources"`
	NormalizedNames map[string]string                `json:"normalizedNames,omitempty"`
}

// =============================================================================
// JSON Schema Types
// =============================================================================

// JSONSchema represents a JSON schema.
type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Items      *JSONSchema            `json:"items,omitempty"`
	Default    interface{}            `json:"default,omitempty"`
}

// ToolInputSchema represents the input schema for a tool.
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// =============================================================================
// MCPServerConnection - Union Type
// =============================================================================

// MCPServerConnection represents any server connection state.
// This is a union type that can be one of: ConnectedMCPServer, FailedMCPServer,
// NeedsAuthMCPServer, PendingMCPServer, or DisabledMCPServer.
type MCPServerConnection struct {
	// Type indicates which connection state this represents
	Type ServerConnectionType `json:"type"`

	// Common fields
	Name   string                `json:"name"`
	Config ScopedMcpServerConfig `json:"config"`

	// Fields for ConnectedMCPServer
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`
	ServerInfo   *ServerInfo         `json:"serverInfo,omitempty"`
	Instructions string              `json:"instructions,omitempty"`

	// Fields for FailedMCPServer
	Error string `json:"error,omitempty"`

	// Fields for PendingMCPServer
	ReconnectAttempt     int `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts int `json:"maxReconnectAttempts,omitempty"`
}

// IsConnected returns true if this is a connected server.
func (c *MCPServerConnection) IsConnected() bool {
	return c.Type == ServerConnectionConnected
}

// IsFailed returns true if this is a failed server.
func (c *MCPServerConnection) IsFailed() bool {
	return c.Type == ServerConnectionFailed
}

// NeedsAuth returns true if this server needs authentication.
func (c *MCPServerConnection) NeedsAuth() bool {
	return c.Type == ServerConnectionNeedsAuth
}

// IsPending returns true if this server is pending.
func (c *MCPServerConnection) IsPending() bool {
	return c.Type == ServerConnectionPending
}

// IsDisabled returns true if this server is disabled.
func (c *MCPServerConnection) IsDisabled() bool {
	return c.Type == ServerConnectionDisabled
}
