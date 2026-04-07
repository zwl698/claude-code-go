package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// LSP Server Types
// =============================================================================

// LspServerState represents the state of an LSP server.
type LspServerState string

const (
	LspServerStateStopped  LspServerState = "stopped"
	LspServerStateStarting LspServerState = "starting"
	LspServerStateRunning  LspServerState = "running"
	LspServerStateStopping LspServerState = "stopping"
	LspServerStateError    LspServerState = "error"
)

// ScopedLspServerConfig represents a scoped LSP server configuration.
type ScopedLspServerConfig struct {
	Name                  string            `json:"name"`
	Command               string            `json:"command"`
	Args                  []string          `json:"args,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	WorkspaceFolder       string            `json:"workspaceFolder,omitempty"`
	ExtensionToLanguage   map[string]string `json:"extensionToLanguage"`
	InitializationOptions interface{}       `json:"initializationOptions,omitempty"`
	StartupTimeout        *time.Duration    `json:"startupTimeout,omitempty"`
	MaxRestarts           int               `json:"maxRestarts,omitempty"`
}

// LSPServerInstance represents a single LSP server instance.
type LSPServerInstance struct {
	mu                 sync.RWMutex
	name               string
	config             ScopedLspServerConfig
	state              LspServerState
	startTime          *time.Time
	lastError          error
	restartCount       int
	crashRecoveryCount int
	client             *LSPClient
	openedFiles        map[string]bool
}

// NewLSPServerInstance creates a new LSP server instance.
func NewLSPServerInstance(name string, config ScopedLspServerConfig) *LSPServerInstance {
	return &LSPServerInstance{
		name:        name,
		config:      config,
		state:       LspServerStateStopped,
		openedFiles: make(map[string]bool),
	}
}

// Name returns the server name.
func (s *LSPServerInstance) Name() string {
	return s.name
}

// Config returns the server configuration.
func (s *LSPServerInstance) Config() ScopedLspServerConfig {
	return s.config
}

// State returns the current server state.
func (s *LSPServerInstance) State() LspServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// StartTime returns when the server was last started.
func (s *LSPServerInstance) StartTime() *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

// LastError returns the last error encountered.
func (s *LSPServerInstance) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// RestartCount returns the number of manual restarts.
func (s *LSPServerInstance) RestartCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.restartCount
}

// Start starts the LSP server.
func (s *LSPServerInstance) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == LspServerStateRunning || s.state == LspServerStateStarting {
		return nil
	}

	// Cap crash-recovery attempts
	maxRestarts := s.config.MaxRestarts
	if maxRestarts == 0 {
		maxRestarts = 3
	}
	if s.state == LspServerStateError && s.crashRecoveryCount > maxRestarts {
		return fmt.Errorf("LSP server '%s' exceeded max crash recovery attempts (%d)", s.name, maxRestarts)
	}

	s.state = LspServerStateStarting

	// Build command args
	args := s.config.Args
	if args == nil {
		args = []string{}
	}

	// Create client
	cmd := append([]string{s.config.Command}, args...)
	s.client = NewLSPClient(s.name, cmd, s.config.WorkspaceFolder)

	// Set up crash handler
	s.client.SetHandlers(&LSPHandlers{
		OnDiagnostics: func(uri string, diagnostics []LSPDiagnostic) {
			// Handle diagnostics - could emit to event bus
		},
	})

	// Start the client
	if err := s.client.Start(ctx); err != nil {
		s.state = LspServerStateError
		s.lastError = err
		return fmt.Errorf("failed to start LSP server '%s': %w", s.name, err)
	}

	// Initialize
	rootPath := s.config.WorkspaceFolder
	if rootPath == "" {
		rootPath = "."
	}

	if err := s.client.Initialize(ctx, rootPath); err != nil {
		s.state = LspServerStateError
		s.lastError = err
		s.client.Shutdown(ctx)
		return fmt.Errorf("failed to initialize LSP server '%s': %w", s.name, err)
	}

	now := time.Now()
	s.startTime = &now
	s.state = LspServerStateRunning
	s.crashRecoveryCount = 0
	return nil
}

// Stop stops the LSP server.
func (s *LSPServerInstance) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == LspServerStateStopped || s.state == LspServerStateStopping {
		return nil
	}

	s.state = LspServerStateStopping

	if s.client != nil {
		if err := s.client.Shutdown(ctx); err != nil {
			s.state = LspServerStateError
			s.lastError = err
			return err
		}
	}

	s.state = LspServerStateStopped
	s.openedFiles = make(map[string]bool)
	return nil
}

// Restart restarts the LSP server.
func (s *LSPServerInstance) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop LSP server '%s' during restart: %w", s.name, err)
	}

	s.mu.Lock()
	s.restartCount++
	s.mu.Unlock()

	maxRestarts := s.config.MaxRestarts
	if maxRestarts == 0 {
		maxRestarts = 3
	}

	if s.restartCount > maxRestarts {
		return fmt.Errorf("max restart attempts (%d) exceeded for server '%s'", maxRestarts, s.name)
	}

	return s.Start(ctx)
}

// IsHealthy checks if the server is healthy.
func (s *LSPServerInstance) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == LspServerStateRunning && s.client != nil && s.client.initialized
}

// SendRequest sends a request to the server.
func (s *LSPServerInstance) SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error) {
	if !s.IsHealthy() {
		return nil, fmt.Errorf("LSP server '%s' is not healthy (state: %s)", s.name, s.state)
	}
	return s.client.call(ctx, method, params)
}

// SendNotification sends a notification to the server.
func (s *LSPServerInstance) SendNotification(method string, params interface{}) error {
	if !s.IsHealthy() {
		return fmt.Errorf("LSP server '%s' is not healthy (state: %s)", s.name, s.state)
	}
	return s.client.notify(method, params)
}

// MarkFileOpened marks a file as opened.
func (s *LSPServerInstance) MarkFileOpened(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.openedFiles[uri] = true
}

// MarkFileClosed marks a file as closed.
func (s *LSPServerInstance) MarkFileClosed(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openedFiles, uri)
}

// IsFileOpen checks if a file is opened.
func (s *LSPServerInstance) IsFileOpen(uri string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openedFiles[uri]
}

// =============================================================================
// LSP Server Manager
// =============================================================================

// LSPServerManager manages multiple LSP server instances.
type LSPServerManager struct {
	mu           sync.RWMutex
	servers      map[string]*LSPServerInstance
	extensionMap map[string][]string
	openedFiles  map[string]string
}

// NewLSPServerManager creates a new LSP server manager.
func NewLSPServerManager() *LSPServerManager {
	return &LSPServerManager{
		servers:      make(map[string]*LSPServerInstance),
		extensionMap: make(map[string][]string),
		openedFiles:  make(map[string]string),
	}
}

// RegisterServer registers an LSP server configuration.
func (m *LSPServerManager) RegisterServer(config ScopedLspServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate config
	if config.Command == "" {
		return fmt.Errorf("server '%s' missing required 'command' field", config.Name)
	}
	if len(config.ExtensionToLanguage) == 0 {
		return fmt.Errorf("server '%s' missing required 'extensionToLanguage' field", config.Name)
	}

	// Map file extensions to this server
	for ext := range config.ExtensionToLanguage {
		normalized := strings.ToLower(ext)
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		if m.extensionMap[normalized] == nil {
			m.extensionMap[normalized] = []string{}
		}
		m.extensionMap[normalized] = append(m.extensionMap[normalized], config.Name)
	}

	// Create server instance
	instance := NewLSPServerInstance(config.Name, config)
	m.servers[config.Name] = instance

	return nil
}

// Initialize initializes all registered servers.
func (m *LSPServerManager) Initialize(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, server := range m.servers {
		if err := server.Start(ctx); err != nil {
			// Log error but continue with other servers
			fmt.Printf("Warning: Failed to start LSP server '%s': %v\n", server.name, err)
		}
	}

	return nil
}

// Shutdown shuts down all servers.
func (m *LSPServerManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, server := range m.servers {
		if err := server.Stop(ctx); err != nil {
			lastErr = fmt.Errorf("failed to stop server '%s': %w", name, err)
		}
	}

	m.servers = make(map[string]*LSPServerInstance)
	m.extensionMap = make(map[string][]string)
	m.openedFiles = make(map[string]string)

	return lastErr
}

// GetServerForFile gets the appropriate server for a file.
func (m *LSPServerManager) GetServerForFile(filePath string) *LSPServerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(filePath))
	serverNames := m.extensionMap[ext]
	if len(serverNames) == 0 {
		return nil
	}

	return m.servers[serverNames[0]]
}

// EnsureServerStarted ensures the server for a file is started.
func (m *LSPServerManager) EnsureServerStarted(ctx context.Context, filePath string) (*LSPServerInstance, error) {
	server := m.GetServerForFile(filePath)
	if server == nil {
		return nil, nil
	}

	if server.State() == LspServerStateStopped || server.State() == LspServerStateError {
		if err := server.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start LSP server for file %s: %w", filePath, err)
		}
	}

	return server, nil
}

// SendRequest sends a request to the appropriate server.
func (m *LSPServerManager) SendRequest(ctx context.Context, filePath string, method string, params interface{}) (interface{}, error) {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}
	return server.SendRequest(ctx, method, params)
}

// OpenFile opens a file in the appropriate server.
func (m *LSPServerManager) OpenFile(ctx context.Context, filePath string, content string) error {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return err
	}
	if server == nil {
		return nil
	}

	fileUri := pathToFileURL(filePath)

	// Skip if already opened
	if server.IsFileOpen(fileUri) {
		return nil
	}

	// Get language ID
	ext := strings.ToLower(filepath.Ext(filePath))
	languageId := "plaintext"
	if lang, ok := server.config.ExtensionToLanguage[ext]; ok {
		languageId = lang
	}

	// Send didOpen notification
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileUri,
			"languageId": languageId,
			"version":    1,
			"text":       content,
		},
	}

	if err := server.SendNotification("textDocument/didOpen", params); err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	server.MarkFileOpened(fileUri)
	m.mu.Lock()
	m.openedFiles[fileUri] = server.name
	m.mu.Unlock()

	return nil
}

// ChangeFile notifies about file changes.
func (m *LSPServerManager) ChangeFile(ctx context.Context, filePath string, content string) error {
	server := m.GetServerForFile(filePath)
	if server == nil || server.State() != LspServerStateRunning {
		return m.OpenFile(ctx, filePath, content)
	}

	fileUri := pathToFileURL(filePath)

	// If not opened, open it first
	if !server.IsFileOpen(fileUri) {
		return m.OpenFile(ctx, filePath, content)
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     fileUri,
			"version": 1,
		},
		"contentChanges": []map[string]interface{}{
			{"text": content},
		},
	}

	return server.SendNotification("textDocument/didChange", params)
}

// SaveFile notifies about file save.
func (m *LSPServerManager) SaveFile(ctx context.Context, filePath string) error {
	server := m.GetServerForFile(filePath)
	if server == nil || server.State() != LspServerStateRunning {
		return nil
	}

	fileUri := pathToFileURL(filePath)
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileUri,
		},
	}

	return server.SendNotification("textDocument/didSave", params)
}

// CloseFile closes a file.
func (m *LSPServerManager) CloseFile(ctx context.Context, filePath string) error {
	server := m.GetServerForFile(filePath)
	if server == nil || server.State() != LspServerStateRunning {
		return nil
	}

	fileUri := pathToFileURL(filePath)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileUri,
		},
	}

	if err := server.SendNotification("textDocument/didClose", params); err != nil {
		return err
	}

	server.MarkFileClosed(fileUri)
	m.mu.Lock()
	delete(m.openedFiles, fileUri)
	m.mu.Unlock()

	return nil
}

// IsFileOpen checks if a file is opened.
func (m *LSPServerManager) IsFileOpen(filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fileUri := pathToFileURL(filePath)
	_, ok := m.openedFiles[fileUri]
	return ok
}

// GetAllServers returns all server instances.
func (m *LSPServerManager) GetAllServers() map[string]*LSPServerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*LSPServerInstance)
	for k, v := range m.servers {
		result[k] = v
	}
	return result
}

// GetDefinition gets the definition at a position.
func (m *LSPServerManager) GetDefinition(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error) {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	fileUri := pathToFileURL(filePath)
	params := LSPTextDocumentPositionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: fileUri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := server.SendRequest(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	// Parse result
	if result == nil {
		return nil, nil
	}

	// Handle both array and single location
	if arr, ok := result.([]interface{}); ok {
		locations := make([]LSPLocation, len(arr))
		for i, item := range arr {
			if locMap, ok := item.(map[string]interface{}); ok {
				locations[i] = parseLocation(locMap)
			}
		}
		return locations, nil
	}

	if locMap, ok := result.(map[string]interface{}); ok {
		return []LSPLocation{parseLocation(locMap)}, nil
	}

	return nil, nil
}

// GetHover gets hover information at a position.
func (m *LSPServerManager) GetHover(ctx context.Context, filePath string, line, character int) (*LSPHover, error) {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	fileUri := pathToFileURL(filePath)
	params := LSPTextDocumentPositionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: fileUri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := server.SendRequest(ctx, "textDocument/hover", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	// Parse hover result
	if hoverMap, ok := result.(map[string]interface{}); ok {
		hover := &LSPHover{}
		if contents, ok := hoverMap["contents"]; ok {
			hover.Contents = contents
		}
		if rangeMap, ok := hoverMap["range"].(map[string]interface{}); ok {
			parsedRange := parseRange(rangeMap)
			if parsedRange != nil {
				hover.Range = parsedRange
			}
		}
		return hover, nil
	}

	return nil, nil
}

// GetCompletions gets completions at a position.
func (m *LSPServerManager) GetCompletions(ctx context.Context, filePath string, line, character int) (*LSPCompletionList, error) {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	fileUri := pathToFileURL(filePath)
	params := LSPCompletionParams{
		TextDocument: LSPTextDocumentIdentifier{URI: fileUri},
		Position:     LSPPosition{Line: line, Character: character},
	}

	result, err := server.SendRequest(ctx, "textDocument/completion", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	// Parse completion result
	if listMap, ok := result.(map[string]interface{}); ok {
		list := &LSPCompletionList{}
		if isIncomplete, ok := listMap["isIncomplete"].(bool); ok {
			list.IsIncomplete = isIncomplete
		}
		if items, ok := listMap["items"].([]interface{}); ok {
			list.Items = make([]LSPCompletionItem, len(items))
			for i, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					list.Items[i] = parseCompletionItem(itemMap)
				}
			}
		}
		return list, nil
	}

	// Try as array of items
	if items, ok := result.([]interface{}); ok {
		list := &LSPCompletionList{}
		list.Items = make([]LSPCompletionItem, len(items))
		for i, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				list.Items[i] = parseCompletionItem(itemMap)
			}
		}
		return list, nil
	}

	return nil, nil
}

// GetReferences gets references at a position.
func (m *LSPServerManager) GetReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]LSPLocation, error) {
	server, err := m.EnsureServerStarted(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	fileUri := pathToFileURL(filePath)
	params := map[string]interface{}{
		"textDocument": LSPTextDocumentIdentifier{URI: fileUri},
		"position":     LSPPosition{Line: line, Character: character},
		"context": map[string]interface{}{
			"includeDeclaration": includeDeclaration,
		},
	}

	result, err := server.SendRequest(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	if arr, ok := result.([]interface{}); ok {
		locations := make([]LSPLocation, len(arr))
		for i, item := range arr {
			if locMap, ok := item.(map[string]interface{}); ok {
				locations[i] = parseLocation(locMap)
			}
		}
		return locations, nil
	}

	return nil, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func pathToFileURL(filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	return "file://" + absPath
}

func parseLocation(m map[string]interface{}) LSPLocation {
	loc := LSPLocation{}
	if uri, ok := m["uri"].(string); ok {
		loc.URI = uri
	}
	if rangeMap, ok := m["range"].(map[string]interface{}); ok {
		if r := parseRange(rangeMap); r != nil {
			loc.Range = *r
		}
	}
	return loc
}

func parseRange(m map[string]interface{}) *LSPRange {
	r := &LSPRange{}
	if start, ok := m["start"].(map[string]interface{}); ok {
		r.Start = parsePosition(start)
	}
	if end, ok := m["end"].(map[string]interface{}); ok {
		r.End = parsePosition(end)
	}
	return r
}

func parsePosition(m map[string]interface{}) LSPPosition {
	p := LSPPosition{}
	if line, ok := m["line"].(float64); ok {
		p.Line = int(line)
	}
	if char, ok := m["character"].(float64); ok {
		p.Character = int(char)
	}
	return p
}

func parseCompletionItem(m map[string]interface{}) LSPCompletionItem {
	item := LSPCompletionItem{}
	if label, ok := m["label"].(string); ok {
		item.Label = label
	}
	if kind, ok := m["kind"].(float64); ok {
		item.Kind = int(kind)
	}
	if detail, ok := m["detail"].(string); ok {
		item.Detail = detail
	}
	if sortText, ok := m["sortText"].(string); ok {
		item.SortText = sortText
	}
	if filterText, ok := m["filterText"].(string); ok {
		item.FilterText = filterText
	}
	if insertText, ok := m["insertText"].(string); ok {
		item.InsertText = insertText
	}
	return item
}

// =============================================================================
// Global LSP Server Manager
// =============================================================================

var globalLSPServerManager *LSPServerManager
var lspServerManagerOnce sync.Once

// GetGlobalLSPServerManager returns the global LSP server manager.
func GetGlobalLSPServerManager() *LSPServerManager {
	lspServerManagerOnce.Do(func() {
		globalLSPServerManager = NewLSPServerManager()
	})
	return globalLSPServerManager
}

// RegisterDefaultLSPServers registers default LSP server configurations.
func RegisterDefaultLSPServers() error {
	manager := GetGlobalLSPServerManager()

	defaultConfigs := []ScopedLspServerConfig{
		{
			Name:    "gopls",
			Command: "gopls",
			ExtensionToLanguage: map[string]string{
				".go": "go",
			},
		},
		{
			Name:    "typescript-language-server",
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			ExtensionToLanguage: map[string]string{
				".ts":  "typescript",
				".tsx": "typescriptreact",
				".js":  "javascript",
				".jsx": "javascriptreact",
			},
		},
		{
			Name:    "pyright",
			Command: "pyright-langserver",
			Args:    []string{"--stdio"},
			ExtensionToLanguage: map[string]string{
				".py": "python",
			},
		},
		{
			Name:    "rust-analyzer",
			Command: "rust-analyzer",
			ExtensionToLanguage: map[string]string{
				".rs": "rust",
			},
		},
	}

	for _, config := range defaultConfigs {
		if err := manager.RegisterServer(config); err != nil {
			fmt.Printf("Warning: Failed to register LSP server '%s': %v\n", config.Name, err)
		}
	}

	return nil
}
