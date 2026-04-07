package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// =============================================================================
// LSP Protocol Types
// =============================================================================

// LSPRequest represents an LSP request.
type LSPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// LSPResponse represents an LSP response.
type LSPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *LSPError       `json:"error,omitempty"`
}

// LSPError represents an LSP error.
type LSPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LSPNotification represents an LSP notification.
type LSPNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// LSPInitializeParams represents initialize request params.
type LSPInitializeParams struct {
	ProcessID             int                    `json:"processId"`
	ClientInfo            *LSPClientInfo         `json:"clientInfo"`
	Locale                string                 `json:"locale,omitempty"`
	RootPath              string                 `json:"rootPath,omitempty"`
	RootURI               string                 `json:"rootUri,omitempty"`
	InitializationOptions interface{}            `json:"initializationOptions,omitempty"`
	Capabilities          *LSPClientCapabilities `json:"capabilities"`
	Trace                 string                 `json:"trace,omitempty"`
	WorkspaceFolders      []LSPWorkspaceFolder   `json:"workspaceFolders,omitempty"`
}

// LSPClientInfo represents client information.
type LSPClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// LSPClientCapabilities represents client capabilities.
type LSPClientCapabilities struct {
	TextDocument *LSPTextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *LSPWorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// LSPTextDocumentClientCapabilities represents text document capabilities.
type LSPTextDocumentClientCapabilities struct {
	Completion     *LSPCompletionCapabilities     `json:"completion,omitempty"`
	Hover          *LSPHoverCapabilities          `json:"hover,omitempty"`
	Definition     *LSPDefinitionCapabilities     `json:"definition,omitempty"`
	References     *LSPReferencesCapabilities     `json:"references,omitempty"`
	DocumentSymbol *LSPDocumentSymbolCapabilities `json:"documentSymbol,omitempty"`
	CodeAction     *LSPCodeActionCapabilities     `json:"codeAction,omitempty"`
	Diagnostic     *LSPDiagnosticCapabilities     `json:"diagnostic,omitempty"`
}

// LSPCompletionCapabilities represents completion capabilities.
type LSPCompletionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	CompletionItem      struct {
		SnippetSupport bool `json:"snippetSupport"`
	} `json:"completionItem"`
}

// LSPHoverCapabilities represents hover capabilities.
type LSPHoverCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

// LSPDefinitionCapabilities represents definition capabilities.
type LSPDefinitionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	LinkSupport         bool `json:"linkSupport"`
}

// LSPReferencesCapabilities represents references capabilities.
type LSPReferencesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

// LSPDocumentSymbolCapabilities represents document symbol capabilities.
type LSPDocumentSymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

// LSPCodeActionCapabilities represents code action capabilities.
type LSPCodeActionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

// LSPDiagnosticCapabilities represents diagnostic capabilities.
type LSPDiagnosticCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

// LSPWorkspaceClientCapabilities represents workspace capabilities.
type LSPWorkspaceClientCapabilities struct {
	WorkspaceFolders bool `json:"workspaceFolders"`
}

// LSPWorkspaceFolder represents a workspace folder.
type LSPWorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// LSPInitializeResult represents the initialize result.
type LSPInitializeResult struct {
	Capabilities LSPServerCapabilities `json:"capabilities"`
	ServerInfo   *LSPServerInfo        `json:"serverInfo,omitempty"`
}

// LSPServerCapabilities represents server capabilities.
type LSPServerCapabilities struct {
	TextDocumentSync       interface{}               `json:"textDocumentSync,omitempty"`
	CompletionProvider     *LSPCompletionOptions     `json:"completionProvider,omitempty"`
	HoverProvider          interface{}               `json:"hoverProvider,omitempty"`
	DefinitionProvider     interface{}               `json:"definitionProvider,omitempty"`
	ReferencesProvider     interface{}               `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider interface{}               `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider     interface{}               `json:"codeActionProvider,omitempty"`
	DiagnosticProvider     *LSPDiagnosticOptions     `json:"diagnosticProvider,omitempty"`
	ExecuteCommandProvider *LSPExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
	Workspace              *LSPWorkspaceCapabilities `json:"workspace,omitempty"`
}

// LSPCompletionOptions represents completion options.
type LSPCompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider"`
}

// LSPDiagnosticOptions represents diagnostic options.
type LSPDiagnosticOptions struct {
	Identifier            string `json:"identifier,omitempty"`
	InterFileDependencies bool   `json:"interFileDependencies"`
}

