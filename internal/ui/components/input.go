package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Input Component
// =============================================================================

// Styles
var (
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	multilineHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// InputModel represents a text input field.
type InputModel struct {
	Value       string
	Placeholder string
	Prompt      string
	CursorPos   int
	Width       int
	Multiline   bool
	Focused     bool
	History     []string
	HistoryPos  int
}

// NewInput creates a new input field.
func NewInput(prompt, placeholder string, width int) *InputModel {
	return &InputModel{
		Value:       "",
		Placeholder: placeholder,
		Prompt:      prompt,
		CursorPos:   0,
		Width:       width,
		Multiline:   false,
		Focused:     true,
		History:     []string{},
		HistoryPos:  -1,
	}
}

// Update handles input updates.
func (m *InputModel) Update(msg tea.Msg) tea.Cmd {
	if !m.Focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyBackspace:
			if m.CursorPos > 0 {
				m.Value = m.Value[:m.CursorPos-1] + m.Value[m.CursorPos:]
				m.CursorPos--
			}
		case tea.KeyDelete:
			if m.CursorPos < len(m.Value) {
				m.Value = m.Value[:m.CursorPos] + m.Value[m.CursorPos+1:]
			}
		case tea.KeyLeft:
			if m.CursorPos > 0 {
				m.CursorPos--
			}
		case tea.KeyRight:
			if m.CursorPos < len(m.Value) {
				m.CursorPos++
			}
		case tea.KeyHome:
			m.CursorPos = 0
		case tea.KeyEnd:
			m.CursorPos = len(m.Value)
		case tea.KeyUp:
			// Navigate history
			if len(m.History) > 0 && m.HistoryPos < len(m.History)-1 {
				m.HistoryPos++
				m.Value = m.History[len(m.History)-1-m.HistoryPos]
				m.CursorPos = len(m.Value)
			}
		case tea.KeyDown:
			// Navigate history
			if m.HistoryPos > 0 {
				m.HistoryPos--
				m.Value = m.History[len(m.History)-1-m.HistoryPos]
				m.CursorPos = len(m.Value)
			} else if m.HistoryPos == 0 {
				m.HistoryPos = -1
				m.Value = ""
				m.CursorPos = 0
			}
		case tea.KeyRunes:
			// Insert character at cursor position
			runes := msg.Runes
			m.Value = m.Value[:m.CursorPos] + string(runes) + m.Value[m.CursorPos:]
			m.CursorPos += len(runes)
		}
	}

	return nil
}

// View renders the input field.
func (m *InputModel) View() string {
	var b strings.Builder

	// Render prompt
	b.WriteString(promptStyle.Render(m.Prompt) + " ")

	// Render value with cursor
	if m.Value == "" {
		// Show placeholder
		b.WriteString(placeholderStyle.Render(m.Placeholder))
	} else {
		// Show value with cursor
		before := m.Value[:m.CursorPos]
		atCursor := " "
		after := ""
		if m.CursorPos < len(m.Value) {
			atCursor = string(m.Value[m.CursorPos])
			after = m.Value[m.CursorPos+1:]
		}

		b.WriteString(before + cursorStyle.Render(atCursor) + after)
	}

	// Multiline hint
	if m.Multiline {
		b.WriteString("\n" + multilineHintStyle.Render("Shift+Enter for new line"))
	}

	return inputBoxStyle.Render(b.String())
}

// SetValue sets the input value.
func (m *InputModel) SetValue(value string) {
	m.Value = value
	m.CursorPos = len(value)
}

// Clear clears the input.
func (m *InputModel) Clear() {
	// Save to history if not empty
	if m.Value != "" {
		m.History = append(m.History, m.Value)
	}
	m.Value = ""
	m.CursorPos = 0
	m.HistoryPos = -1
}

// Focus focuses the input.
func (m *InputModel) Focus() {
	m.Focused = true
}

// Blur unfocuses the input.
func (m *InputModel) Blur() {
	m.Focused = false
}

// =============================================================================
// Multiline Input Component
// =============================================================================

// MultilineInputModel represents a multiline text input.
type MultilineInputModel struct {
	*InputModel
	Lines []string
}

// NewMultilineInput creates a new multiline input.
func NewMultilineInput(prompt, placeholder string, width int) *MultilineInputModel {
	return &MultilineInputModel{
		InputModel: NewInput(prompt, placeholder, width),
		Lines:      []string{""},
	}
}

// View renders the multiline input.
func (m *MultilineInputModel) View() string {
	var b strings.Builder

	// Render prompt
	b.WriteString(promptStyle.Render(m.Prompt) + "\n")

	// Render each line
	for i, line := range m.Lines {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  " + line)
	}

	// Show cursor on last line if empty
	if len(m.Lines) == 1 && m.Lines[0] == "" {
		b.WriteString(placeholderStyle.Render(m.Placeholder))
	}

	return inputBoxStyle.Render(b.String())
}
