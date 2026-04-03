package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Tool represents a tool interface for filtering
type Tool interface {
	GetName() string
	IsMcp() bool
}

// Command represents a command interface for filtering
type Command interface {
	GetName() string
	GetType() string
	GetLoadedFrom() string
	IsMcp() bool
}

// FilterToolsByServer filters tools by MCP server name.
func FilterToolsByServer(tools []Tool, serverName string) []Tool {
	prefix := "mcp__" + NormalizeNameForMCP(serverName) + "__"
	var result []Tool
	for _, tool := range tools {
		if strings.HasPrefix(tool.GetName(), prefix) {
			result = append(result, tool)
		}
	}
	return result
}

// CommandBelongsToServer checks if a command belongs to the given MCP server.
// MCP prompts are named `mcp__<server>__<prompt>` (wire-format constraint);
// MCP skills are named `<server>:<skill>` (matching plugin/nested-dir skill naming).
func CommandBelongsToServer(command Command, serverName string) bool {
	normalized := NormalizeNameForMCP(serverName)
	name := command.GetName()
	if name == "" {
		return false
	}
	return strings.HasPrefix(name, "mcp__"+normalized+"__") || strings.HasPrefix(name, normalized+":")
}

// FilterCommandsByServer filters commands by MCP server name.
func FilterCommandsByServer(commands []Command, serverName string) []Command {
	var result []Command
	for _, c := range commands {
		if CommandBelongsToServer(c, serverName) {
			result = append(result, c)
		}
	}
	return result
}

// FilterMcpPromptsByServer filters MCP prompts (not skills) by server.
// Used by the /mcp menu capabilities display — skills are a separate feature shown in /skills,
// so they mustn't inflate the "prompts" capability badge.
// The distinguisher is `loadedFrom === 'mcp'`: MCP skills set it, MCP prompts don't.
func FilterMcpPromptsByServer(commands []Command, serverName string) []Command {
	var result []Command
	for _, c := range commands {
		if CommandBelongsToServer(c, serverName) &&
			!(c.GetType() == "prompt" && c.GetLoadedFrom() == "mcp") {
			result = append(result, c)
		}
	}
	return result
}

// FilterResourcesByServer filters resources by MCP server name.
func FilterResourcesByServer(resources []ServerResource, serverName string) []ServerResource {
	var result []ServerResource
	for _, resource := range resources {
		if resource.Server == serverName {
			result = append(result, resource)
		}
	}
	return result
}

// ExcludeToolsByServer removes tools belonging to a specific MCP server.
func ExcludeToolsByServer(tools []Tool, serverName string) []Tool {
	prefix := "mcp__" + NormalizeNameForMCP(serverName) + "__"
	var result []Tool
	for _, tool := range tools {
		if !strings.HasPrefix(tool.GetName(), prefix) {
			result = append(result, tool)
		}
	}
	return result
}

// ExcludeCommandsByServer removes commands belonging to a specific MCP server.
func ExcludeCommandsByServer(commands []Command, serverName string) []Command {
	var result []Command
	for _, c := range commands {
		if !CommandBelongsToServer(c, serverName) {
			result = append(result, c)
		}
	}
	return result
}

// ExcludeResourcesByServer removes resources belonging to a specific MCP server.
func ExcludeResourcesByServer(resources map[string][]ServerResource, serverName string) map[string][]ServerResource {
	result := make(map[string][]ServerResource)
	for k, v := range resources {
		if k != serverName {
			result[k] = v
		}
	}
	return result
}

// HashMcpConfig computes a stable hash of an MCP server config for change detection.
// Excludes `scope` (provenance, not content — moving a server from .mcp.json
// to settings.json shouldn't reconnect it). Keys sorted so `{a:1,b:2}` and
// `{b:2,a:1}` hash the same.
func HashMcpConfig(config ScopedMcpServerConfig) string {
	// Create a copy without scope for hashing
	configCopy := config.McpServerConfig

	// Marshal with sorted keys
	stable := sortedJSONMarshal(configCopy)

	hash := sha256.Sum256([]byte(stable))
	return hex.EncodeToString(hash[:])[:16]
}

