package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/types"
)

// =============================================================================
// Permission Manager
// =============================================================================

// PermissionManager manages permission rules and decisions.
type PermissionManager struct {
	mu                       sync.RWMutex
	mode                     types.PermissionMode
	additionalWorkingDirs    map[string]types.AdditionalWorkingDirectory
	alwaysAllowRules         types.ToolPermissionRulesBySource
	alwaysDenyRules          types.ToolPermissionRulesBySource
	alwaysAskRules           types.ToolPermissionRulesBySource
	bypassPermissionsEnabled bool
	denialTracking           *DenialTrackingState
}

// DenialTrackingState tracks denials for rate limiting
type DenialTrackingState struct {
	denialCount  int
	lastDenial   time.Time
	denialReason string
}

// NewPermissionManager creates a new permission manager.
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		mode:                     types.PermissionModeDefault,
		additionalWorkingDirs:    make(map[string]types.AdditionalWorkingDirectory),
		alwaysAllowRules:         make(types.ToolPermissionRulesBySource),
		alwaysDenyRules:          make(types.ToolPermissionRulesBySource),
		alwaysAskRules:           make(types.ToolPermissionRulesBySource),
		bypassPermissionsEnabled: false,
		denialTracking:           &DenialTrackingState{},
	}
}

// GetMode returns the current permission mode.
func (pm *PermissionManager) GetMode() types.PermissionMode {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.mode
}

// SetMode sets the permission mode.
func (pm *PermissionManager) SetMode(mode types.PermissionMode) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.mode = mode
}

// GetContext returns the current permission context.
func (pm *PermissionManager) GetContext() types.ToolPermissionContext {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return types.ToolPermissionContext{
		Mode:                             pm.mode,
		AdditionalWorkingDirectories:     pm.additionalWorkingDirs,
		AlwaysAllowRules:                 pm.alwaysAllowRules,
		AlwaysDenyRules:                  pm.alwaysDenyRules,
		AlwaysAskRules:                   pm.alwaysAskRules,
		IsBypassPermissionsModeAvailable: pm.bypassPermissionsEnabled,
	}
}

// AddAllowRule adds an allow rule from a source.
func (pm *PermissionManager) AddAllowRule(source types.PermissionRuleSource, rule string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.alwaysAllowRules[source] == nil {
		pm.alwaysAllowRules[source] = []string{}
	}
	pm.alwaysAllowRules[source] = append(pm.alwaysAllowRules[source], rule)
}

// AddDenyRule adds a deny rule from a source.
func (pm *PermissionManager) AddDenyRule(source types.PermissionRuleSource, rule string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.alwaysDenyRules[source] == nil {
		pm.alwaysDenyRules[source] = []string{}
	}
	pm.alwaysDenyRules[source] = append(pm.alwaysDenyRules[source], rule)
}

// AddAskRule adds an ask rule from a source.
func (pm *PermissionManager) AddAskRule(source types.PermissionRuleSource, rule string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.alwaysAskRules[source] == nil {
		pm.alwaysAskRules[source] = []string{}
	}
	pm.alwaysAskRules[source] = append(pm.alwaysAskRules[source], rule)
}

// AddWorkingDirectory adds an additional working directory.
func (pm *PermissionManager) AddWorkingDirectory(path string, source types.WorkingDirectorySource) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	pm.additionalWorkingDirs[absPath] = types.AdditionalWorkingDirectory{
		Path:   absPath,
		Source: source,
	}
}

// =============================================================================
// Permission Checking
// =============================================================================

