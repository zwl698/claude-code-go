// Package utils provides utility functions for the claude-code CLI.
// This file contains effort level configuration and utilities.
package utils

import (
	"os"
	"strconv"
	"strings"
)

// EffortLevel represents an effort level for model thinking.
type EffortLevel string

const (
	EffortLevelLow    EffortLevel = "low"
	EffortLevelMedium EffortLevel = "medium"
	EffortLevelHigh   EffortLevel = "high"
	EffortLevelMax    EffortLevel = "max"
)

// EffortLevels is the list of valid effort levels.
var EffortLevels = []EffortLevel{
	EffortLevelLow,
	EffortLevelMedium,
	EffortLevelHigh,
	EffortLevelMax,
}

// EffortValue represents either an effort level or a numeric value.
type EffortValue struct {
	Level   *EffortLevel
	Numeric *int
}

// NewEffortValueFromLevel creates an EffortValue from an effort level.
func NewEffortValueFromLevel(level EffortLevel) EffortValue {
	return EffortValue{Level: &level}
}

// NewEffortValueFromNumeric creates an EffortValue from a numeric value.
func NewEffortValueFromNumeric(value int) EffortValue {
	return EffortValue{Numeric: &value}
}

// ModelSupportsEffort checks if a model supports the effort parameter.
func ModelSupportsEffort(model string) bool {
	m := strings.ToLower(model)

	// Check for environment override
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_ALWAYS_ENABLE_EFFORT")) {
		return true
	}

	// Check for 3P model capability override
	if override := Get3PModelCapabilityOverride(model, "effort"); override != nil {
		return *override
	}

	// Supported by a subset of Claude 4 models
	if strings.Contains(m, "opus-4-6") || strings.Contains(m, "sonnet-4-6") {
		return true
	}

	// Exclude any other known legacy models (haiku, older opus/sonnet variants)
	if strings.Contains(m, "haiku") || strings.Contains(m, "sonnet") || strings.Contains(m, "opus") {
		return false
	}

	// Default to true for unknown model strings on 1P.
	// Do not default to true for 3P as they have different formats for their model strings.
	return GetAPIProvider() == APIProviderFirstParty
}

// ModelSupportsMaxEffort checks if a model supports 'max' effort level.
func ModelSupportsMaxEffort(model string) bool {
	// Check for 3P model capability override
	if override := Get3PModelCapabilityOverride(model, "max_effort"); override != nil {
		return *override
	}

	if strings.Contains(strings.ToLower(model), "opus-4-6") {
		return true
	}

	// Check for ant-only models
	if os.Getenv("USER_TYPE") == "ant" {
		if resolved := ResolveAntModel(model); resolved != nil {
			return true
		}
	}

	return false
}

// IsEffortLevel checks if a string is a valid effort level.
func IsEffortLevel(value string) bool {
	for _, level := range EffortLevels {
		if string(level) == value {
			return true
		}
	}
	return false
}

// ParseEffortValue parses an effort value from various input types.
func ParseEffortValue(value interface{}) *EffortValue {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
		lower := strings.ToLower(v)
		if IsEffortLevel(lower) {
			level := EffortLevel(lower)
			return &EffortValue{Level: &level}
		}
		if num, err := strconv.Atoi(v); err == nil && IsValidNumericEffort(num) {
			return &EffortValue{Numeric: &num}
		}
	case int:
		if IsValidNumericEffort(v) {
			return &EffortValue{Numeric: &v}
		}
	case float64:
		if IsValidNumericEffort(int(v)) {
			num := int(v)
			return &EffortValue{Numeric: &num}
		}
	}

	return nil
}

// ToPersistableEffort converts an effort value to a persistable effort level.
// Numeric values are model-default only and not persisted.
// 'max' is session-scoped for external users (ants can persist it).
func ToPersistableEffort(value *EffortValue) *EffortLevel {
	if value == nil {
		return nil
	}

	if value.Level != nil {
		level := *value.Level
		if level == EffortLevelLow || level == EffortLevelMedium || level == EffortLevelHigh {
			return &level
		}
		if level == EffortLevelMax && os.Getenv("USER_TYPE") == "ant" {
			return &level
		}
	}

	return nil
}

