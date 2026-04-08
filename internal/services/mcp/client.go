package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// =============================================================================
// MCP Client
// =============================================================================

// Client represents an MCP client.
type Client struct {
	name                string
	config              McpServerConfig
	connected           bool
	cmd                 *exec.Cmd
	stdin               io.WriteCloser
	stdout              io.Reader
	stderr              io.Reader
	mu                  sync.Mutex
	requestID           int64
	handlers            map[int64]chan *Response
	notificationHandler func(method string, params interface{})
	capabilities        *ServerCapabilities
	serverInfo          *ServerInfo
}

// NewClient creates a new MCP client.
func NewClient(name string, config McpServerConfig) *Client {
	return &Client{
		name:     name,
		config:   config,
		handlers: make(map[int64]chan *Response),
	}
}

// Connect connects to the MCP server.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Handle different transport types
	switch cfg := c.config.(type) {
	case *McpStdioServerConfig:
		return c.connectStdio(ctx, cfg)
	case *McpHTTPServerConfig:
		return c.connectHTTP(ctx, cfg)
	case *McpWebSocketServerConfig:
		return c.connectWebSocket(ctx, cfg)
	default:
		return fmt.Errorf("unsupported transport type: %T", cfg)
	}
}

// connectStdio connects to a stdio-based MCP server.
func (c *Client) connectStdio(ctx context.Context, cfg *McpStdioServerConfig) error {
	// Create command
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr

	// Start reading responses
	go c.readResponses()
	go c.readStderr()

	// Initialize connection
	if err := c.initialize(ctx); err != nil {
		c.Disconnect()
		return err
	}

	c.connected = true
	return nil
}

// connectHTTP connects to an HTTP-based MCP server.
func (c *Client) connectHTTP(ctx context.Context, cfg *McpHTTPServerConfig) error {
	// HTTP transport implementation
	// This would use net/http for SSE/streamable HTTP
	c.connected = true
	return nil
}

// connectWebSocket connects to a WebSocket-based MCP server.
func (c *Client) connectWebSocket(ctx context.Context, cfg *McpWebSocketServerConfig) error {
	// WebSocket transport implementation
	// This would use gorilla/websocket or similar
	c.connected = true
	return nil
}

// Disconnect disconnects from the MCP server.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// Close stdin
	if c.stdin != nil {
		c.stdin.Close()
	}

	// Kill process
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	c.connected = false
	return nil
}

// =============================================================================
// JSON-RPC Types
// =============================================================================

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents a JSON-RPC notification.
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// =============================================================================
// JSON-RPC Methods
// =============================================================================

// Call makes a JSON-RPC call.
func (c *Client) Call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	c.mu.Unlock()

	// Create request
	req := &Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Create response channel
	respChan := make(chan *Response, 1)
	c.mu.Lock()
	c.handlers[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()
	}()

	// Send request
	if err := c.sendRequest(req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timed out")
	}
}

// sendRequest sends a JSON-RPC request.
func (c *Client) sendRequest(req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin == nil {
		return fmt.Errorf("not connected")
	}

	_, err = c.stdin.Write(data)
	return err
}

// readResponses reads responses from the server.
func (c *Client) readResponses() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		// Check if this is a response or notification
		if resp.ID != 0 {
			c.mu.Lock()
			handler, ok := c.handlers[resp.ID]
			c.mu.Unlock()

			if ok {
				handler <- &resp
			}
		} else {
			// This is a notification
			var notif Notification
			if err := json.Unmarshal(line, &notif); err == nil {
				c.mu.Lock()
				handler := c.notificationHandler
				c.mu.Unlock()

				if handler != nil {
					handler(notif.Method, notif.Params)
				}
			}
		}
	}
}

// readStderr reads stderr output.
func (c *Client) readStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		// Log stderr output
		fmt.Fprintf(os.Stderr, "[MCP %s stderr] %s\n", c.name, scanner.Text())
	}
}

// SetNotificationHandler sets the notification handler.
func (c *Client) SetNotificationHandler(handler func(method string, params interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notificationHandler = handler
}

// GetCapabilities returns the server capabilities.
func (c *Client) GetCapabilities() *ServerCapabilities {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.capabilities
}

// GetServerInfo returns the server info.
func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.serverInfo
}

