package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Status Components
// =============================================================================

// Styles
var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	successStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	infoStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	warningStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))
)

// Spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerModel represents a loading spinner.
type SpinnerModel struct {
	Frame  int
	Frames []string
}

// NewSpinner creates a new spinner.
func NewSpinner() *SpinnerModel {
	return &SpinnerModel{
		Frame:  0,
		Frames: spinnerFrames,
	}
}

// Update advances the spinner frame.
func (s *SpinnerModel) Update() {
	s.Frame = (s.Frame + 1) % len(s.Frames)
}

// View renders the spinner.
func (s *SpinnerModel) View() string {
	return spinnerStyle.Render(s.Frames[s.Frame])
}

// SpinnerWithText renders a spinner with text.
func (s *SpinnerModel) ViewWithText(text string) string {
	return fmt.Sprintf("%s %s", s.View(), text)
}

// =============================================================================
// Progress Bar
// =============================================================================

// ProgressBarModel represents a progress bar.
type ProgressBarModel struct {
	Percent float64
	Width   int
	Label   string
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(width int) *ProgressBarModel {
	return &ProgressBarModel{
		Percent: 0,
		Width:   width,
		Label:   "",
	}
}

// SetPercent sets the progress percentage.
func (p *ProgressBarModel) SetPercent(percent float64) {
	if percent > 1 {
		percent = 1
	}
	if percent < 0 {
		percent = 0
	}
	p.Percent = percent
}

// View renders the progress bar.
func (p *ProgressBarModel) View() string {
	width := p.Width
	if width < 10 {
		width = 20
	}

	// Account for label and percentage
	barWidth := width - len(p.Label) - 8
	if barWidth < 5 {
		barWidth = 5
	}

	filled := int(float64(barWidth) * p.Percent)
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentText := fmt.Sprintf("%3.0f%%", p.Percent*100)

	return fmt.Sprintf("%s [%s] %s", p.Label, progressBarStyle.Render(bar), percentText)
}

// =============================================================================
// Status Bar
// =============================================================================

// StatusBarModel represents a status bar.
type StatusBarModel struct {
	LeftText  string
	RightText string
	Width     int
	Style     string // "info", "success", "error", "warning"
}

// NewStatusBar creates a new status bar.
func NewStatusBar(width int) *StatusBarModel {
	return &StatusBarModel{
		Width: width,
		Style: "info",
	}
}

// View renders the status bar.
func (s *StatusBarModel) View() string {
	var style lipgloss.Style
	switch s.Style {
	case "success":
		style = successStatusStyle
	case "error":
		style = errorStatusStyle
	case "warning":
		style = warningStatusStyle
	default:
		style = infoStatusStyle
	}

	// Calculate padding
	leftLen := lipgloss.Width(s.LeftText)
	rightLen := lipgloss.Width(s.RightText)
	padding := s.Width - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	content := s.LeftText + strings.Repeat(" ", padding) + s.RightText
	return statusBarStyle.Render(style.Render(content))
}

// =============================================================================
// Processing Indicator
// =============================================================================

// ProcessingModel represents a processing indicator.
type ProcessingModel struct {
	Spinner  *SpinnerModel
	Message  string
	Details  string
	Progress *ProgressBarModel
}

// NewProcessingIndicator creates a new processing indicator.
func NewProcessingIndicator(message string) *ProcessingModel {
	return &ProcessingModel{
		Spinner: NewSpinner(),
		Message: message,
	}
}

// Update advances the processing animation.
func (p *ProcessingModel) Update() {
	p.Spinner.Update()
}

// View renders the processing indicator.
func (p *ProcessingModel) View() string {
	var b strings.Builder

	// Spinner with message
	b.WriteString(p.Spinner.ViewWithText(p.Message))

	// Details if present
	if p.Details != "" {
		b.WriteString("\n  " + infoStatusStyle.Render(p.Details))
	}

	// Progress bar if present
	if p.Progress != nil {
		b.WriteString("\n" + p.Progress.View())
	}

	return b.String()
}

// SetProgress sets the progress.
func (p *ProcessingModel) SetProgress(percent float64, details string) {
	if p.Progress == nil {
		p.Progress = NewProgressBar(40)
	}
	p.Progress.SetPercent(percent)
	p.Details = details
}
