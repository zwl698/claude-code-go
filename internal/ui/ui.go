package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 2)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	messageStyle = lipgloss.NewStyle().
			Padding(0, 1)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Padding(0, 1)

	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141")).
				Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
)

// Message represents a chat message for display.
type Message struct {
	Role    string
	Content string
}

// Model represents the main UI state.
type Model struct {
	Messages    []Message
	Input       string
	Ready       bool
	Err         error
	Width       int
	Height      int
	IsProcessing bool
	StatusText  string
}

// InitialModel creates the initial UI model.
func InitialModel() Model {
	return Model{
		Messages: []Message{},
		Input:    "",
		Ready:    false,
	}
}

// Init initializes the UI.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles UI updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.Input != "" {
				// Add user message
				m.Messages = append(m.Messages, Message{
					Role:    "user",
					Content: m.Input,
				})
				m.Input = ""
				m.IsProcessing = true
				m.StatusText = "Thinking..."
			}
			return m, nil

		case tea.KeyBackspace:
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
			}
			return m, nil

		default:
			if msg.Type == tea.KeyRunes {
				m.Input += string(msg.Runes)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI.
func (m Model) View() string {
	if !m.Ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("Claude Code")
	b.WriteString(title + "\n\n")

	// Messages
	visibleHeight := m.Height - 6 // Reserve space for input and status
	messageLines := 0

	for _, msg := range m.Messages {
		var styledMsg string
		if msg.Role == "user" {
			styledMsg = userMessageStyle.Render("You: " + msg.Content)
		} else {
			styledMsg = assistantMessageStyle.Render("Claude: " + msg.Content)
		}

		// Truncate if needed
		lines := strings.Split(styledMsg, "\n")
		for _, line := range lines {
			if messageLines >= visibleHeight {
				break
			}
			b.WriteString(line + "\n")
			messageLines++
		}
	}

	// Fill remaining space
	for i := messageLines; i < visibleHeight; i++ {
		b.WriteString("\n")
	}

	// Status line
	if m.IsProcessing {
		b.WriteString(helpStyle.Render(m.StatusText) + "\n")
	} else {
		b.WriteString("\n")
	}

	// Input area
	inputPrompt := inputStyle.Render("> " + m.Input)
	b.WriteString(inputPrompt + "\n")

	// Help text
	help := helpStyle.Render("Ctrl+C to quit | Enter to send")
	b.WriteString(help)

	return b.String()
}

// AddMessage adds a message to the chat.
func (m *Model) AddMessage(role, content string) {
	m.Messages = append(m.Messages, Message{
		Role:    role,
		Content: content,
	})
}

// SetStatus sets the status text.
func (m *Model) SetStatus(text string) {
	m.StatusText = text
	m.IsProcessing = text != ""
}

// SetError sets an error message.
func (m *Model) SetError(err error) {
	m.Err = err
	m.IsProcessing = false
}

// ========================================
// Components
// ========================================

// SpinnerModel represents a loading spinner.
type SpinnerModel struct {
	frame  int
	frames []string
}

// NewSpinner creates a new spinner.
func NewSpinner() SpinnerModel {
	return SpinnerModel{
		frame:  0,
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Update advances the spinner.
func (s SpinnerModel) Update() SpinnerModel {
	s.frame = (s.frame + 1) % len(s.frames)
	return s
}

// View renders the spinner.
func (s SpinnerModel) View() string {
	return s.frames[s.frame]
}

// ProgressModel represents a progress bar.
type ProgressModel struct {
	percent float64
	width   int
}

// NewProgress creates a new progress bar.
func NewProgress(width int) ProgressModel {
	return ProgressModel{
		percent: 0,
		width:   width,
	}
}

// SetPercent sets the progress percentage.
func (p *ProgressModel) SetPercent(percent float64) {
	p.percent = percent
}

// View renders the progress bar.
func (p ProgressModel) View() string {
	width := p.width - 2 // Account for brackets
	if width < 1 {
		width = 20
	}

	filled := int(float64(width) * p.percent)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s] %.0f%%", bar, p.percent*100)
}

// ListBox represents a selectable list.
type ListBox struct {
	items    []string
	selected int
}

// NewListBox creates a new list.
func NewListBox(items []string) *ListBox {
	return &ListBox{
		items:    items,
		selected: 0,
	}
}

// Up moves selection up.
func (l *ListBox) Up() {
	if l.selected > 0 {
		l.selected--
	}
}

// Down moves selection down.
func (l *ListBox) Down() {
	if l.selected < len(l.items)-1 {
		l.selected++
	}
}

// Selected returns the selected item.
func (l *ListBox) Selected() string {
	if len(l.items) == 0 {
		return ""
	}
	return l.items[l.selected]
}

// View renders the list.
func (l ListBox) View() string {
	var b strings.Builder
	for i, item := range l.items {
		if i == l.selected {
			b.WriteString("> " + item + "\n")
		} else {
			b.WriteString("  " + item + "\n")
		}
	}
	return b.String()
}

// DialogModel represents a dialog box.
type DialogModel struct {
	title   string
	message string
	buttons []string
	focus   int
}

// NewDialog creates a new dialog.
func NewDialog(title, message string, buttons []string) *DialogModel {
	return &DialogModel{
		title:   title,
		message: message,
		buttons: buttons,
		focus:   0,
	}
}

// Left moves focus left.
func (d *DialogModel) Left() {
	if d.focus > 0 {
		d.focus--
	}
}

// Right moves focus right.
func (d *DialogModel) Right() {
	if d.focus < len(d.buttons)-1 {
		d.focus++
	}
}

// Selected returns the selected button.
func (d *DialogModel) Selected() string {
	if len(d.buttons) == 0 {
		return ""
	}
	return d.buttons[d.focus]
}

// View renders the dialog.
func (d DialogModel) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render(d.title)
	b.WriteString(title + "\n\n")

	// Message
	b.WriteString(d.message + "\n\n")

	// Buttons
	for i, btn := range d.buttons {
		if i == d.focus {
			b.WriteString("[ " + btn + " ] ")
		} else {
			b.WriteString("  " + btn + "   ")
		}
	}

	return boxStyle.Render(b.String())
}

