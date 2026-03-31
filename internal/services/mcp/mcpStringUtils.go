package mcp

import "strings"

// McpInfo represents parsed MCP server information from a tool name
type McpInfo struct {
	ServerName string
	ToolName   string
}

// McpInfoFromString extracts MCP server information from a tool name string.
// Expected format: "mcp__serverName__toolName"
//
// Known limitation: If a server name contains "__", parsing will be incorrect.
// For example, "mcp__my__server__tool" would parse as server="my" and tool="server__tool"
// instead of server="my__server" and tool="tool". This is rare in practice since server
// names typically don't contain double underscores.
func McpInfoFromString(toolString string) *McpInfo {
	parts := strings.Split(toolString, "__")
	if len(parts) < 2 {
		return nil
	}

	mcpPart := parts[0]
	serverName := parts[1]

	if mcpPart != "mcp" || serverName == "" {
		return nil
	}

	// Join all parts after server name to preserve double underscores in tool names
	var toolName string
	if len(parts) > 2 {
		toolName = strings.Join(parts[2:], "__")
	}

	return &McpInfo{
		ServerName: serverName,
		ToolName:   toolName,
	}
}

// GetMcpPrefix generates the MCP tool/command name prefix for a given server.
func GetMcpPrefix(serverName string) string {
	return "mcp__" + NormalizeNameForMCP(serverName) + "__"
}

// BuildMcpToolName builds a fully qualified MCP tool name from server and tool names.
// Inverse of McpInfoFromString().
func BuildMcpToolName(serverName, toolName string) string {
	return GetMcpPrefix(serverName) + NormalizeNameForMCP(toolName)
}

// GetMcpDisplayName extracts the display name from an MCP tool/command name.
// For example, "mcp__server_name__tool_name" with serverName "server_name" returns "tool_name".
func GetMcpDisplayName(fullName, serverName string) string {
	prefix := "mcp__" + NormalizeNameForMCP(serverName) + "__"
	return strings.TrimPrefix(fullName, prefix)
}

// ExtractMcpToolDisplayName extracts just the tool/command display name from a userFacingName.
// For example, "github - Add comment to issue (MCP)" returns "Add comment to issue".
func ExtractMcpToolDisplayName(userFacingName string) string {
	// Remove the (MCP) suffix if present
	withoutSuffix := strings.TrimSuffix(userFacingName, "(MCP)")
	withoutSuffix = strings.TrimSpace(withoutSuffix)

	// Remove the server prefix (everything before " - ")
	dashIndex := strings.Index(withoutSuffix, " - ")
	if dashIndex != -1 {
		return strings.TrimSpace(withoutSuffix[dashIndex+3:])
	}

	return withoutSuffix
}