// LSPExecuteCommandOptions represents execute command options.
type LSPExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// LSPWorkspaceCapabilities represents workspace capabilities.
type LSPWorkspaceCapabilities struct {
	WorkspaceFolders *LSPWorkspaceFoldersCapabilities `json:"workspaceFolders,omitempty"`
}

// LSPWorkspaceFoldersCapabilities represents workspace folders capabilities.
type LSPWorkspaceFoldersCapabilities struct {
	Supported           bool `json:"supported"`
	ChangeNotifications bool `json:"changeNotifications"`
}

// LSPServerInfo represents server information.
type LSPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// LSPTextDocumentItem represents a text document item.
type LSPTextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// LSPTextDocumentIdentifier represents a text document identifier.
type LSPTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// LSPPosition represents a position in a document.
type LSPPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// LSPTextDocumentPositionParams represents text document position params.
type LSPTextDocumentPositionParams struct {
	TextDocument LSPTextDocumentIdentifier `json:"textDocument"`
	Position     LSPPosition               `json:"position"`
}

// LSPCompletionParams represents completion params.
type LSPCompletionParams struct {
	TextDocument LSPTextDocumentIdentifier `json:"textDocument"`
	Position     LSPPosition               `json:"position"`
	Context      *LSPCompletionContext     `json:"context,omitempty"`
}

// LSPCompletionContext represents completion context.
type LSPCompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

// LSPCompletionItem represents a completion item.
type LSPCompletionItem struct {
	Label         string      `json:"label"`
	Kind          int         `json:"kind,omitempty"`
	Detail        string      `json:"detail,omitempty"`
	Documentation interface{} `json:"documentation,omitempty"`
	SortText      string      `json:"sortText,omitempty"`
	FilterText    string      `json:"filterText,omitempty"`
	InsertText    string      `json:"insertText,omitempty"`
	TextEdit      interface{} `json:"textEdit,omitempty"`
}

// LSPCompletionList represents a completion list.
type LSPCompletionList struct {
	IsIncomplete bool                `json:"isIncomplete"`
	Items        []LSPCompletionItem `json:"items"`
}

// LSPHover represents hover information.
type LSPHover struct {
	Contents interface{} `json:"contents"`
	Range    *LSPRange   `json:"range,omitempty"`
}

// LSPRange represents a range in a document.
type LSPRange struct {
	Start LSPPosition `json:"start"`
	End   LSPPosition `json:"end"`
}

// LSPLocation represents a location.
type LSPLocation struct {
	URI   string   `json:"uri"`
	Range LSPRange `json:"range"`
}

// LSPDiagnosticParams represents diagnostic params.
type LSPDiagnosticParams struct {
	TextDocument LSPTextDocumentIdentifier `json:"textDocument"`
}

// LSPDiagnosticNotification represents a diagnostic notification.
type LSPDiagnosticNotification struct {
	URI         string          `json:"uri"`
	Diagnostics []LSPDiagnostic `json:"diagnostics"`
}

// LSPDocumentSymbolParams represents document symbol params.
type LSPDocumentSymbolParams struct {
	TextDocument LSPTextDocumentIdentifier `json:"textDocument"`
}

// LSPDocumentSymbol represents a document symbol.
type LSPDocumentSymbol struct {
	Name           string              `json:"name"`
	Detail         string              `json:"detail,omitempty"`
	Kind           int                 `json:"kind"`
	Range          LSPRange            `json:"range"`
	SelectionRange LSPRange            `json:"selectionRange"`
	Children       []LSPDocumentSymbol `json:"children,omitempty"`
}

// =============================================================================
// LSP Client
// =============================================================================

// LSPClient represents an LSP client connection.
type LSPClient struct {
	mu           sync.RWMutex
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Reader
	stderr       io.Reader
	name         string
	idCounter    int
	pending      map[int]chan *LSPResponse
	capabilities *LSPServerCapabilities
	serverInfo   *LSPServerInfo
	initialized  bool
	diagnostics  map[string][]LSPDiagnostic
	handlers     *LSPHandlers
}

// LSPHandlers contains handlers for LSP notifications.
type LSPHandlers struct {
	OnDiagnostics func(uri string, diagnostics []LSPDiagnostic)
	OnLogMessage  func(message string, level int)
	OnShowMessage func(message string, type_ int)
}

// NewLSPClient creates a new LSP client.
func NewLSPClient(name string, command []string, cwd string) *LSPClient {
	return &LSPClient{
		name:        name,
		pending:     make(map[int]chan *LSPResponse),
		diagnostics: make(map[string][]LSPDiagnostic),
		cmd:         exec.Command(command[0], command[1:]...),
	}
}

