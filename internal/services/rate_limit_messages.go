package services

import (
	"fmt"
	"strings"
	"time"

	"claude-code-go/internal/utils"
)

// Rate limit error message prefixes
var RateLimitErrorPrefixes = []string{
	"You've hit your",
	"You've used",
	"You're now using extra usage",
	"You're close to",
	"You're out of extra usage",
}

// RateLimitSeverity represents the severity of a rate limit message
type RateLimitSeverity string

const (
	RateLimitSeverityError   RateLimitSeverity = "error"
	RateLimitSeverityWarning RateLimitSeverity = "warning"
)

// RateLimitMessage represents a rate limit message with severity
type RateLimitMessage struct {
	Message  string            `json:"message"`
	Severity RateLimitSeverity `json:"severity"`
}

// RateLimitType represents different types of rate limits
type RateLimitType string

const (
	RateLimitTypeSevenDaySonnet RateLimitType = "seven_day_sonnet"
	RateLimitTypeSevenDayOpus   RateLimitType = "seven_day_opus"
	RateLimitTypeSevenDay       RateLimitType = "seven_day"
	RateLimitTypeFiveHour       RateLimitType = "five_hour"
	RateLimitTypeOverage        RateLimitType = "overage"
)

// OverageStatus represents the status of overage usage
type OverageStatus string

const (
	OverageStatusAllowed  OverageStatus = "allowed"
	OverageStatusWarning  OverageStatus = "allowed_warning"
	OverageStatusRejected OverageStatus = "rejected"
)

// ClaudeAILimits represents Claude AI usage limits
type ClaudeAILimits struct {
	Status                OverageStatus `json:"status"`
	IsUsingOverage        bool          `json:"is_using_overage"`
	OverageStatus         OverageStatus `json:"overage_status,omitempty"`
	OverageDisabledReason string        `json:"overage_disabled_reason,omitempty"`
	RateLimitType         RateLimitType `json:"rate_limit_type,omitempty"`
	Utilization           float64       `json:"utilization,omitempty"`
	ResetsAt              *time.Time    `json:"resets_at,omitempty"`
	OverageResetsAt       *time.Time    `json:"overage_resets_at,omitempty"`
}

