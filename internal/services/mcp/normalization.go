package mcp

import "strings"

// ClaudeAIServerPrefix is the prefix for claude.ai server names
const ClaudeAIServerPrefix = "claude.ai "

// NormalizeNameForMCP normalizes server names to be compatible with the API pattern ^[a-zA-Z0-9_-]{1,64}$
// Replaces any invalid characters (including dots and spaces) with underscores.
//
// For claude.ai servers (names starting with "claude.ai "), also collapses
// consecutive underscores and strips leading/trailing underscores to prevent
// interference with the __ delimiter used in MCP tool names.
func NormalizeNameForMCP(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	normalized := result.String()

	// For claude.ai servers, collapse consecutive underscores and strip leading/trailing
	if strings.HasPrefix(name, ClaudeAIServerPrefix) {
		// Collapse consecutive underscores
		var collapsed strings.Builder
		prevUnderscore := false
		for _, r := range normalized {
			if r == '_' {
				if !prevUnderscore {
					collapsed.WriteRune(r)
				}
				prevUnderscore = true
			} else {
				collapsed.WriteRune(r)
				prevUnderscore = false
			}
		}
		normalized = strings.Trim(collapsed.String(), "_")
	}

	return normalized
}
