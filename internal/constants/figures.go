package constants

import "runtime"

// Unicode symbols for UI display
// The former is better vertically aligned, but isn't usually supported on Windows/Linux
var (
	// BlackCircle - used for opus 1m merge notice
	BlackCircle = "⏺"
	// BulletOperator - bullet point
	BulletOperator = "∙"
	// TeardropAsterisk - decorative
	TeardropAsterisk = "✻"
	// UpArrow - used for opus 1m merge notice
	UpArrow = "↑" // \u2191
	// DownArrow - used for scroll hint
	DownArrow = "↓" // \u2193
	// LightningBolt - used for fast mode indicator
	LightningBolt = "↯" // \u21af
)

// Effort level indicators
const (
	// EffortLow - effort level: low
	EffortLow = "○" // \u25cb
	// EffortMedium - effort level: medium
	EffortMedium = "◐" // \u25d0
	// EffortHigh - effort level: high
	EffortHigh = "●" // \u25cf
	// EffortMax - effort level: max (Opus 4.6 only)
	EffortMax = "◉" // \u25c9
)

// Media/trigger status indicators
const (
	// PlayIcon - play button
	PlayIcon = "▶" // \u25b6
	// PauseIcon - pause button
	PauseIcon = "⏸" // \u23f8
)

// MCP subscription indicators
const (
	// RefreshArrow - used for resource update indicator
	RefreshArrow = "↻" // \u21bb
	// ChannelArrow - inbound channel message indicator
	ChannelArrow = "←" // \u2190
	// InjectedArrow - cross-session injected message indicator
	InjectedArrow = "→" // \u2192
	// ForkGlyph - fork directive indicator
	ForkGlyph = "⑂" // \u2442
)

// Review status indicators (ultrareview diamond states)
const (
	// DiamondOpen - running
	DiamondOpen = "◇" // \u25c7
	// DiamondFilled - completed/failed
	DiamondFilled = "◆" // \u25c6
	// ReferenceMark - komejirushi, away-summary recap marker
	ReferenceMark = "※" // \u203b
)

// Issue flag indicator
const (
	// FlagIcon - used for issue flag banner
	FlagIcon = "⚑" // \u2691
)

// Blockquote indicator
const (
	// BlockquoteBar - left one-quarter block, used as blockquote line prefix
	BlockquoteBar = "▎" // \u258e
	// HeavyHorizontal - heavy box-drawing horizontal
	HeavyHorizontal = "━" // \u2501
)

// Bridge status indicators
var (
	// BridgeSpinnerFrames - animation frames for bridge status
	BridgeSpinnerFrames = []string{
		"·|·",
		"·/·",
		"·—·",
		"·\\·",
	}
	// BridgeReadyIndicator - bridge ready status
	BridgeReadyIndicator = "·✔︎·"
	// BridgeFailedIndicator - bridge failed status
	BridgeFailedIndicator = "×"
)

func init() {
	// The black circle is better vertically aligned on macOS, but isn't usually supported on Windows/Linux
	if runtime.GOOS != "darwin" {
		BlackCircle = "●"
	}
}