// Start starts the LSP server process.
func (c *LSPClient) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create pipes
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

	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	c.stderr = stderr

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start LSP server: %w", err)
	}

	// Start reader goroutine
	go c.readLoop()

	// Start stderr reader
	go c.readStderr()

	return nil
}

// Initialize initializes the LSP connection.
func (c *LSPClient) Initialize(ctx context.Context, rootPath string) error {
	params := &LSPInitializeParams{
		ProcessID: os.Getpid(),
		ClientInfo: &LSPClientInfo{
			Name:    "claude-code-go",
			Version: "1.0.0",
		},
		RootPath: rootPath,
		RootURI:  "file://" + rootPath,
		Capabilities: &LSPClientCapabilities{
			TextDocument: &LSPTextDocumentClientCapabilities{
				Completion: &LSPCompletionCapabilities{
					DynamicRegistration: true,
				},
				Hover: &LSPHoverCapabilities{
					DynamicRegistration: true,
					ContentFormat:       []string{"markdown", "plaintext"},
				},
				Definition: &LSPDefinitionCapabilities{
					DynamicRegistration: true,
					LinkSupport:         true,
				},
				References: &LSPReferencesCapabilities{
					DynamicRegistration: true,
				},
				DocumentSymbol: &LSPDocumentSymbolCapabilities{
					DynamicRegistration: true,
				},
			},
			Workspace: &LSPWorkspaceClientCapabilities{
				WorkspaceFolders: true,
			},
		},
	}

	result, err := c.call(ctx, "initialize", params)
	if err != nil {
		return err
	}

	var initResult LSPInitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.capabilities = &initResult.Capabilities
	c.serverInfo = initResult.ServerInfo

	// Send initialized notification
	if err := c.notify("initialized", nil); err != nil {
		return err
	}

	c.initialized = true
	return nil
}

// Shutdown shuts down the LSP server.
func (c *LSPClient) Shutdown(ctx context.Context) error {
	if !c.initialized {
		return nil
	}

	_, err := c.call(ctx, "shutdown", nil)
	if err != nil {
		return err
	}

	if err := c.notify("exit", nil); err != nil {
		return err
	}

	if c.stdin != nil {
		c.stdin.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Wait()
	}

	c.initialized = false
	return nil
}

// OpenDocument opens a document in the LSP server.
func (c *LSPClient) OpenDocument(ctx context.Context, uri, languageID, content string) error {
	params := map[string]interface{}{
		"textDocument": LSPTextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}
	return c.notify("textDocument/didOpen", params)
}

// CloseDocument closes a document in the LSP server.
func (c *LSPClient) CloseDocument(ctx context.Context, uri string) error {
	params := map[string]interface{}{
		"textDocument": LSPTextDocumentIdentifier{URI: uri},
	}
	return c.notify("textDocument/didClose", params)
}

// ChangeDocument notifies about document changes.
func (c *LSPClient) ChangeDocument(ctx context.Context, uri string, version int, content string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]interface{}{
			{"text": content},
		},
	}
	return c.notify("textDocument/didChange", params)
}

// GetCompletions gets completions at a position.
func (c *LSPClient) GetCompletions(ctx context.Context, uri string, line, character int) (*LSPCompletionList, error) {
	params := LSPCompletionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: uri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := c.call(ctx, "textDocument/completion", params)
	if err != nil {
		return nil, err
	}

	var completions LSPCompletionList
	if err := json.Unmarshal(result, &completions); err != nil {
		// Try as array
		var items []LSPCompletionItem
		if err := json.Unmarshal(result, &items); err != nil {
			return nil, fmt.Errorf("failed to parse completion result: %w", err)
		}
		return &LSPCompletionList{Items: items}, nil
	}

	return &completions, nil
}

