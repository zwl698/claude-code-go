package permissions

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// =============================================================================
// Permission Types
// =============================================================================

// PermissionBehavior represents the action to take for a permission.
type PermissionBehavior string

const (
	PermissionBehaviorAllow       PermissionBehavior = "allow"
	PermissionBehaviorDeny        PermissionBehavior = "deny"
	PermissionBehaviorAsk         PermissionBehavior = "ask"
	PermissionBehaviorPassthrough PermissionBehavior = "passthrough"
)

// PermissionRuleSource indicates where a permission rule originated from.
type PermissionRuleSource string

const (
	SourceUserSettings    PermissionRuleSource = "userSettings"
	SourceProjectSettings PermissionRuleSource = "projectSettings"
	SourceLocalSettings   PermissionRuleSource = "localSettings"
	SourceFlagSettings    PermissionRuleSource = "flagSettings"
	SourcePolicySettings  PermissionRuleSource = "policySettings"
	SourceCliArg          PermissionRuleSource = "cliArg"
	SourceCommand         PermissionRuleSource = "command"
	SourceSession         PermissionRuleSource = "session"
)

// PermissionRuleValue specifies which tool and optional content for a rule.
type PermissionRuleValue struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
}

// PermissionRule represents a permission rule with its source and behavior.
type PermissionRule struct {
	Source       PermissionRuleSource `json:"source"`
	RuleBehavior PermissionBehavior   `json:"ruleBehavior"`
	RuleValue    PermissionRuleValue  `json:"ruleValue"`
}

// PermissionDecision represents a user's decision on a permission request.
type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
	PermissionDecisionAsk   PermissionDecision = "ask"
)

// PermissionDecisionReason explains why a permission decision was made.
type PermissionDecisionReason struct {
	Type       string          `json:"type"`
	Rule       *PermissionRule `json:"rule,omitempty"`
	Reason     string          `json:"reason,omitempty"`
	HookName   string          `json:"hookName,omitempty"`
	Mode       string          `json:"mode,omitempty"`
	Classifier string          `json:"classifier,omitempty"`
}

// PermissionResult represents the result of a permission check.
type PermissionResult struct {
	Behavior       PermissionBehavior        `json:"behavior"`
	Message        string                    `json:"message,omitempty"`
	DecisionReason *PermissionDecisionReason `json:"decisionReason,omitempty"`
	Rule           *PermissionRule           `json:"rule,omitempty"`
}

// IsAllowed returns true if the permission is allowed.
func (r *PermissionResult) IsAllowed() bool {
	return r.Behavior == PermissionBehaviorAllow
}

// IsDenied returns true if the permission is denied.
func (r *PermissionResult) IsDenied() bool {
	return r.Behavior == PermissionBehaviorDeny
}

// NeedsPrompt returns true if the permission needs user input.
func (r *PermissionResult) NeedsPrompt() bool {
	return r.Behavior == PermissionBehaviorAsk || r.Behavior == PermissionBehaviorPassthrough
}

// =============================================================================
// Permission Manager
// =============================================================================

// Manager manages permission rules and checks.
type Manager struct {
	mu    sync.RWMutex
	rules map[PermissionRuleSource][]PermissionRule
	// Wildcard pattern cache
	wildcardCache map[string]*regexp.Regexp
}

// NewManager creates a new permission manager.
func NewManager() *Manager {
	return &Manager{
		rules:         make(map[PermissionRuleSource][]PermissionRule),
		wildcardCache: make(map[string]*regexp.Regexp),
	}
}

// AddRule adds a permission rule.
func (m *Manager) AddRule(source PermissionRuleSource, rule PermissionRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules[source] = append(m.rules[source], rule)
}

// RemoveRule removes a permission rule.
func (m *Manager) RemoveRule(source PermissionRuleSource, ruleValue PermissionRuleValue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rules := m.rules[source]
	newRules := make([]PermissionRule, 0, len(rules))
	for _, r := range rules {
		if r.RuleValue.ToolName != ruleValue.ToolName || r.RuleValue.RuleContent != ruleValue.RuleContent {
			newRules = append(newRules, r)
		}
	}
	m.rules[source] = newRules
}

// GetRules returns all rules from a source.
func (m *Manager) GetRules(source PermissionRuleSource) []PermissionRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]PermissionRule{}, m.rules[source]...)
}

// ClearRules removes all rules from a source.
func (m *Manager) ClearRules(source PermissionRuleSource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.rules, source)
}

// =============================================================================
// Permission Checking
// =============================================================================

// CheckPermission checks if an action is permitted.
func (m *Manager) CheckPermission(ctx context.Context, toolName, ruleContent string) *PermissionResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check rules in priority order
	sources := []PermissionRuleSource{
		SourcePolicySettings,
		SourceFlagSettings,
		SourceCliArg,
		SourceProjectSettings,
		SourceLocalSettings,
		SourceUserSettings,
		SourceCommand,
		SourceSession,
	}

	for _, source := range sources {
		rules := m.rules[source]
		for _, rule := range rules {
			if m.matchesRule(rule.RuleValue, toolName, ruleContent) {
				return &PermissionResult{
					Behavior: rule.RuleBehavior,
					Rule:     &rule,
					DecisionReason: &PermissionDecisionReason{
						Type: "rule",
						Rule: &rule,
					},
				}
			}
		}
	}

	// Default: ask for permission
	return &PermissionResult{
		Behavior: PermissionBehaviorAsk,
		Message:  fmt.Sprintf("Permission required for %s", toolName),
	}
}