// GetEffortEnvOverride gets the effort level override from environment.
func GetEffortEnvOverride() *EffortValue {
	envOverride := os.Getenv("CLAUDE_CODE_EFFORT_LEVEL")
	if envOverride == "" {
		return nil
	}

	lower := strings.ToLower(envOverride)
	if lower == "unset" || lower == "auto" {
		// Return special nil to indicate "unset"
		return nil
	}

	return ParseEffortValue(envOverride)
}

// ResolveAppliedEffort resolves the effort value that will be sent to the API.
// Priority: env CLAUDE_CODE_EFFORT_LEVEL → appState.effortValue → model default
func ResolveAppliedEffort(model string, appStateEffortValue *EffortValue) *EffortValue {
	envOverride := GetEffortEnvOverride()
	if envOverride == nil && os.Getenv("CLAUDE_CODE_EFFORT_LEVEL") != "" {
		// Env was explicitly set to "unset" or "auto"
		return nil
	}

	var resolved *EffortValue
	if envOverride != nil {
		resolved = envOverride
	} else if appStateEffortValue != nil {
		resolved = appStateEffortValue
	} else {
		resolved = GetDefaultEffortForModel(model)
	}

	// API rejects 'max' on non-Opus-4.6 models — downgrade to 'high'
	if resolved != nil && resolved.Level != nil && *resolved.Level == EffortLevelMax {
		if !ModelSupportsMaxEffort(model) {
			high := EffortLevelHigh
			return &EffortValue{Level: &high}
		}
	}

	return resolved
}

// GetDisplayedEffortLevel resolves the effort level to show the user.
// Wraps ResolveAppliedEffort with the 'high' fallback.
func GetDisplayedEffortLevel(model string, appStateEffort *EffortValue) EffortLevel {
	resolved := ResolveAppliedEffort(model, appStateEffort)
	if resolved == nil {
		return EffortLevelHigh
	}
	return ConvertEffortValueToLevel(resolved)
}

// GetEffortSuffix builds the "with {level} effort" suffix shown in UI.
// Returns empty string if the user hasn't explicitly set an effort value.
func GetEffortSuffix(model string, effortValue *EffortValue) string {
	if effortValue == nil {
		return ""
	}

	resolved := ResolveAppliedEffort(model, effortValue)
	if resolved == nil {
		return ""
	}

	level := ConvertEffortValueToLevel(resolved)
	return " with " + string(level) + " effort"
}

// IsValidNumericEffort checks if a numeric value is a valid effort number.
func IsValidNumericEffort(value int) bool {
	return true // All integers are technically valid
}

// ConvertEffortValueToLevel converts an effort value to an effort level.
func ConvertEffortValueToLevel(value *EffortValue) EffortLevel {
	if value == nil {
		return EffortLevelHigh
	}

	if value.Level != nil {
		level := *value.Level
		// Runtime guard: value may come from remote config
		if IsEffortLevel(string(level)) {
			return level
		}
		return EffortLevelHigh
	}

	if value.Numeric != nil && os.Getenv("USER_TYPE") == "ant" {
		num := *value.Numeric
		if num <= 50 {
			return EffortLevelLow
		}
		if num <= 85 {
			return EffortLevelMedium
		}
		if num <= 100 {
			return EffortLevelHigh
		}
		return EffortLevelMax
	}

	return EffortLevelHigh
}

// GetEffortLevelDescription returns a user-facing description for effort levels.
func GetEffortLevelDescription(level EffortLevel) string {
	switch level {
	case EffortLevelLow:
		return "Quick, straightforward implementation with minimal overhead"
	case EffortLevelMedium:
		return "Balanced approach with standard implementation and testing"
	case EffortLevelHigh:
		return "Comprehensive implementation with extensive testing and documentation"
	case EffortLevelMax:
		return "Maximum capability with deepest reasoning (Opus 4.6 only)"
	default:
		return "Balanced approach with standard implementation and testing"
	}
}

