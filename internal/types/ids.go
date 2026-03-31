// Package types contains core type definitions for the claude-code CLI.
// These types are translated from the TypeScript source to Go.
package types

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
)

// SessionId uniquely identifies a Claude Code session.
// This is a branded type to prevent mixing with other string IDs.
type SessionId string

// AgentId uniquely identifies a subagent within a session.
// When present, indicates the context is a subagent (not the main session).
type AgentId string

// AsSessionId casts a raw string to SessionId.
// Use sparingly - prefer GetSessionId() when possible.
func AsSessionId(id string) SessionId {
	return SessionId(id)
}

// AsAgentId casts a raw string to AgentId.
// Use sparingly - prefer CreateAgentId() when possible.
func AsAgentId(id string) AgentId {
	return AgentId(id)
}

// Agent ID pattern: a + optional label- + 16 hex chars
var agentIdPattern = regexp.MustCompile(`^a(?:.+-)?[0-9a-f]{16}$`)

// ToAgentId validates and brands a string as AgentId.
// Returns empty string if the string doesn't match the pattern.
func ToAgentId(s string) AgentId {
	if agentIdPattern.MatchString(s) {
		return AgentId(s)
	}
	return ""
}

// CreateAgentId generates a new unique AgentId.
// Format: a + 16 hex characters
func CreateAgentId() AgentId {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return AgentId("a" + hex.EncodeToString(bytes))
}

// String returns the string representation of the SessionId.
func (s SessionId) String() string {
	return string(s)
}

// String returns the string representation of the AgentId.
func (a AgentId) String() string {
	return string(a)
}

// IsValid checks if the AgentId matches the expected pattern.
func (a AgentId) IsValid() bool {
	return agentIdPattern.MatchString(string(a))
}

// GenerateTaskId generates a unique task ID with the given type prefix.
// Task ID format: prefix + 8 lowercase alphanumeric characters.
func GenerateTaskId(taskType TaskType) string {
	prefix := getTaskIdPrefix(taskType)
	bytes := make([]byte, 8)
	rand.Read(bytes)

	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	id := prefix
	for i := 0; i < 8; i++ {
		id += string(alphabet[bytes[i]%byte(len(alphabet))])
	}
	return id
}

// Task ID prefixes
func getTaskIdPrefix(t TaskType) string {
	prefixes := map[TaskType]string{
		TaskTypeLocalBash:         "b",
		TaskTypeLocalAgent:        "a",
		TaskTypeRemoteAgent:       "r",
		TaskTypeInProcessTeammate: "t",
		TaskTypeLocalWorkflow:     "w",
		TaskTypeMonitorMcp:        "m",
		TaskTypeDream:             "d",
	}
	if prefix, ok := prefixes[t]; ok {
		return prefix
	}
	return "x"
}