// CheckPermission checks if a tool can be used with the given input.
func (pm *PermissionManager) CheckPermission(ctx context.Context, toolName string, input json.RawMessage, toolCtx *types.ToolContext) (*types.PermissionResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Check bypass mode
	if pm.mode == types.PermissionModeBypassPermissions && pm.bypassPermissionsEnabled {
		return &types.PermissionResult{
			Behavior: types.PermissionBehaviorAllow,
		}, nil
	}

	// Check accept edits mode (only for file edit tools)
	if pm.mode == types.PermissionModeAcceptEdits {
		if toolName == "Edit" || toolName == "Write" || toolName == "MultiEdit" {
			return &types.PermissionResult{
				Behavior: types.PermissionBehaviorAllow,
			}, nil
		}
	}

	// Check dont-ask mode
	if pm.mode == types.PermissionModeDontAsk {
		return &types.PermissionResult{
			Behavior: types.PermissionBehaviorDeny,
			Message:  "Permission mode is set to 'dont-ask'. All operations are denied.",
			DecisionReason: &types.PermissionDecisionReason{
				Type: "mode",
				Mode: types.PermissionModeDontAsk,
			},
		}, nil
	}

	// Check deny rules first
	for source, rules := range pm.alwaysDenyRules {
		for _, rule := range rules {
			if matchesRule(toolName, rule, input) {
				return &types.PermissionResult{
					Behavior: types.PermissionBehaviorDeny,
					Message:  fmt.Sprintf("Denied by rule '%s' from %s", rule, source),
					DecisionReason: &types.PermissionDecisionReason{
						Type: "rule",
						Rule: &types.PermissionRule{
							Source:       source,
							RuleBehavior: types.PermissionBehaviorDeny,
							RuleValue:    types.PermissionRuleValue{ToolName: toolName, RuleContent: rule},
						},
					},
				}, nil
			}
		}
	}

	// Check allow rules
	for source, rules := range pm.alwaysAllowRules {
		for _, rule := range rules {
			if matchesRule(toolName, rule, input) {
				return &types.PermissionResult{
					Behavior: types.PermissionBehaviorAllow,
					DecisionReason: &types.PermissionDecisionReason{
						Type: "rule",
						Rule: &types.PermissionRule{
							Source:       source,
							RuleBehavior: types.PermissionBehaviorAllow,
							RuleValue:    types.PermissionRuleValue{ToolName: toolName, RuleContent: rule},
						},
					},
				}, nil
			}
		}
	}

	// Check ask rules
	for source, rules := range pm.alwaysAskRules {
		for _, rule := range rules {
			if matchesRule(toolName, rule, input) {
				return &types.PermissionResult{
					Behavior: types.PermissionBehaviorAsk,
					Message:  fmt.Sprintf("Rule '%s' from %s requires approval", rule, source),
					DecisionReason: &types.PermissionDecisionReason{
						Type: "rule",
						Rule: &types.PermissionRule{
							Source:       source,
							RuleBehavior: types.PermissionBehaviorAsk,
							RuleValue:    types.PermissionRuleValue{ToolName: toolName, RuleContent: rule},
						},
					},
				}, nil
			}
		}
	}

	// Default: ask for permission
	return &types.PermissionResult{
		Behavior: types.PermissionBehaviorAsk,
		Message:  fmt.Sprintf("Tool '%s' requires permission", toolName),
	}, nil
}

// =============================================================================
// Rule Matching
// =============================================================================

// matchesRule checks if a tool/input matches a permission rule.
func matchesRule(toolName, rule string, input json.RawMessage) bool {
	// Parse rule (format: "ToolName" or "ToolName(pattern)")
	if strings.HasPrefix(rule, toolName) {
		if len(rule) == len(toolName) {
			// Exact tool name match
			return true
		}
		if rule[len(toolName)] == '(' {
			// Has pattern - extract and match
			pattern := rule[len(toolName)+1:]
			if idx := strings.LastIndex(pattern, ")"); idx != -1 {
				pattern = pattern[:idx]
			}
			return matchesPattern(pattern, input)
		}
	}

	// Check for wildcard rules
	if rule == "*" || rule == "**" {
		return true
	}

	return false
}

// matchesPattern checks if input matches a pattern.
func matchesPattern(pattern string, input json.RawMessage) bool {
	// Parse input as map
	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		return false
	}

	// Check common patterns
	switch {
	case strings.HasPrefix(pattern, "file:"):
		filePattern := strings.TrimPrefix(pattern, "file:")
		if filePath, ok := inputMap["file_path"].(string); ok {
			matched, _ := filepath.Match(filePattern, filePath)
			return matched
		}
		if filePath, ok := inputMap["target_file"].(string); ok {
			matched, _ := filepath.Match(filePattern, filePath)
			return matched
		}

	case strings.HasPrefix(pattern, "path:"):
		pathPattern := strings.TrimPrefix(pattern, "path:")
		if path, ok := inputMap["path"].(string); ok {
			matched, _ := filepath.Match(pathPattern, path)
			return matched
		}

	case strings.HasPrefix(pattern, "command:"):
		cmdPattern := strings.TrimPrefix(pattern, "command:")
		if command, ok := inputMap["command"].(string); ok {
			return strings.Contains(command, cmdPattern)
		}

	default:
		// Try to match any string field
		for _, v := range inputMap {
			if str, ok := v.(string); ok {
				matched, _ := filepath.Match(pattern, str)
				if matched {
					return true
				}
			}
		}
	}

	return false
}

// =============================================================================
// Path Validation
// =============================================================================

// ValidatePath checks if a path is allowed for access.
func (pm *PermissionManager) ValidatePath(path string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check working directory
	cwd, err := os.Getwd()
	if err == nil {
		if strings.HasPrefix(absPath, cwd) {
			return nil
		}
	}

	// Check additional working directories
	for dirPath := range pm.additionalWorkingDirs {
		if strings.HasPrefix(absPath, dirPath) {
			return nil
		}
	}

	// Check common allowed directories
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(absPath, homeDir) {
		return nil
	}

	// Check temp directories
	if strings.HasPrefix(absPath, "/tmp/") || strings.HasPrefix(absPath, "/var/tmp/") {
		return nil
	}

	return fmt.Errorf("path '%s' is outside allowed directories", path)
}