// IsRateLimitErrorMessage checks if a message is a rate limit error
func IsRateLimitErrorMessage(text string) bool {
	for _, prefix := range RateLimitErrorPrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// GetRateLimitMessage returns the appropriate rate limit message based on limit state
func GetRateLimitMessage(limits ClaudeAILimits, model string) *RateLimitMessage {
	// Check overage scenarios first
	if limits.IsUsingOverage {
		// Show warning if approaching overage spending limit
		if limits.OverageStatus == OverageStatusWarning {
			return &RateLimitMessage{
				Message:  "You're close to your extra usage spending limit",
				Severity: RateLimitSeverityWarning,
			}
		}
		return nil
	}

	// ERROR STATES - when limits are rejected
	if limits.Status == OverageStatusRejected {
		return &RateLimitMessage{
			Message:  getLimitReachedText(limits, model),
			Severity: RateLimitSeverityError,
		}
	}

	// WARNING STATES - when approaching limits with early warning
	if limits.Status == OverageStatusWarning {
		// Only show warnings when utilization is above threshold (70%)
		const warningThreshold = 0.7
		if limits.Utilization < warningThreshold {
			return nil
		}

		// Check subscription type and billing access
		if !utils.HasBillingAccess() {
			// Free tier users get different messaging
			text := getEarlyWarningText(limits)
			if text != "" {
				return &RateLimitMessage{
					Message:  text + " · /upgrade for more",
					Severity: RateLimitSeverityWarning,
				}
			}
		}

		text := getEarlyWarningText(limits)
		if text != "" {
			return &RateLimitMessage{
				Message:  text,
				Severity: RateLimitSeverityWarning,
			}
		}
	}

	return nil
}

// GetRateLimitErrorMessage returns the error message for API errors
func GetRateLimitErrorMessage(limits ClaudeAILimits, model string) string {
	message := GetRateLimitMessage(limits, model)
	if message != nil && message.Severity == RateLimitSeverityError {
		return message.Message
	}
	return ""
}

// GetRateLimitWarning returns the warning message for UI footer
func GetRateLimitWarning(limits ClaudeAILimits, model string) string {
	message := GetRateLimitMessage(limits, model)
	if message != nil && message.Severity == RateLimitSeverityWarning {
		return message.Message
	}
	return ""
}

func getLimitReachedText(limits ClaudeAILimits, model string) string {
	var resetMessage string
	if limits.ResetsAt != nil {
		resetTime := formatResetTime(*limits.ResetsAt, true)
		resetMessage = " · resets " + resetTime
	}

	// If BOTH subscription and overage are exhausted
	if limits.OverageStatus == OverageStatusRejected {
		var overageResetMessage string
		if limits.ResetsAt != nil && limits.OverageResetsAt != nil {
			// Use the earlier reset time
			if limits.ResetsAt.Before(*limits.OverageResetsAt) {
				overageResetMessage = " · resets " + formatResetTime(*limits.ResetsAt, true)
			} else {
				overageResetMessage = " · resets " + formatResetTime(*limits.OverageResetsAt, true)
			}
		} else if limits.ResetsAt != nil {
			overageResetMessage = resetMessage
		} else if limits.OverageResetsAt != nil {
			overageResetMessage = " · resets " + formatResetTime(*limits.OverageResetsAt, true)
		}

		if limits.OverageDisabledReason == "out_of_credits" {
			return "You're out of extra usage" + overageResetMessage
		}

		return formatLimitReachedText("limit", overageResetMessage, model)
	}

	var limitName string
	switch limits.RateLimitType {
	case RateLimitTypeSevenDaySonnet:
		// Check subscription type for pro/enterprise
		if utils.GetSubscriptionType() == utils.SubscriptionPro || utils.GetSubscriptionType() == utils.SubscriptionEnterprise {
			limitName = "Sonnet pro limit"
		} else {
			limitName = "Sonnet limit"
		}
	case RateLimitTypeSevenDayOpus:
		limitName = "Opus limit"
	case RateLimitTypeSevenDay:
		limitName = "weekly limit"
	case RateLimitTypeFiveHour:
		limitName = "session limit"
	default:
		limitName = "usage limit"
	}

	return formatLimitReachedText(limitName, resetMessage, model)
}

func getEarlyWarningText(limits ClaudeAILimits) string {
	var limitName string
	switch limits.RateLimitType {
	case RateLimitTypeSevenDay:
		limitName = "weekly limit"
	case RateLimitTypeFiveHour:
		limitName = "session limit"
	case RateLimitTypeSevenDayOpus:
		limitName = "Opus limit"
	case RateLimitTypeSevenDaySonnet:
		limitName = "Sonnet limit"
	case RateLimitTypeOverage:
		limitName = "extra usage"
	default:
		return ""
	}

	used := int(limits.Utilization * 100)
	var resetTime string
	if limits.ResetsAt != nil {
		resetTime = formatResetTime(*limits.ResetsAt, true)
	}

	// Get upsell command based on subscription type and limit type
	upsell := getWarningUpsellText(limits.RateLimitType)

	if used > 0 && resetTime != "" {
		base := fmt.Sprintf("You've used %d%% of your %s · resets %s", used, limitName, resetTime)
		if upsell != "" {
			return base + " · " + upsell
		}
		return base
	}

	if used > 0 {
		base := fmt.Sprintf("You've used %d%% of your %s", used, limitName)
		if upsell != "" {
			return base + " · " + upsell
		}
		return base
	}

	if limits.RateLimitType == RateLimitTypeOverage {
		limitName += " limit"
	}

	if resetTime != "" {
		base := fmt.Sprintf("Approaching %s · resets %s", limitName, resetTime)
		if upsell != "" {
			return base + " · " + upsell
		}
		return base
	}

	base := fmt.Sprintf("Approaching %s", limitName)
	if upsell != "" {
		return base + " · " + upsell
	}
	return base
}

func getWarningUpsellText(rateLimitType RateLimitType) string {
	// Check subscription type for appropriate upsell message
	subType := utils.GetSubscriptionType()

	// Free tier users always get upsell
	if subType == utils.SubscriptionFree {
		if rateLimitType == RateLimitTypeFiveHour {
			return "/upgrade to keep using Claude Code"
		}
		return "/upgrade for higher limits"
	}

	// Pro/Team/Enterprise users don't need upsell
	return ""
}

// GetUsingOverageText returns notification text for overage mode transitions
func GetUsingOverageText(limits ClaudeAILimits) string {
	var resetTime string
	if limits.ResetsAt != nil {
		resetTime = formatResetTime(*limits.ResetsAt, true)
	}

	var limitName string
	switch limits.RateLimitType {
	case RateLimitTypeFiveHour:
		limitName = "session limit"
	case RateLimitTypeSevenDay:
		limitName = "weekly limit"
	case RateLimitTypeSevenDayOpus:
		limitName = "Opus limit"
	case RateLimitTypeSevenDaySonnet:
		limitName = "Sonnet limit"
	}

	if limitName == "" {
		return "Now using extra usage"
	}

	if resetTime != "" {
		return fmt.Sprintf("You're now using extra usage · Your %s resets %s", limitName, resetTime)
	}
	return "You're now using extra usage"
}

func formatLimitReachedText(limit, resetMessage, model string) string {
	// Check if user is internal (ant)
	if utils.IsInternalUser() {
		return fmt.Sprintf("You've reached the %s%s", limit, resetMessage)
	}

	// Standard message for external users
	return fmt.Sprintf("You've hit your %s%s", limit, resetMessage)
}

// formatResetTime formats the reset time for display
func formatResetTime(t time.Time, short bool) string {
	now := time.Now()
	duration := t.Sub(now)

	if duration < 0 {
		return "soon"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if short {
		if hours > 0 {
			return fmt.Sprintf("in %dh %dm", hours, minutes)
		}
		return fmt.Sprintf("in %dm", minutes)
	}

	if hours > 24 {
		days := hours / 24
		remainingHours := hours % 24
		if remainingHours > 0 {
			return fmt.Sprintf("in %d days %d hours", days, remainingHours)
		}
		return fmt.Sprintf("in %d days", days)
	}

	if hours > 0 {
		return fmt.Sprintf("in %d hours %d minutes", hours, minutes)
	}
	return fmt.Sprintf("in %d minutes", minutes)
}
