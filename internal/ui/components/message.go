package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Message Components
// =============================================================================

// Styles for message display
var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Bold(true)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	toolUseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	toolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	codeBlockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	messageBoxStyle = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
)

// ContentBlock represents a block of content in a message.
type ContentBlock struct {
	Type string `json:"type"`

	// For text blocks
	Text string `json:"text,omitempty"`

	// For tool_use blocks
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// For tool_result blocks
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`

	// For thinking blocks
	Thinking string `json:"thinking,omitempty"`
}

// MessageModel represents a message for display.
type MessageModel struct {
	Role        string         `json:"role"`
	Content     []ContentBlock `json:"content"`
	Timestamp   string         `json:"timestamp,omitempty"`
	IsStreaming bool           `json:"isStreaming,omitempty"`
}

// RenderMessage renders a message with styling.
func RenderMessage(msg MessageModel, width int) string {
	var b strings.Builder

	// Render role prefix
	var prefix string
	switch msg.Role {
	case "user":
		prefix = userStyle.Render("You:")
	case "assistant":
		prefix = assistantStyle.Render("Claude:")
	case "system":
		prefix = systemStyle.Render("System:")
	default:
		prefix = msg.Role + ":"
	}

	b.WriteString(prefix + "\n")

	// Render content blocks
	for _, block := range msg.Content {
		renderedBlock := renderContentBlock(block, width-4)
		b.WriteString(messageBoxStyle.Render(renderedBlock) + "\n")
	}

	// Add timestamp if present
	if msg.Timestamp != "" {
		b.WriteString(timestampStyle.Render(msg.Timestamp) + "\n")
	}

	return b.String()
}

// renderContentBlock renders a single content block.
func renderContentBlock(block ContentBlock, width int) string {
	switch block.Type {
	case "text":
		return wrapText(block.Text, width)
	case "tool_use":
		return renderToolUse(block, width)
	case "tool_result":
		return renderToolResult(block, width)
	case "thinking":
		return renderThinking(block, width)
	default:
		return fmt.Sprintf("[%s block]", block.Type)
	}
}

// renderToolUse renders a tool use block.
func renderToolUse(block ContentBlock, width int) string {
	var b strings.Builder

	// Tool name with icon
	b.WriteString(toolUseStyle.Render("🔧 "+block.Name) + "\n")

	// Input preview (truncated if too long)
	if len(block.Input) > 0 {
		inputPreview := string(block.Input)
		if len(inputPreview) > 200 {
			inputPreview = inputPreview[:200] + "..."
		}
		b.WriteString(wrapText(inputPreview, width-2))
	}

	return b.String()
}

// renderToolResult renders a tool result block.
func renderToolResult(block ContentBlock, width int) string {
	var b strings.Builder

	// Result header
	icon := "✓"
	style := toolResultStyle
	if block.IsError {
		icon = "✗"
		style = errorStyle
	}

	b.WriteString(style.Render(icon+" Tool Result") + "\n")

	// Content preview
	contentStr := fmt.Sprintf("%v", block.Content)
	if len(contentStr) > 500 {
		contentStr = contentStr[:500] + "\n... (truncated)"
	}
	b.WriteString(wrapText(contentStr, width-2))

	return b.String()
}

// renderThinking renders a thinking block.
func renderThinking(block ContentBlock, width int) string {
	var b strings.Builder

	b.WriteString(systemStyle.Render("💭 Thinking:") + "\n")

	// Truncate thinking if too long
	thinking := block.Thinking
	if len(thinking) > 300 {
		thinking = thinking[:300] + "..."
	}
	b.WriteString(wrapText(thinking, width-2))

	return b.String()
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Wrap long lines
		for len(line) > width {
			// Find a good break point
			breakPoint := width
			for j := width - 1; j >= 0 && j >= width-20; j-- {
				if line[j] == ' ' || line[j] == '\t' {
					breakPoint = j
					break
				}
			}

			result.WriteString(line[:breakPoint] + "\n")
			line = line[breakPoint:]
			if line[0] == ' ' || line[0] == '\t' {
				line = line[1:]
			}
		}
		result.WriteString(line)
	}

	return result.String()
}

// =============================================================================
// Message List Component
// =============================================================================

// MessageListModel represents a list of messages.
type MessageListModel struct {
	Messages   []MessageModel
	ScrollPos  int
	MaxVisible int
	Width      int
	Height     int
}

// NewMessageList creates a new message list.
func NewMessageList(width, height int) *MessageListModel {
	return &MessageListModel{
		Messages:   []MessageModel{},
		ScrollPos:  0,
		MaxVisible: height - 4, // Reserve space for input
		Width:      width,
		Height:     height,
	}
}

// AddMessage adds a message to the list.
func (m *MessageListModel) AddMessage(msg MessageModel) {
	m.Messages = append(m.Messages, msg)
	// Auto-scroll to bottom
	m.ScrollPos = len(m.Messages) - 1
}

// ScrollUp scrolls the list up.
func (m *MessageListModel) ScrollUp() {
	if m.ScrollPos > 0 {
		m.ScrollPos--
	}
}

// ScrollDown scrolls the list down.
func (m *MessageListModel) ScrollDown() {
	if m.ScrollPos < len(m.Messages)-1 {
		m.ScrollPos++
	}
}

// View renders the message list.
func (m *MessageListModel) View() string {
	var b strings.Builder

	// Calculate visible range
	start := 0
	if len(m.Messages) > m.MaxVisible {
		start = len(m.Messages) - m.MaxVisible
		if m.ScrollPos < start {
			start = m.ScrollPos
		}
	}
	end := start + m.MaxVisible
	if end > len(m.Messages) {
		end = len(m.Messages)
	}

	// Render visible messages
	for i := start; i < end; i++ {
		msg := m.Messages[i]
		b.WriteString(RenderMessage(msg, m.Width))
		b.WriteString("\n")
	}

	// Fill remaining space
	linesUsed := strings.Count(b.String(), "\n")
	for i := linesUsed; i < m.MaxVisible; i++ {
		b.WriteString("\n")
	}

	return b.String()
}
