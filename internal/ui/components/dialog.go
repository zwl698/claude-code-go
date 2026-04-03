package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Dialog Components
// =============================================================================

// Styles
var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Margin(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)

	dialogMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 2)

	focusedButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Background(lipgloss.Color("236")).
				Bold(true).
				Padding(0, 2)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	cancelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// DialogModel represents a dialog box.
type DialogModel struct {
	Title   string
	Message string
	Buttons []string
	Focus   int
	Width   int
	Height  int
	Closed  bool
	Result  string
}

// NewDialog creates a new dialog.
func NewDialog(title, message string, buttons []string) *DialogModel {
	return &DialogModel{
		Title:   title,
		Message: message,
		Buttons: buttons,
		Focus:   0,
		Width:   60,
		Height:  10,
		Closed:  false,
	}
}

// Init initializes the dialog.
func (d *DialogModel) Init() tea.Cmd {
	return nil
}

// Update handles dialog updates.
func (d *DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyLeft, tea.KeyShiftTab:
			if d.Focus > 0 {
				d.Focus--
			} else {
				d.Focus = len(d.Buttons) - 1
			}
		case tea.KeyRight, tea.KeyTab:
			if d.Focus < len(d.Buttons)-1 {
				d.Focus++
			} else {
				d.Focus = 0
			}
		case tea.KeyEnter:
			d.Result = d.Buttons[d.Focus]
			d.Closed = true
			return d, tea.Quit
		case tea.KeyEsc:
			d.Result = ""
			d.Closed = true
			return d, tea.Quit
		}
	}
	return d, nil
}

// View renders the dialog.
func (d *DialogModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render(d.Title) + "\n\n")

	// Message (wrapped)
	b.WriteString(dialogMessageStyle.Render(wrapText(d.Message, d.Width-4)) + "\n\n")

	// Buttons
	buttonRow := ""
	for i, btn := range d.Buttons {
		if i == d.Focus {
			buttonRow += focusedButtonStyle.Render("[ "+btn+" ]") + " "
		} else {
			buttonRow += buttonStyle.Render("  "+btn+"  ") + " "
		}
	}
	b.WriteString(buttonRow + "\n")

	// Help text
	b.WriteString("\n" + cancelStyle.Render("Esc to cancel, ←→ to navigate, Enter to confirm"))

	return dialogBoxStyle.Render(b.String())
}

// Show shows the dialog and returns the result.
func (d *DialogModel) Show() string {
	p := tea.NewProgram(d)
	if _, err := p.Run(); err != nil {
		return ""
	}
	return d.Result
}

// =============================================================================
// Confirmation Dialog
// =============================================================================

// ConfirmDialog represents a yes/no confirmation.
type ConfirmDialog struct {
	*DialogModel
}

// NewConfirmDialog creates a new confirmation dialog.
func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		DialogModel: NewDialog(title, message, []string{"Yes", "No"}),
	}
}

// Confirm shows the dialog and returns true if confirmed.
func (d *ConfirmDialog) Confirm() bool {
	result := d.Show()
	return result == "Yes"
}

// =============================================================================
// Alert Dialog
// =============================================================================

// AlertDialog represents an alert dialog.
type AlertDialog struct {
	*DialogModel
}

// NewAlertDialog creates a new alert dialog.
func NewAlertDialog(title, message string) *AlertDialog {
	return &AlertDialog{
		DialogModel: NewDialog(title, message, []string{"OK"}),
	}
}

// Show shows the alert dialog.
func (d *AlertDialog) Show() {
	d.DialogModel.Show()
}

// =============================================================================
// Permission Request Dialog
// =============================================================================

// PermissionDialog represents a permission request.
type PermissionDialog struct {
	*DialogModel
	ToolName string
	Input    string
}

// NewPermissionDialog creates a new permission dialog.
func NewPermissionDialog(toolName, input string) *PermissionDialog {
	message := fmt.Sprintf("Tool '%s' wants to run:\n\n%s", toolName, input)
	return &PermissionDialog{
		DialogModel: NewDialog("Permission Required", message, []string{"Allow", "Deny", "Allow Always"}),
		ToolName:    toolName,
		Input:       input,
	}
}

// =============================================================================
// Input Dialog
// =============================================================================