// matchesRule checks if a rule matches the given tool and content.
func (m *Manager) matchesRule(ruleValue PermissionRuleValue, toolName, ruleContent string) bool {
	// Check tool name match
	if !m.matchToolName(ruleValue.ToolName, toolName) {
		return false
	}

	// If rule has no content, it matches all uses of the tool
	if ruleValue.RuleContent == "" {
		return true
	}

	// If rule has content, check if it matches
	if ruleContent == "" {
		return false
	}

	// Wildcard matching for rule content
	return m.matchPattern(ruleValue.RuleContent, ruleContent)
}

// matchToolName checks if a tool name pattern matches.
func (m *Manager) matchToolName(pattern, toolName string) bool {
	// Exact match
	if pattern == toolName {
		return true
	}

	// Wildcard match (e.g., "Bash(git:*)" matches "Bash")
	if strings.Contains(pattern, "(") {
		basePattern := strings.Split(pattern, "(")[0]
		return basePattern == toolName
	}

	return false
}

// matchPattern matches a wildcard pattern against a string.
func (m *Manager) matchPattern(pattern, s string) bool {
	// Check cache
	re, cached := m.wildcardCache[pattern]
	if !cached {
		// Convert wildcard pattern to regex
		regexPattern := m.wildcardToRegex(pattern)
		re = regexp.MustCompile("^" + regexPattern + "$")
		m.wildcardCache[pattern] = re
	}

	return re.MatchString(s)
}

// wildcardToRegex converts a wildcard pattern to regex.
func (m *Manager) wildcardToRegex(pattern string) string {
	var result strings.Builder
	result.Grow(len(pattern) * 2)

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			result.WriteString(".*")
		case '?':
			result.WriteString(".")
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			result.WriteByte('\\')
			result.WriteByte(ch)
		default:
			result.WriteByte(ch)
		}
	}

	return result.String()
}

// =============================================================================
// Permission Parsing
// =============================================================================

// ParseRuleValue parses a rule string like "Bash(git:*)" into a PermissionRuleValue.
func ParseRuleValue(s string) PermissionRuleValue {
	// Format: ToolName or ToolName(content)
	if !strings.Contains(s, "(") {
		return PermissionRuleValue{
			ToolName:    s,
			RuleContent: "",
		}
	}

	// Extract tool name and content
	parts := strings.SplitN(s, "(", 2)
	toolName := parts[0]
	content := strings.TrimSuffix(parts[1], ")")

	return PermissionRuleValue{
		ToolName:    toolName,
		RuleContent: content,
	}
}

// FormatRuleValue formats a PermissionRuleValue as a string.
func FormatRuleValue(v PermissionRuleValue) string {
	if v.RuleContent == "" {
		return v.ToolName
	}
	return fmt.Sprintf("%s(%s)", v.ToolName, v.RuleContent)
}

// =============================================================================
// Permission Modes
// =============================================================================

// PermissionMode represents a permission mode.
type PermissionMode string

const (
	PermissionModeDefault PermissionMode = "default"
	PermissionModeAccept  PermissionMode = "accept"
	PermissionModePlan    PermissionMode = "plan"
)

// GetModeBehavior returns the default behavior for a mode.
func GetModeBehavior(mode PermissionMode) PermissionBehavior {
	switch mode {
	case PermissionModeAccept:
		return PermissionBehaviorAllow
	case PermissionModePlan:
		return PermissionBehaviorAsk
	default:
		return PermissionBehaviorAsk
	}
}

// =============================================================================
// Permission Update
// =============================================================================

// PermissionUpdateDestination specifies where an update should be saved.
type PermissionUpdateDestination string

const (
	DestinationUser    PermissionUpdateDestination = "user"
	DestinationProject PermissionUpdateDestination = "project"
	DestinationLocal   PermissionUpdateDestination = "local"
)

// PermissionUpdate represents a permission update.
type PermissionUpdate struct {
	Destination PermissionUpdateDestination `json:"destination"`
	Operation   string                      `json:"operation"` // add, remove, replace
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
}

// Apply applies a permission update to the manager.
func (m *Manager) Apply(update PermissionUpdate) error {
	var source PermissionRuleSource
	switch update.Destination {
	case DestinationUser:
		source = SourceUserSettings
	case DestinationProject:
		source = SourceProjectSettings
	case DestinationLocal:
		source = SourceLocalSettings
	default:
		return fmt.Errorf("unknown destination: %s", update.Destination)
	}

	switch update.Operation {
	case "add":
		for _, ruleValue := range update.Rules {
			m.AddRule(source, PermissionRule{
				Source:       source,
				RuleBehavior: update.Behavior,
				RuleValue:    ruleValue,
			})
		}
	case "remove":
		for _, ruleValue := range update.Rules {
			m.RemoveRule(source, ruleValue)
		}
	case "replace":
		m.ClearRules(source)
		for _, ruleValue := range update.Rules {
			m.AddRule(source, PermissionRule{
				Source:       source,
				RuleBehavior: update.Behavior,
				RuleValue:    ruleValue,
			})
		}
	default:
		return fmt.Errorf("unknown operation: %s", update.Operation)
	}

	return nil
}
