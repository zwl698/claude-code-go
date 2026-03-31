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
)

// Client represents an MCP client connection
type Client struct {
	mu           sync.RWMutex
	name         string
	config       ScopedMcpServerConfig
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Reader
	connected    bool
	capabilities *ServerCapabilities
	serverInfo   *ServerInfo
	requestID    int64
	pending      map[int64]chan json.RawMessage
	done         chan struct{}
}

// NewClient creates a new MCP client
func NewClient(name string, config ScopedMcpServerConfig) *Client {
	return &Client{
		name:    name,
		config:  config,
		pending: make(map[int64]chan json.RawMessage),
		done:    make(chan struct{}),
	}
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	c.cmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	if c.config.Env != nil {
		c.cmd.Env = os.Environ()
		for k, v := range c.config.Env {
			c.cmd.Env = append(c.cmd.Env, k+"="+v)
		}
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	go c.readResponses()

	if err := c.initialize(ctx); err != nil {
		c.cmd.Process.Kill()
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	c.connected = true
	return nil
}

func (c *Client) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "claude-code-go",
			"version": "1.0.0",
		},
	}

	result, err := c.sendRequest(ctx, "initialize", params)
	if err != nil {
		return err
	}

	var initResult struct {
		Capabilities *ServerCapabilities `json:"capabilities"`
		ServerInfo   *ServerInfo         `json:"serverInfo"`
	}
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.capabilities = initResult.Capabilities
	c.serverInfo = initResult.ServerInfo
	return c.sendNotification("notifications/initialized", nil)
}

// readResponses reads JSON-RPC responses from the server
func (c *Client) readResponses() {
	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return
		}

		var msg json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		var response struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(line, &response); err != nil {
			continue
		}

		c.mu.RLock()
		ch, ok := c.pending[response.ID]
		c.mu.RUnlock()

		if ok {
			ch <- response.Result
		}
	}
}

func (c *Client) nextRequestID() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestID++
	return c.requestID
}

func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextRequestID()
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	responseCh := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = responseCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-responseCh:
		return result, nil
	}
}

func (c *Client) sendNotification(method string, params interface{}) error {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

// ListTools returns the list of available tools from the server
func (c *Client) ListTools(ctx context.Context) ([]SerializedTool, error) {
	result, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Tools []SerializedTool `json:"tools"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	for i := range response.Tools {
		response.Tools[i].IsMcp = true
	}

	return response.Tools, nil
}

// CallTool invokes a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	if response.IsError {
		return nil, fmt.Errorf("tool error: %s", response.Content[0].Text)
	}

	if len(response.Content) > 0 {
		return response.Content[0].Text, nil
	}
	return nil, nil
}

// Close closes the connection to the MCP server
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	close(c.done)
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetName returns the client name
func (c *Client) GetName() string {
	return c.name
}

// GetCapabilities returns the server capabilities
func (c *Client) GetCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// GetServerInfo returns the server info
func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}