// GetHover gets hover information at a position.
func (c *LSPClient) GetHover(ctx context.Context, uri string, line, character int) (*LSPHover, error) {
	params := LSPTextDocumentPositionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: uri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := c.call(ctx, "textDocument/hover", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	var hover LSPHover
	if err := json.Unmarshal(result, &hover); err != nil {
		return nil, fmt.Errorf("failed to parse hover result: %w", err)
	}

	return &hover, nil
}

// GetDefinition gets the definition at a position.
func (c *LSPClient) GetDefinition(ctx context.Context, uri string, line, character int) ([]LSPLocation, error) {
	params := LSPTextDocumentPositionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: uri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := c.call(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	var locations []LSPLocation
	if err := json.Unmarshal(result, &locations); err != nil {
		// Try as single location
		var loc LSPLocation
		if err := json.Unmarshal(result, &loc); err != nil {
			return nil, fmt.Errorf("failed to parse definition result: %w", err)
		}
		return []LSPLocation{loc}, nil
	}

	return locations, nil
}

// GetReferences gets references at a position.
func (c *LSPClient) GetReferences(ctx context.Context, uri string, line, character int, includeDeclaration bool) ([]LSPLocation, error) {
	params := map[string]interface{}{
		"textDocument": LSPTextDocumentIdentifier{URI: uri},
		"position":     LSPPosition{Line: line, Character: character},
		"context": map[string]interface{}{
			"includeDeclaration": includeDeclaration,
		},
	}

	result, err := c.call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	var locations []LSPLocation
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, fmt.Errorf("failed to parse references result: %w", err)
	}

	return locations, nil
}

// GetDocumentSymbols gets symbols in a document.
func (c *LSPClient) GetDocumentSymbols(ctx context.Context, uri string) ([]LSPDocumentSymbol, error) {
	params := LSPDocumentSymbolParams{
		TextDocument: LSPTextDocumentIdentifier{URI: uri},
	}

	result, err := c.call(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	var symbols []LSPDocumentSymbol
	if err := json.Unmarshal(result, &symbols); err != nil {
		return nil, fmt.Errorf("failed to parse document symbols result: %w", err)
	}

	return symbols, nil
}

// GetDiagnostics gets diagnostics for a file.
func (c *LSPClient) GetDiagnostics(uri string) []LSPDiagnostic {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.diagnostics[uri]
}

// GetCapabilities returns server capabilities.
func (c *LSPClient) GetCapabilities() *LSPServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// GetServerInfo returns server info.
func (c *LSPClient) GetServerInfo() *LSPServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// SetHandlers sets notification handlers.
func (c *LSPClient) SetHandlers(handlers *LSPHandlers) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = handlers
}

// =============================================================================
// Internal Methods
// =============================================================================

func (c *LSPClient) nextID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idCounter++
	return c.idCounter
}

func (c *LSPClient) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID()
	req := &LSPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	respChan := make(chan *LSPResponse, 1)
	c.mu.Lock()
	c.pending[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.send(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("LSP error: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}
		return resp.Result, nil
	}
}

func (c *LSPClient) notify(method string, params interface{}) error {
	req := &LSPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.send(req)
}

func (c *LSPClient) send(req *LSPRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// LSP uses content-length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *LSPClient) readLoop() {
	for {
		// Read content-length header
		var contentLength int
		for {
			line, err := c.stdout.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length: ") {
				contentLength, _ = strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read body
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(c.stdout, body); err != nil {
			return
		}

		// Parse message
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      *int            `json:"id"`
			Method  string          `json:"method,omitempty"`
			Params  json.RawMessage `json:"params,omitempty"`
			Result  json.RawMessage `json:"result,omitempty"`
			Error   *LSPError       `json:"error,omitempty"`
		}

		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		if msg.ID != nil {
			// Response
			c.mu.Lock()
			if ch, ok := c.pending[*msg.ID]; ok {
				ch <- &LSPResponse{
					JSONRPC: msg.JSONRPC,
					ID:      *msg.ID,
					Result:  msg.Result,
					Error:   msg.Error,
				}
			}
			c.mu.Unlock()
		} else if msg.Method != "" {
			// Notification
			c.handleNotification(msg.Method, msg.Params)
		}
	}
}

func (c *LSPClient) readStderr() {
	if c.stderr == nil {
		return
	}
	buf := make([]byte, 1024)
	for {
		n, err := c.stderr.Read(buf)
		if err != nil {
			return
		}
		// Log stderr output
		fmt.Fprintf(os.Stderr, "[LSP %s] %s", c.name, string(buf[:n]))
	}
}

func (c *LSPClient) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		var notification LSPDiagnosticNotification
		if err := json.Unmarshal(params, &notification); err != nil {
			return
		}
		c.mu.Lock()
		c.diagnostics[notification.URI] = notification.Diagnostics
		c.mu.Unlock()

		if c.handlers != nil && c.handlers.OnDiagnostics != nil {
			c.handlers.OnDiagnostics(notification.URI, notification.Diagnostics)
		}

	case "window/logMessage":
		var msg struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &msg); err != nil {
			return
		}
		if c.handlers != nil && c.handlers.OnLogMessage != nil {
			c.handlers.OnLogMessage(msg.Message, msg.Type)
		}

	case "window/showMessage":
		var msg struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &msg); err != nil {
			return
		}
		if c.handlers != nil && c.handlers.OnShowMessage != nil {
			c.handlers.OnShowMessage(msg.Message, msg.Type)
		}
	}
}