// GetEffortValueDescription returns a user-facing description for effort values.
func GetEffortValueDescription(value *EffortValue) string {
	if value == nil {
		return GetEffortLevelDescription(EffortLevelHigh)
	}

	if value.Numeric != nil && os.Getenv("USER_TYPE") == "ant" {
		return "[ANT-ONLY] Numeric effort value of " + strconv.Itoa(*value.Numeric)
	}

	if value.Level != nil {
		return GetEffortLevelDescription(*value.Level)
	}

	return GetEffortLevelDescription(EffortLevelHigh)
}

// OpusDefaultEffortConfig represents the default effort config for Opus models.
type OpusDefaultEffortConfig struct {
	Enabled           bool   `json:"enabled"`
	DialogTitle       string `json:"dialogTitle"`
	DialogDescription string `json:"dialogDescription"`
}

// DefaultOpusEffortConfig is the default config for Opus default effort.
var DefaultOpusEffortConfig = OpusDefaultEffortConfig{
	Enabled:           true,
	DialogTitle:       "We recommend medium effort for Opus",
	DialogDescription: "Effort determines how long Claude thinks for when completing your task. We recommend medium effort for most tasks to balance speed and intelligence and maximize rate limits. Use ultrathink to trigger high effort when needed.",
}

// GetOpusDefaultEffortConfig returns the Opus default effort configuration.
func GetOpusDefaultEffortConfig() OpusDefaultEffortConfig {
	// In the full implementation, this would fetch from GrowthBook
	return DefaultOpusEffortConfig
}

// GetDefaultEffortForModel returns the default effort value for a model.
func GetDefaultEffortForModel(model string) *EffortValue {
	m := strings.ToLower(model)

	// Check for ant-specific model defaults
	if os.Getenv("USER_TYPE") == "ant" {
		if config := GetAntModelOverrideConfig(); config != nil {
			if config.DefaultModel != "" && m == strings.ToLower(config.DefaultModel) {
				if config.DefaultModelEffortLevel != nil {
					return &EffortValue{Level: config.DefaultModelEffortLevel}
				}
			}
		}

		if antModel := ResolveAntModel(model); antModel != nil {
			if antModel.DefaultEffortLevel != nil {
				return &EffortValue{Level: antModel.DefaultEffortLevel}
			}
			if antModel.DefaultEffortValue != nil {
				return &EffortValue{Numeric: antModel.DefaultEffortValue}
			}
		}

		// Always default ants to undefined/high
		return nil
	}

	// Default effort on Opus 4.6 to medium for Pro subscribers
	if strings.Contains(m, "opus-4-6") {
		// Check subscriber status (would need auth manager integration)
		// For now, return nil to use API default
	}

	// When ultrathink feature is on, default effort to medium
	if IsUltrathinkEnabled() && ModelSupportsEffort(model) {
		medium := EffortLevelMedium
		return &EffortValue{Level: &medium}
	}

	// Fallback to undefined, which means we don't set an effort level
	return nil
}

// AntModelConfig represents ant-only model configuration.
type AntModelConfig struct {
	DefaultModel            string       `json:"defaultModel"`
	DefaultModelEffortLevel *EffortLevel `json:"defaultModelEffortLevel,omitempty"`
}

// AntModel represents an ant-only model definition.
type AntModel struct {
	Name               string       `json:"name"`
	DefaultEffortLevel *EffortLevel `json:"defaultEffortLevel,omitempty"`
	DefaultEffortValue *int         `json:"defaultEffortValue,omitempty"`
}

// Get3PModelCapabilityOverride returns an override for 3P model capabilities.
func Get3PModelCapabilityOverride(model, capability string) *bool {
	// In the full implementation, this would check model support overrides
	return nil
}

// ResolveAntModel resolves an ant-only model by name.
func ResolveAntModel(model string) *AntModel {
	// In the full implementation, this would look up ant-only models
	return nil
}

// GetAntModelOverrideConfig returns the ant model override configuration.
func GetAntModelOverrideConfig() *AntModelConfig {
	// In the full implementation, this would fetch from config
	return nil
}

// IsUltrathinkEnabled checks if ultrathink mode is enabled.
func IsUltrathinkEnabled() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_ULTRATHINK"))
}