// =============================================================================
// Bash Command Classification
// =============================================================================

// DangerousPatterns contains patterns that are considered dangerous.
var DangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"rm -rf ~",
	"rm -rf $HOME",
	":(){ :|:& };:", // Fork bomb
	"mkfs",
	"dd if=",
	"shred",
	"wget http",
	"curl http",
	"> /dev/sd",
	"> /dev/hd",
	"chmod -R 777 /",
	"chown -R",
	"git push --force",
	"git push -f",
	"git reset --hard",
	"DROP DATABASE",
	"DROP TABLE",
	"TRUNCATE TABLE",
	"DELETE FROM",
	"sudo rm",
}

// ClassifyBashCommand classifies a bash command for risk level.
func (pm *PermissionManager) ClassifyBashCommand(command string) (riskLevel types.RiskLevel, reason string) {
	command = strings.ToLower(command)

	// Check for dangerous patterns
	for _, pattern := range DangerousPatterns {
		if strings.Contains(command, strings.ToLower(pattern)) {
			return types.RiskLevelHigh, fmt.Sprintf("Command contains dangerous pattern: %s", pattern)
		}
	}

	// Check for network operations
	if strings.Contains(command, "curl") || strings.Contains(command, "wget") || strings.Contains(command, "nc ") {
		return types.RiskLevelMedium, "Command involves network operations"
	}

	// Check for file deletion
	if strings.Contains(command, "rm ") {
		if strings.Contains(command, "-rf") || strings.Contains(command, "-r") || strings.Contains(command, "-f") {
			return types.RiskLevelMedium, "Command involves recursive/forced deletion"
		}
		return types.RiskLevelLow, "Command involves file deletion"
	}

	// Check for sudo
	if strings.Contains(command, "sudo ") {
		return types.RiskLevelMedium, "Command requires elevated privileges"
	}

	// Check for package managers
	if strings.Contains(command, "npm install") || strings.Contains(command, "pip install") ||
		strings.Contains(command, "apt install") || strings.Contains(command, "yum install") {
		return types.RiskLevelLow, "Command installs packages"
	}

	// Default to low risk
	return types.RiskLevelLow, "Standard command"
}

// =============================================================================
// Permission Updates
// =============================================================================

// ApplyUpdate applies a permission update.
func (pm *PermissionManager) ApplyUpdate(update types.PermissionUpdate) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Convert destination to source
	source := types.PermissionRuleSource(update.Destination)

	switch update.Type {
	case types.UpdateTypeAddRules:
		for _, rule := range update.Rules {
			switch update.Behavior {
			case types.PermissionBehaviorAllow:
				if pm.alwaysAllowRules[source] == nil {
					pm.alwaysAllowRules[source] = []string{}
				}
				pm.alwaysAllowRules[source] = append(pm.alwaysAllowRules[source], ruleToString(rule))
			case types.PermissionBehaviorDeny:
				if pm.alwaysDenyRules[source] == nil {
					pm.alwaysDenyRules[source] = []string{}
				}
				pm.alwaysDenyRules[source] = append(pm.alwaysDenyRules[source], ruleToString(rule))
			case types.PermissionBehaviorAsk:
				if pm.alwaysAskRules[source] == nil {
					pm.alwaysAskRules[source] = []string{}
				}
				pm.alwaysAskRules[source] = append(pm.alwaysAskRules[source], ruleToString(rule))
			}
		}

	case types.UpdateTypeRemoveRules:
		for _, rule := range update.Rules {
			ruleStr := ruleToString(rule)
			pm.alwaysAllowRules[source] = removeRule(pm.alwaysAllowRules[source], ruleStr)
			pm.alwaysDenyRules[source] = removeRule(pm.alwaysDenyRules[source], ruleStr)
			pm.alwaysAskRules[source] = removeRule(pm.alwaysAskRules[source], ruleStr)
		}

	case types.UpdateTypeSetMode:
		pm.mode = update.Mode

	case types.UpdateTypeAddDirectories:
		for _, dir := range update.Directories {
			pm.additionalWorkingDirs[dir] = types.AdditionalWorkingDirectory{
				Path:   dir,
				Source: types.SourceSession,
			}
		}

	case types.UpdateTypeRemoveDirectories:
		for _, dir := range update.Directories {
			delete(pm.additionalWorkingDirs, dir)
		}
	}

	return nil
}

// ruleToString converts a PermissionRuleValue to a string.
func ruleToString(rule types.PermissionRuleValue) string {
	if rule.RuleContent != "" {
		return fmt.Sprintf("%s(%s)", rule.ToolName, rule.RuleContent)
	}
	return rule.ToolName
}

// removeRule removes a rule from a slice.
func removeRule(rules []string, rule string) []string {
	for i, r := range rules {
		if r == rule {
			return append(rules[:i], rules[i+1:]...)
		}
	}
	return rules
}
