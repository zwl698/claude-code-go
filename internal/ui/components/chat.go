package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Chat Interface Component
// =============================================================================

// Styles
var (
	chatContainerStyle = lipgloss.NewStyle().
				Padding(1, 2)

	chatHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 1)

	chatFooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))
)

// ChatState represents the state of the chat interface.
type ChatState int

const (
	ChatStateIdle ChatState = iota
	ChatStateProcessing
	ChatStateWaitingForInput
	ChatStateError
)

// ChatModel represents the main chat interface.
type ChatModel struct {
	Messages    *MessageListModel
	Input       *InputModel
	Status      *StatusBarModel
	Spinner     *SpinnerModel
	State       ChatState
	Width       int
	Height      int
	Error       error
	HelpVisible bool
}

// NewChatModel creates a new chat interface.
func NewChatModel(width, height int) *ChatModel {
	return &ChatModel{
		Messages:    NewMessageList(width, height-6),
		Input:       NewInput(">", "Type your message...", width-4),
		Status:      NewStatusBar(width),
		Spinner:     NewSpinner(),
		State:       ChatStateIdle,
		Width:       width,
		Height:      height,
		HelpVisible: false,
	}
}

// Init initializes the chat interface.
func (m *ChatModel) Init() tea.Cmd {
	return nil
}

// Update handles chat interface updates.
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Messages.Width = msg.Width
		m.Messages.Height = msg.Height - 6
		m.Input.Width = msg.Width - 4
		m.Status.Width = msg.Width

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlH:
			m.HelpVisible = !m.HelpVisible
		case tea.KeyEnter:
			if m.State == ChatStateIdle && m.Input.Value != "" {
				// Add user message
				m.Messages.AddMessage(MessageModel{
					Role:    "user",
					Content: []ContentBlock{{Type: "text", Text: m.Input.Value}},
				})
				m.Input.Clear()
				m.State = ChatStateProcessing
			}
		default:
			if m.State == ChatStateIdle {
				m.Input.Update(msg)
			}
		}
	}

	// Update spinner if processing
	if m.State == ChatStateProcessing {
		m.Spinner.Update()
	}

	return m, tea.Batch(cmds...)
}

// View renders the chat interface.
func (m *ChatModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(chatHeaderStyle.Render("Claude Code") + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.Width)) + "\n")

	// Messages area
	b.WriteString(m.Messages.View())

	// Status bar or processing indicator
	if m.State == ChatStateProcessing {
		b.WriteString(m.Spinner.ViewWithText("Thinking...") + "\n")
	} else if m.State == ChatStateError {
		b.WriteString(errorStyle.Render(m.Error.Error()) + "\n")
	}

	// Input area
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.Width)) + "\n")
	b.WriteString(m.Input.View() + "\n")

	// Footer with help
	footerText := "Ctrl+C: quit | Ctrl+H: help | Enter: send"
	if m.HelpVisible {
		footerText = "↑↓: scroll | Ctrl+C: quit | Ctrl+H: hide help | Enter: send"
	}
	b.WriteString(chatFooterStyle.Render(footerText))

	return chatContainerStyle.Render(b.String())
}

// AddAssistantMessage adds an assistant message.
func (m *ChatModel) AddAssistantMessage(content string) {
	m.Messages.AddMessage(MessageModel{
		Role:    "assistant",
		Content: []ContentBlock{{Type: "text", Text: content}},
	})
	m.State = ChatStateIdle
}

// AddSystemMessage adds a system message.
func (m *ChatModel) AddSystemMessage(content string) {
	m.Messages.AddMessage(MessageModel{
		Role:    "system",
		Content: []ContentBlock{{Type: "text", Text: content}},
	})
}

// AddToolResult adds a tool result message.
func (m *ChatModel) AddToolResult(toolName, content string) {
	m.Messages.AddMessage(MessageModel{
		Role:    "tool_result",
		Content: []ContentBlock{{Type: "tool_result", Text: fmt.Sprintf("%s: %s", toolName, content)}},
	})
}

// SetError sets an error state.
func (m *ChatModel) SetError(err error) {
	m.Error = err
	m.State = ChatStateError
}

// ClearError clears any error.
func (m *ChatModel) ClearError() {
	m.Error = nil
	m.State = ChatStateIdle
}

// =============================================================================
// Conversation History
// =============================================================================

// ConversationHistory represents the conversation history.
type ConversationHistory struct {
	Messages []MessageModel
	Current  int
}

// NewConversationHistory creates a new conversation history.
func NewConversationHistory() *ConversationHistory {
	return &ConversationHistory{
		Messages: []MessageModel{},
		Current:  -1,
	}
}

// Add adds a message to the history.
func (h *ConversationHistory) Add(msg MessageModel) {
	h.Messages = append(h.Messages, msg)
	h.Current = len(h.Messages) - 1
}

// Previous returns the previous message.
func (h *ConversationHistory) Previous() *MessageModel {
	if h.Current > 0 {
		h.Current--
		return &h.Messages[h.Current]
	}
	return nil
}

// Next returns the next message.
func (h *ConversationHistory) Next() *MessageModel {
	if h.Current < len(h.Messages)-1 {
		h.Current++
		return &h.Messages[h.Current]
	}
	return nil
}

// Last returns the last message.
func (h *ConversationHistory) Last() *MessageModel {
	if len(h.Messages) > 0 {
		return &h.Messages[len(h.Messages)-1]
	}
	return nil
}

// =============================================================================
// Streaming Response Handler
// =============================================================================

// StreamingResponse represents a streaming response.
type StreamingResponse struct {
	Content  strings.Builder
	Done     bool
	Error    error
	OnUpdate func(string)
}

// NewStreamingResponse creates a new streaming response.
func NewStreamingResponse() *StreamingResponse {
	return &StreamingResponse{
		Content: strings.Builder{},
		Done:    false,
	}
}

// Append appends content to the response.
func (r *StreamingResponse) Append(content string) {
	r.Content.WriteString(content)
	if r.OnUpdate != nil {
		r.OnUpdate(r.Content.String())
	}
}

// Complete marks the response as complete.
func (r *StreamingResponse) Complete() {
	r.Done = true
}

// Fail marks the response as failed.
func (r *StreamingResponse) Fail(err error) {
	r.Error = err
	r.Done = true
}

// String returns the current content.
func (r *StreamingResponse) String() string {
	return r.Content.String()
}