// =============================================================================
// MCP Protocol Methods
// =============================================================================

// initialize initializes the MCP connection.
func (c *Client) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]interface{}{
			"name":    "claude-code-go",
			"version": "1.0.0",
		},
		"capabilities": map[string]interface{}{},
	}

	result, err := c.Call(ctx, "initialize", params)
	if err != nil {
		return err
	}

	// Parse capabilities and server info
	if resultMap, ok := result.(map[string]interface{}); ok {
		// Parse server info
		if serverInfo, ok := resultMap["serverInfo"].(map[string]interface{}); ok {
			c.serverInfo = &ServerInfo{
				Name:    getString(serverInfo, "name"),
				Version: getString(serverInfo, "version"),
			}
		}

		// Parse capabilities
		if caps, ok := resultMap["capabilities"].(map[string]interface{}); ok {
			c.capabilities = &ServerCapabilities{
				Tools:     parseToolsCapability(caps),
				Resources: parseResourcesCapability(caps),
				Prompts:   parsePromptsCapability(caps),
			}
		}
	}

	// Send initialized notification
	return c.sendRequest(&Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
}

// ListTools lists available tools.
func (c *Client) ListTools(ctx context.Context) ([]SerializedTool, error) {
	result, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	// Parse result
	var tools []SerializedTool
	if resultMap, ok := result.(map[string]interface{}); ok {
		if toolsArray, ok := resultMap["tools"].([]interface{}); ok {
			for _, t := range toolsArray {
				if toolMap, ok := t.(map[string]interface{}); ok {
					tool := SerializedTool{
						Name:        getString(toolMap, "name"),
						Description: getString(toolMap, "description"),
						IsMCP:       true,
					}
					if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
						tool.InputJSONSchema = schema
					}
					tools = append(tools, tool)
				}
			}
		}
	}

	return tools, nil
}

// CallTool calls a tool.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	return c.Call(ctx, "tools/call", params)
}

// ListResources lists available resources.
func (c *Client) ListResources(ctx context.Context) ([]ServerResource, error) {
	result, err := c.Call(ctx, "resources/list", nil)
	if err != nil {
		return nil, err
	}

	var resources []ServerResource
	if resultMap, ok := result.(map[string]interface{}); ok {
		if resArray, ok := resultMap["resources"].([]interface{}); ok {
			for _, r := range resArray {
				if resMap, ok := r.(map[string]interface{}); ok {
					resources = append(resources, ServerResource{
						URI:  getString(resMap, "uri"),
						Name: getString(resMap, "name"),
					})
				}
			}
		}
	}

	return resources, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// parseToolsCapability parses the tools capability from server capabilities.
func parseToolsCapability(caps map[string]interface{}) *ToolsCapabilities {
	tools, ok := caps["tools"].(map[string]interface{})
	if !ok {
		return nil
	}

	capability := &ToolsCapabilities{}
	if listChanged, ok := tools["listChanged"].(bool); ok {
		capability.ListChanged = listChanged
	}

	return capability
}

// parseResourcesCapability parses the resources capability from server capabilities.
func parseResourcesCapability(caps map[string]interface{}) *ResourcesCapabilities {
	resources, ok := caps["resources"].(map[string]interface{})
	if !ok {
		return nil
	}

	capability := &ResourcesCapabilities{}
	if subscribe, ok := resources["subscribe"].(bool); ok {
		capability.Subscribe = subscribe
	}
	if listChanged, ok := resources["listChanged"].(bool); ok {
		capability.ListChanged = listChanged
	}

	return capability
}

// parsePromptsCapability parses the prompts capability from server capabilities.
func parsePromptsCapability(caps map[string]interface{}) *PromptsCapabilities {
	prompts, ok := caps["prompts"].(map[string]interface{})
	if !ok {
		return nil
	}

	capability := &PromptsCapabilities{}
	if listChanged, ok := prompts["listChanged"].(bool); ok {
		capability.ListChanged = listChanged
	}

	return capability
}