// InputDialog represents an input dialog.
type InputDialog struct {
	Title       string
	Message     string
	Placeholder string
	Value       string
	CursorPos   int
	Focused     bool
	Width       int
}

// NewInputDialog creates a new input dialog.
func NewInputDialog(title, message, placeholder string) *InputDialog {
	return &InputDialog{
		Title:       title,
		Message:     message,
		Placeholder: placeholder,
		Value:       "",
		CursorPos:   0,
		Focused:     true,
		Width:       60,
	}
}

// Init initializes the input dialog.
func (d *InputDialog) Init() tea.Cmd {
	return nil
}

// Update handles input dialog updates.
func (d *InputDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return d, tea.Quit
		case tea.KeyEsc:
			d.Value = ""
			return d, tea.Quit
		case tea.KeyBackspace:
			if d.CursorPos > 0 {
				d.Value = d.Value[:d.CursorPos-1] + d.Value[d.CursorPos:]
				d.CursorPos--
			}
		case tea.KeyLeft:
			if d.CursorPos > 0 {
				d.CursorPos--
			}
		case tea.KeyRight:
			if d.CursorPos < len(d.Value) {
				d.CursorPos++
			}
		case tea.KeyRunes:
			runes := msg.Runes
			d.Value = d.Value[:d.CursorPos] + string(runes) + d.Value[d.CursorPos:]
			d.CursorPos += len(runes)
		}
	}
	return d, nil
}

// View renders the input dialog.
func (d *InputDialog) View() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render(d.Title) + "\n\n")

	// Message
	b.WriteString(dialogMessageStyle.Render(d.Message) + "\n\n")

	// Input field
	inputBox := dialogBoxStyle.Copy().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("86"))

	if d.Value == "" {
		b.WriteString(inputBox.Render(placeholderStyle.Render(d.Placeholder)))
	} else {
		before := d.Value[:d.CursorPos]
		atCursor := " "
		after := ""
		if d.CursorPos < len(d.Value) {
			atCursor = string(d.Value[d.CursorPos])
			after = d.Value[d.CursorPos+1:]
		}
		inputText := before + cursorStyle.Render(atCursor) + after
		b.WriteString(inputBox.Render(inputText))
	}

	b.WriteString("\n\n" + cancelStyle.Render("Enter to confirm, Esc to cancel"))

	return dialogBoxStyle.Render(b.String())
}

// Show shows the input dialog and returns the result.
func (d *InputDialog) Show() string {
	p := tea.NewProgram(d)
	if _, err := p.Run(); err != nil {
		return ""
	}
	return d.Value
}

// =============================================================================
// Select Dialog
// =============================================================================

// SelectDialog represents a selection dialog.
type SelectDialog struct {
	Title    string
	Options  []string
	Selected int
	Width    int
	Closed   bool
	Result   string
}

// NewSelectDialog creates a new select dialog.
func NewSelectDialog(title string, options []string) *SelectDialog {
	return &SelectDialog{
		Title:    title,
		Options:  options,
		Selected: 0,
		Width:    60,
		Closed:   false,
	}
}

// Init initializes the select dialog.
func (d *SelectDialog) Init() tea.Cmd {
	return nil
}

// Update handles select dialog updates.
func (d *SelectDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if d.Selected > 0 {
				d.Selected--
			}
		case tea.KeyDown:
			if d.Selected < len(d.Options)-1 {
				d.Selected++
			}
		case tea.KeyEnter:
			d.Result = d.Options[d.Selected]
			d.Closed = true
			return d, tea.Quit
		case tea.KeyEsc:
			d.Result = ""
			d.Closed = true
			return d, tea.Quit
		}
	}
	return d, nil
}

// View renders the select dialog.
func (d *SelectDialog) View() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render(d.Title) + "\n\n")

	// Options
	for i, opt := range d.Options {
		if i == d.Selected {
			b.WriteString(focusedButtonStyle.Render("→ "+opt) + "\n")
		} else {
			b.WriteString(buttonStyle.Render("  "+opt) + "\n")
		}
	}

	b.WriteString("\n" + cancelStyle.Render("↑↓ to navigate, Enter to select, Esc to cancel"))

	return dialogBoxStyle.Render(b.String())
}

// Show shows the select dialog and returns the result.
func (d *SelectDialog) Show() string {
	p := tea.NewProgram(d)
	if _, err := p.Run(); err != nil {
		return ""
	}
	return d.Result
}
