package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// LSPServerState represents the state of an LSP server
type LSPServerState string

const (
	LSPServerStateStarting LSPServerState = "starting"
	LSPServerStateRunning  LSPServerState = "running"
	LSPServerStateError    LSPServerState = "error"
	LSPServerStateStopped  LSPServerState = "stopped"
)

// LSPServerConfig holds configuration for an LSP server
type LSPServerConfig struct {
	Name      string   `json:"name"`
	Command   []string `json:"command"`
	FileTypes []string `json:"fileTypes"`
}

// LSPServer represents a running LSP server instance
type LSPServer struct {
	Name         string          `json:"name"`
	State        LSPServerState  `json:"state"`
	Config       LSPServerConfig `json:"config"`
	StartTime    time.Time       `json:"startTime"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
}

// LSPDiagnostic represents a diagnostic from an LSP server
type LSPDiagnostic struct {
	URI      string `json:"uri"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity int    `json:"severity"`
}

// LSPService manages LSP servers
type LSPService struct {
	mu      sync.RWMutex
	servers map[string]*LSPServer
	enabled bool
}

// NewLSPService creates a new LSP service
func NewLSPService() *LSPService {
	return &LSPService{
		servers: make(map[string]*LSPServer),
		enabled: true,
	}
}

// Initialize initializes the LSP service
func (s *LSPService) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers["gopls"] = &LSPServer{
		Name:  "gopls",
		State: LSPServerStateRunning,
		Config: LSPServerConfig{
			Name:      "gopls",
			Command:   []string{"gopls"},
			FileTypes: []string{".go"},
		},
		StartTime: time.Now(),
	}

	return nil
}

// GetServer gets an LSP server by name
func (s *LSPService) GetServer(name string) (*LSPServer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.servers[name], s.servers[name] != nil
}

// GetAllServers returns all LSP servers
func (s *LSPService) GetAllServers() map[string]*LSPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*LSPServer)
	for k, v := range s.servers {
		result[k] = v
	}
	return result
}

// IsConnected checks if any LSP server is connected
func (s *LSPService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, server := range s.servers {
		if server.State == LSPServerStateRunning {
			return true
		}
	}
	return false
}

// Shutdown shuts down all LSP servers
func (s *LSPService) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.servers = make(map[string]*LSPServer)
	return nil
}

// GetDiagnostics gets diagnostics for a file
func (s *LSPService) GetDiagnostics(uri string) []LSPDiagnostic {
	return nil
}

// GetCompletions gets completions at a position
func (s *LSPService) GetCompletions(ctx context.Context, uri string, line, col int) []string {
	return nil
}

// GetHover gets hover information at a position
func (s *LSPService) GetHover(ctx context.Context, uri string, line, col int) string {
	return ""
}

// Global instance
var lspServiceInstance *LSPService
var lspOnce sync.Once

// GetLSPService returns the global LSP service instance
func GetLSPService() *LSPService {
	lspOnce.Do(func() {
		lspServiceInstance = NewLSPService()
	})
	return lspServiceInstance
}