// =============================================================================
// LSP Manager
// =============================================================================

// LSPManager manages multiple LSP servers.
type LSPManager struct {
	mu      sync.RWMutex
	clients map[string]*LSPClient
	configs map[string]LSPServerConfig
}

// NewLSPManager creates a new LSP manager.
func NewLSPManager() *LSPManager {
	return &LSPManager{
		clients: make(map[string]*LSPClient),
		configs: make(map[string]LSPServerConfig),
	}
}

// RegisterServer registers an LSP server configuration.
func (m *LSPManager) RegisterServer(config LSPServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.Name] = config
}

// StartServer starts an LSP server.
func (m *LSPManager) StartServer(ctx context.Context, name string, rootPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, ok := m.configs[name]
	if !ok {
		return fmt.Errorf("unknown LSP server: %s", name)
	}

	client := NewLSPClient(name, config.Command, rootPath)
	if err := client.Start(ctx); err != nil {
		return err
	}

	if err := client.Initialize(ctx, rootPath); err != nil {
		client.Shutdown(ctx)
		return err
	}

	m.clients[name] = client
	return nil
}

// StopServer stops an LSP server.
func (m *LSPManager) StopServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return nil
	}

	if err := client.Shutdown(ctx); err != nil {
		return err
	}

	delete(m.clients, name)
	return nil
}

// GetClient gets an LSP client by name.
func (m *LSPManager) GetClient(name string) (*LSPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// GetClientForFile gets the appropriate LSP client for a file.
func (m *LSPManager) GetClientForFile(filename string) (*LSPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext := getFileExtension(filename)
	for name, client := range m.clients {
		config := m.configs[name]
		for _, ft := range config.FileTypes {
			if ft == ext {
				return client, true
			}
		}
	}
	return nil, false
}

// OpenDocument opens a document in the appropriate LSP server.
func (m *LSPManager) OpenDocument(ctx context.Context, uri, content string) error {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil
	}
	languageID := getLanguageID(uri)
	return client.OpenDocument(ctx, uri, languageID, content)
}

// ChangeDocument notifies about document changes.
func (m *LSPManager) ChangeDocument(ctx context.Context, uri string, version int, content string) error {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil
	}
	return client.ChangeDocument(ctx, uri, version, content)
}

// CloseDocument closes a document.
func (m *LSPManager) CloseDocument(ctx context.Context, uri string) error {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil
	}
	return client.CloseDocument(ctx, uri)
}

// GetCompletions gets completions for a file.
func (m *LSPManager) GetCompletions(ctx context.Context, uri string, line, character int) (*LSPCompletionList, error) {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil, nil
	}
	return client.GetCompletions(ctx, uri, line, character)
}

// GetHover gets hover information for a file.
func (m *LSPManager) GetHover(ctx context.Context, uri string, line, character int) (*LSPHover, error) {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil, nil
	}
	return client.GetHover(ctx, uri, line, character)
}

// GetDefinition gets definition for a file.
func (m *LSPManager) GetDefinition(ctx context.Context, uri string, line, character int) ([]LSPLocation, error) {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil, nil
	}
	return client.GetDefinition(ctx, uri, line, character)
}

// GetReferences gets references for a file.
func (m *LSPManager) GetReferences(ctx context.Context, uri string, line, character int, includeDeclaration bool) ([]LSPLocation, error) {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil, nil
	}
	return client.GetReferences(ctx, uri, line, character, includeDeclaration)
}

// GetDiagnostics gets diagnostics for a file.
func (m *LSPManager) GetDiagnostics(uri string) []LSPDiagnostic {
	client, ok := m.GetClientForFile(uri)
	if !ok {
		return nil
	}
	return client.GetDiagnostics(uri)
}

// ShutdownAll shuts down all LSP servers.
func (m *LSPManager) ShutdownAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, client := range m.clients {
		if err := client.Shutdown(ctx); err != nil {
			lastErr = err
		}
		delete(m.clients, name)
	}
	return lastErr
}

// =============================================================================
// Helper Functions
// =============================================================================

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}
	return ""
}

