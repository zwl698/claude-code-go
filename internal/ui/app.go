package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"claude-code-go/internal/query"
	"claude-code-go/internal/types"
	"claude-code-go/internal/ui/components"
)

// AppModel integrates QueryEngine with the Bubble Tea UI.
type AppModel struct {
	queryEngine  *query.QueryEngine
	chat         *components.ChatModel
	ctx          context.Context
	cancel       context.CancelFunc
	messageChan  <-chan interface{}
	currentModel string
	models       map[string]types.ThinkingConfig
	mu           sync.RWMutex
	err          error
}

// NewAppModel creates a new UI app model.
func NewAppModel(engine *query.QueryEngine, width, height int) *AppModel {
	ctx, cancel := context.WithCancel(context.Background())
	return &AppModel{
		queryEngine:  engine,
		chat:         components.NewChatModel(width, height),
		ctx:          ctx,
		cancel:       cancel,
		currentModel: "claude-sonnet-4-20250514",
		models:       make(map[string]types.ThinkingConfig),
	}
}

// Init initializes the app.
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.chat.Init(),
	)
}

// Update handles app updates.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.chat.Width = msg.Width
		m.chat.Height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancel()
			return m, tea.Quit
		case tea.KeyEnter:
			if m.chat.State == components.ChatStateIdle && m.chat.Input.Value != "" {
				prompt := m.chat.Input.Value
				m.chat.Input.Clear()
				m.chat.State = components.ChatStateProcessing

				// Submit message to query engine
				output, err := m.queryEngine.SubmitMessage(m.ctx, prompt)
				if err != nil {
					m.chat.State = components.ChatStateError
					m.chat.Error = err
					return m, nil
				}
				m.messageChan = output
				cmds = append(cmds, m.waitForMessages())
			}
		}

	case QueryEngineMsg:
		switch data := msg.Data.(type) {
		case query.SDKMessage:
			m.handleSDKMessage(data)
		case query.ResultMessage:
			m.handleResultMessage(data)
		}
	}

	// Update chat component
	newChat, cmd := m.chat.Update(msg)
	m.chat = newChat.(*components.ChatModel)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the app.
func (m *AppModel) View() string {
	return m.chat.View()
}

// QueryEngineMsg wraps messages from QueryEngine for Bubble Tea.
type QueryEngineMsg struct {
	Data interface{}
}

// waitForMessages waits for messages from the QueryEngine.
func (m *AppModel) waitForMessages() tea.Cmd {
	return func() tea.Msg {
		if m.messageChan == nil {
			return nil
		}

		data, ok := <-m.messageChan
		if !ok {
			m.chat.State = components.ChatStateIdle
			return nil
		}

		return QueryEngineMsg{Data: data}
	}
}

// handleSDKMessage handles SDK messages from QueryEngine.
func (m *AppModel) handleSDKMessage(msg query.SDKMessage) {
	switch msg.Type {
	case "assistant":
		if content, ok := msg.Message.(map[string]interface{}); ok {
			if contentBlocks, ok := content["content"].([]map[string]interface{}); ok {
				var textParts []string
				for _, block := range contentBlocks {
					if block["type"] == "text" {
						if text, ok := block["text"].(string); ok {
							textParts = append(textParts, text)
						}
					}
				}
				if len(textParts) > 0 {
					m.chat.AddAssistantMessage(strings.Join(textParts, "\n"))
				}
			}
		}

	case "tool_result":
		if data, ok := msg.Message.(map[string]interface{}); ok {
			toolName, _ := data["tool_name"].(string)
			content, _ := data["content"].(string)
			m.chat.AddToolResult(toolName, content)
		}

	case "system":
		if data, ok := msg.Message.(map[string]interface{}); ok {
			if subtype, ok := data["subtype"].(string); ok {
				switch subtype {
				case "interrupted":
					m.chat.State = components.ChatStateIdle
					m.chat.AddSystemMessage("Operation interrupted")
				case "error":
					if errMsg, ok := data["error"].(string); ok {
						m.chat.State = components.ChatStateError
						m.chat.Error = fmt.Errorf("%s", errMsg)
					}
				}
			}
		}
	}
}

// handleResultMessage handles result messages from QueryEngine.
func (m *AppModel) handleResultMessage(msg query.ResultMessage) {
	m.chat.State = components.ChatStateIdle

	if msg.IsError {
		m.chat.Error = fmt.Errorf("query failed: %s", msg.Subtype)
	} else {
		duration := fmt.Sprintf("%.2fs", float64(msg.DurationMs)/1000.0)
		cost := fmt.Sprintf("$%.6f", msg.TotalCostUsd)
		m.chat.AddSystemMessage(fmt.Sprintf("Completed in %s | Cost: %s | Turns: %d", duration, cost, msg.NumTurns))
	}
}

// RunUI runs the interactive UI.
func RunUI(engine *query.QueryEngine) error {
	p := tea.NewProgram(
		NewAppModel(engine, 80, 24),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