// sortedJSONMarshal marshals a value with sorted keys
func sortedJSONMarshal(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// sortedObjectKeys returns a JSON object with sorted keys
func sortedObjectKeys(obj map[string]interface{}) map[string]interface{} {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make(map[string]interface{})
	for _, k := range keys {
		result[k] = obj[k]
	}
	return result
}

// ExcludeStalePluginClients removes stale MCP clients and their tools/commands/resources.
// A client is stale if:
//   - scope 'dynamic' and name no longer in configs (plugin disabled), or
//   - config hash changed (args/url/env edited in .mcp.json) — any scope
//
// Returns the stale clients so the caller can disconnect them.
func ExcludeStalePluginClients(
	clients []MCPServerConnection,
	tools []Tool,
	commands []Command,
	resources map[string][]ServerResource,
	configs map[string]ScopedMcpServerConfig,
) (filteredClients []MCPServerConnection, filteredTools []Tool, filteredCommands []Command, filteredResources map[string][]ServerResource, stale []MCPServerConnection) {
	for _, c := range clients {
		fresh, exists := configs[c.Name]
		isStale := false

		if !exists {
			isStale = c.Config.Scope == ConfigScopeDynamic
		} else {
			isStale = HashMcpConfig(c.Config) != HashMcpConfig(fresh)
		}

		if isStale {
			stale = append(stale, c)
		}
	}

	if len(stale) == 0 {
		return clients, tools, commands, resources, nil
	}

	// Build set of stale names
	staleNames := make(map[string]bool)
	for _, s := range stale {
		staleNames[s.Name] = true
	}

	// Filter out stale tools, commands, resources
	filteredTools = tools
	filteredCommands = commands
	filteredResources = resources

	for _, s := range stale {
		filteredTools = ExcludeToolsByServer(filteredTools, s.Name)
		filteredCommands = ExcludeCommandsByServer(filteredCommands, s.Name)
		filteredResources = ExcludeResourcesByServer(filteredResources, s.Name)
	}

	// Filter clients
	for _, c := range clients {
		if !staleNames[c.Name] {
			filteredClients = append(filteredClients, c)
		}
	}

	return filteredClients, filteredTools, filteredCommands, filteredResources, stale
}

// IsToolFromMcpServer checks if a tool name belongs to a specific MCP server.
func IsToolFromMcpServer(toolName, serverName string) bool {
	info := McpInfoFromString(toolName)
	return info != nil && info.ServerName == serverName
}

// IsMcpTool checks if a tool belongs to any MCP server.
func IsMcpTool(tool Tool) bool {
	name := tool.GetName()
	return strings.HasPrefix(name, "mcp__") || tool.IsMcp()
}

// IsMcpCommand checks if a command belongs to any MCP server.
func IsMcpCommand(command Command) bool {
	name := command.GetName()
	return strings.HasPrefix(name, "mcp__") || command.IsMcp()
}

// DescribeMcpConfigFilePath describes the file path for a given MCP config scope.
func DescribeMcpConfigFilePath(scope ConfigScope, cwd, globalClaudeFile, enterpriseMcpFilePath string) string {
	switch scope {
	case ConfigScopeUser:
		return globalClaudeFile
	case ConfigScopeProject:
		return cwd + "/.mcp.json"
	case ConfigScopeLocal:
		return globalClaudeFile + " [project: " + cwd + "]"
	case ConfigScopeDynamic:
		return "Dynamically configured"
	case ConfigScopeEnterprise:
		return enterpriseMcpFilePath
	case ConfigScopeClaudeAI:
		return "claude.ai"
	default:
		return string(scope)
	}
}

// GetScopeLabel returns a human-readable label for a config scope.
func GetScopeLabel(scope ConfigScope) string {
	switch scope {
	case ConfigScopeLocal:
		return "Local config (private to you in this project)"
	case ConfigScopeProject:
		return "Project config (shared via .mcp.json)"
	case ConfigScopeUser:
		return "User config (available in all your projects)"
	case ConfigScopeDynamic:
		return "Dynamic config (from command line)"
	case ConfigScopeEnterprise:
		return "Enterprise config (managed by your organization)"
	case ConfigScopeClaudeAI:
		return "claude.ai config"
	default:
		return string(scope)
	}
}

// ValidConfigScopes returns all valid config scope values
func ValidConfigScopes() []ConfigScope {
	return []ConfigScope{
		ConfigScopeLocal,
		ConfigScopeUser,
		ConfigScopeProject,
		ConfigScopeDynamic,
		ConfigScopeEnterprise,
		ConfigScopeClaudeAI,
		ConfigScopeManaged,
	}
}

// EnsureConfigScope validates and returns a valid ConfigScope.
// Returns an error if the scope is invalid (matching TypeScript behavior).
func EnsureConfigScope(scope string) (ConfigScope, error) {
	if scope == "" {
		return ConfigScopeLocal, nil
	}

	for _, s := range ValidConfigScopes() {
		if string(s) == scope {
			return s, nil
		}
	}

	return ConfigScopeLocal, fmt.Errorf("invalid scope: %s. Must be one of: %s",
		scope, strings.Join(scopesToStrings(ValidConfigScopes()), ", "))
}

// scopesToStrings converts config scopes to strings
func scopesToStrings(scopes []ConfigScope) []string {
	result := make([]string, len(scopes))
	for i, s := range scopes {
		result[i] = string(s)
	}
	return result
}

// ValidTransports returns all valid transport type values
func ValidTransports() []Transport {
	return []Transport{
		TransportStdio,
		TransportSSE,
		TransportSSEIDE,
		TransportHTTP,
		TransportWS,
		TransportSDK,
	}
}

// EnsureTransport validates and returns a valid transport type.
// Returns an error if the transport type is invalid (matching TypeScript behavior).
func EnsureTransport(transportType string) (Transport, error) {
	if transportType == "" {
		return TransportStdio, nil
	}

	for _, t := range ValidTransports() {
		if string(t) == transportType {
			return t, nil
		}
	}

	return TransportStdio, fmt.Errorf("invalid transport type: %s. Must be one of: stdio, sse, http",
		transportType)
}

// ParseHeaders parses an array of header strings into a map.
// Each header should be in the format "Header-Name: value".
func ParseHeaders(headerArray []string) (map[string]string, error) {
	headers := make(map[string]string)

	for _, header := range headerArray {
		colonIndex := strings.Index(header, ":")
		if colonIndex == -1 {
			return nil, &InvalidHeaderError{Header: header, Reason: "expected format 'Header-Name: value'"}
		}

		key := strings.TrimSpace(header[:colonIndex])
		value := strings.TrimSpace(header[colonIndex+1:])

		if key == "" {
			return nil, &InvalidHeaderError{Header: header, Reason: "header name cannot be empty"}
		}

		headers[key] = value
	}

	return headers, nil
}

// InvalidHeaderError represents an error parsing a header
type InvalidHeaderError struct {
	Header string
	Reason string
}

func (e *InvalidHeaderError) Error() string {
	return "Invalid header format: \"" + e.Header + "\". " + e.Reason
}

// Type guards for MCP server config types

// IsStdioConfig checks if config is a stdio MCP server config.
func IsStdioConfig(config McpServerConfig) bool {
	return config.GetType() == TransportStdio || config.GetType() == ""
}

// IsSSEConfig checks if config is an SSE MCP server config.
func IsSSEConfig(config McpServerConfig) bool {
	return config.GetType() == TransportSSE
}

// IsHTTPConfig checks if config is an HTTP MCP server config.
func IsHTTPConfig(config McpServerConfig) bool {
	return config.GetType() == TransportHTTP
}

// IsWebSocketConfig checks if config is a WebSocket MCP server config.
func IsWebSocketConfig(config McpServerConfig) bool {
	return config.GetType() == TransportWS
}

// IsSSEIDEConfig checks if config is an SSE IDE MCP server config.
func IsSSEIDEConfig(config McpServerConfig) bool {
	return config.GetType() == TransportSSEIDE
}

// IsSDKConfig checks if config is an SDK MCP server config.
func IsSDKConfig(config McpServerConfig) bool {
	return config.GetType() == TransportSDK
}

// IsClaudeAIProxyConfig checks if config is a Claude.ai proxy MCP server config.
func IsClaudeAIProxyConfig(config McpServerConfig) bool {
	return config.GetType() == "claudeai-proxy"
}

// GetLoggingSafeMcpBaseUrl extracts the MCP server base URL for analytics logging.
// Query strings are stripped because they can contain access tokens.
// Trailing slashes are also removed for normalization.
// Returns empty string for stdio/sdk servers or if URL parsing fails.
func GetLoggingSafeMcpBaseUrl(config McpServerConfig) string {
	urlStr := GetServerURL(config)
	if urlStr == "" {
		return ""
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// Remove query string
	parsed.RawQuery = ""
	parsed.Fragment = ""

	// Remove trailing slash
	result := parsed.String()
	return strings.TrimSuffix(result, "/")
}

// GetServerCommandArray extracts command array from server config (stdio servers only).
// Returns nil for non-stdio servers.
func GetServerCommandArray(config McpServerConfig) []string {
	if config.GetType() != TransportStdio && config.GetType() != "" {
		return nil
	}

	if stdio, ok := config.(*McpStdioServerConfig); ok {
		cmd := []string{stdio.Command}
		cmd = append(cmd, stdio.Args...)
		return cmd
	}
	return nil
}

// GetServerUrl extracts URL from server config (remote servers only).
// Returns empty string for stdio/sdk servers.
func GetServerUrl(config McpServerConfig) string {
	return GetServerURL(config)
}

// CCR proxy URL path markers
var ccrProxyPathMarkers = []string{
	"/v2/session_ingress/shttp/mcp/",
	"/v2/ccr-sessions/",
}

// UnwrapCcrProxyUrl extracts the original vendor URL from CCR proxy URL.
// If the URL is not a CCR proxy URL, returns it unchanged.
func UnwrapCcrProxyUrl(urlStr string) string {
	for _, marker := range ccrProxyPathMarkers {
		if strings.Contains(urlStr, marker) {
			parsed, err := url.Parse(urlStr)
			if err != nil {
				return urlStr
			}
			original := parsed.Query().Get("mcp_url")
			if original != "" {
				return original
			}
			return urlStr
		}
	}
	return urlStr
}

// GetMcpServerSignature computes a dedup signature for an MCP server config.
// Two configs with the same signature are considered "the same server" for
// plugin deduplication. Ignores env and headers.
// Returns empty string for configs with neither command nor url (sdk type).
func GetMcpServerSignature(config McpServerConfig) string {
	cmd := GetServerCommandArray(config)
	if cmd != nil {
		cmdJSON, _ := json.Marshal(cmd)
		return "stdio:" + string(cmdJSON)
	}

	urlStr := GetServerUrl(config)
	if urlStr != "" {
		return "url:" + UnwrapCcrProxyUrl(urlStr)
	}

	return ""
}

// CommandArraysMatch checks if two command arrays match exactly.
func CommandArraysMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// UrlPatternToRegex converts a URL pattern with wildcards to a regex pattern.
// Supports * as wildcard matching any characters.
func UrlPatternToRegex(pattern string) string {
	// Escape regex special characters except *
	escaped := ""
	for _, r := range pattern {
		switch r {
		case '.', '+', '?', '^', '$', '{', '}', '(', ')', '|', '[', ']', '\\':
			escaped += "\\" + string(r)
		case '*':
			escaped += ".*"
		default:
			escaped += string(r)
		}
	}
	return "^" + escaped + "$"
}

// UrlMatchesPattern checks if a URL matches a pattern with wildcard support.
// Supports * as wildcard matching any characters.
// Examples:
//
//	"https://example.com/*" matches "https://example.com/api/v1"
//	"https://*.example.com/*" matches "https://api.example.com/path"
//	"https://example.com:*/\*" matches any port
func UrlMatchesPattern(urlStr, pattern string) bool {
	// Use regex matching for accurate wildcard support
	regexPattern := UrlPatternToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, urlStr)
	if err != nil {
		// Fallback to simple matching on regex error
		return urlStr == pattern
	}
	return matched
}