func getLanguageID(filename string) string {
	ext := getFileExtension(filename)
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".jsx":
		return "javascriptreact"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	default:
		return ext[1:] // Remove the dot
	}
}

// =============================================================================
// Default LSP Server Configurations
// =============================================================================

// GetDefaultLSPConfigs returns default LSP server configurations.
func GetDefaultLSPConfigs() map[string]LSPServerConfig {
	return map[string]LSPServerConfig{
		"gopls": {
			Name:      "gopls",
			Command:   []string{"gopls"},
			FileTypes: []string{".go"},
		},
		"typescript-language-server": {
			Name:      "typescript-language-server",
			Command:   []string{"typescript-language-server", "--stdio"},
			FileTypes: []string{".ts", ".tsx", ".js", ".jsx"},
		},
		"pyright": {
			Name:      "pyright",
			Command:   []string{"pyright-langserver", "--stdio"},
			FileTypes: []string{".py"},
		},
		"rust-analyzer": {
			Name:      "rust-analyzer",
			Command:   []string{"rust-analyzer"},
			FileTypes: []string{".rs"},
		},
	}
}

// InitializeDefaultLSP initializes default LSP servers.
func InitializeDefaultLSP(ctx context.Context, rootPath string) (*LSPManager, error) {
	manager := NewLSPManager()

	for name, config := range GetDefaultLSPConfigs() {
		manager.RegisterServer(config)

		// Check if server is available
		if _, err := exec.LookPath(config.Command[0]); err != nil {
			continue
		}

		// Start the server
		if err := manager.StartServer(ctx, name, rootPath); err != nil {
			// Non-fatal: server might not be configured properly
			continue
		}
	}

	return manager, nil
}

// =============================================================================
// LSP Tool Integration
// =============================================================================

// LSPTool represents an LSP-based tool.
type LSPTool struct {
	manager *LSPManager
}

// NewLSPTool creates a new LSP tool.
func NewLSPTool(manager *LSPManager) *LSPTool {
	return &LSPTool{manager: manager}
}

// GetDefinitionAt returns the definition at a position.
func (t *LSPTool) GetDefinitionAt(ctx context.Context, uri string, line, character int) ([]LSPLocation, error) {
	return t.manager.GetDefinition(ctx, uri, line, character)
}

// GetHoverAt returns hover information at a position.
func (t *LSPTool) GetHoverAt(ctx context.Context, uri string, line, character int) (string, error) {
	hover, err := t.manager.GetHover(ctx, uri, line, character)
	if err != nil {
		return "", err
	}
	if hover == nil {
		return "", nil
	}

	switch v := hover.Contents.(type) {
	case string:
		return v, nil
	case []interface{}:
		var result string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if lang, ok := m["language"].(string); ok && lang != "" {
					result += fmt.Sprintf("```%s\n", lang)
				}
				if value, ok := m["value"].(string); ok {
					result += value
				}
				if _, ok := m["language"].(string); ok {
					result += "\n```\n"
				}
			} else if s, ok := item.(string); ok {
				result += s + "\n"
			}
		}
		return result, nil
	case map[string]interface{}:
		if value, ok := v["value"].(string); ok {
			return value, nil
		}
	}

	return "", nil
}

// GetCompletionsAt returns completions at a position.
func (t *LSPTool) GetCompletionsAt(ctx context.Context, uri string, line, character int) ([]string, error) {
	list, err := t.manager.GetCompletions(ctx, uri, line, character)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, nil
	}

	result := make([]string, len(list.Items))
	for i, item := range list.Items {
		result[i] = item.Label
	}
	return result, nil
}

// GetDiagnostics returns diagnostics for a file.
func (t *LSPTool) GetDiagnostics(uri string) []LSPDiagnostic {
	return t.manager.GetDiagnostics(uri)
}

// =============================================================================
// Global LSP Manager
// =============================================================================

var globalLSPManager *LSPManager
var lspManagerOnce sync.Once

// GetGlobalLSPManager returns the global LSP manager.
func GetGlobalLSPManager() *LSPManager {
	lspManagerOnce.Do(func() {
		globalLSPManager = NewLSPManager()
	})
	return globalLSPManager
}

// InitializeLSP initializes the global LSP manager.
func InitializeLSP(ctx context.Context, rootPath string) error {
	manager := GetGlobalLSPManager()

	for _, config := range GetDefaultLSPConfigs() {
		manager.RegisterServer(config)
	}

	return nil
}
