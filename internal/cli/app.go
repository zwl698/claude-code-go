package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"claude-code/internal/commands"
	"claude-code/internal/query"
	"claude-code/internal/tools"
	"claude-code/internal/types"
	"claude-code/internal/ui"
)

// App represents the CLI application.
type App struct {
	config        *Config
	registry      *commands.Registry
	toolRegistry  *tools.Registry
	queryEngine   *query.QueryEngine
	uiModel       *ui.Model
	ctx           context.Context
	cancel        context.CancelFunc
	initialPrompt string
	version       string
}

// Config holds CLI configuration.
type Config struct {
	Debug          bool
	Verbose        bool
	PrintMode      bool
	Model          string
	PermissionMode string
	Cwd            string
	MaxTurns       int
}

// NewApp creates a new CLI application.
func NewApp(config *Config, version string) *App {
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		version: version,
	}
}

// Initialize sets up all application components.
func (a *App) Initialize() error {
	// Get current working directory
	cwd := a.config.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Initialize command registry
	a.registry = commands.NewRegistry()
	a.registerCommands()

	// Initialize tool registry
	a.toolRegistry = tools.NewToolRegistry()
	a.registerTools()

	// Initialize query engine
	queryConfig := query.QueryEngineConfig{
		Cwd:      cwd,
		Tools:    a.toolRegistry.List(),
		MaxTurns: a.config.MaxTurns,
		GetAppState: func() *types.AppState {
			return &types.AppState{
				MainLoopModel: a.config.Model,
			}
		},
	}

	if a.config.Model != "" {
		queryConfig.UserSpecifiedModel = a.config.Model
	}

	a.queryEngine = query.NewQueryEngine(queryConfig)

	return nil
}

// Run starts the application.
func (a *App) Run(initialPrompt string) error {
	a.initialPrompt = initialPrompt

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		a.cancel()
	}()

	if a.config.PrintMode {
		return a.runPrintMode()
	}
	return a.runInteractiveMode()
}

// runPrintMode runs in non-interactive mode.
func (a *App) runPrintMode() error {
	if a.initialPrompt == "" {
		return fmt.Errorf("no prompt provided in print mode")
	}

	// Process the prompt
	output, err := a.queryEngine.SubmitMessage(a.ctx, a.initialPrompt)
	if err != nil {
		return fmt.Errorf("failed to process prompt: %w", err)
	}

	// Collect and display results
	for msg := range output {
		switch m := msg.(type) {
		case query.SDKMessage:
			a.printSDKMessage(m)
		case query.ResultMessage:
			a.printResultMessage(m)
		}
	}

	return nil
}

// runInteractiveMode runs the interactive UI.
func (a *App) runInteractiveMode() error {
	// Initialize UI model
	model := ui.InitialModel()
	a.uiModel = &model

	// Add welcome message
	a.uiModel.AddMessage("assistant", "Welcome to Claude Code! How can I help you today?")

	// Add initial prompt if provided
	if a.initialPrompt != "" {
		a.uiModel.AddMessage("user", a.initialPrompt)
		go a.processUserInput(a.initialPrompt)
	}

	// Create and run the tea program
	p := tea.NewProgram(a.uiModel, tea.WithAltScreen())

	// Handle UI events in a goroutine
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				p.Quit()
				return
			}
		}
	}()

	// Start the UI
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running UI: %w", err)
	}

	// Handle final state
	if m, ok := finalModel.(ui.Model); ok {
		if m.Err != nil {
			return m.Err
		}
	}

	return nil
}

// processUserInput handles user input.
func (a *App) processUserInput(input string) {
	// Check for slash commands
	if strings.HasPrefix(input, "/") {
		cmdName, args := commands.ParseCommand(input)
		cmd, ok := a.registry.Get(cmdName)
		if ok {
			ctx := &commands.CommandContext{
				Cwd:           a.config.Cwd,
				Args:          args,
				IsInteractive: !a.config.PrintMode,
				Verbose:       a.config.Verbose,
			}

			result, err := cmd.Execute(a.ctx, args, ctx)
			if err != nil {
				a.uiModel.AddMessage("system", fmt.Sprintf("Error: %v", err))
				return
			}

			if result.Type == "text" {
				a.uiModel.AddMessage("system", result.Value)
			}
			return
		}
	}

	// Process as a query
	output, err := a.queryEngine.SubmitMessage(a.ctx, input)
	if err != nil {
		a.uiModel.AddMessage("system", fmt.Sprintf("Error: %v", err))
		return
	}

	// Process results
	for msg := range output {
		switch m := msg.(type) {
		case query.SDKMessage:
			a.handleSDKMessage(m)
		case query.ResultMessage:
			a.handleResultMessage(m)
		}
	}
}

// handleSDKMessage handles SDK messages.
func (a *App) handleSDKMessage(msg query.SDKMessage) {
	switch msg.Type {
	case "user":
		// User message already displayed
	case "assistant":
		// Handle assistant response
		if response, ok := msg.Message.(*types.Message); ok {
			a.uiModel.AddMessage("assistant", string(response.Content))
		} else if responseMap, ok := msg.Message.(map[string]interface{}); ok {
			if content, ok := responseMap["content"]; ok {
				if contentSlice, ok := content.([]interface{}); ok {
					var textParts []string
					for _, c := range contentSlice {
						if cMap, ok := c.(map[string]interface{}); ok {
							if text, ok := cMap["text"].(string); ok {
								textParts = append(textParts, text)
							}
						}
					}
					a.uiModel.AddMessage("assistant", strings.Join(textParts, "\n"))
				}
			}
		}
	case "tool_result":
		// Tool result
		if m, ok := msg.Message.(map[string]interface{}); ok {
			toolName, _ := m["tool_name"].(string)
			content, _ := m["content"].(string)
			a.uiModel.AddMessage("system", fmt.Sprintf("[Tool: %s]\n%s", toolName, content))
		}
	case "system":
		// System message
		if m, ok := msg.Message.(map[string]interface{}); ok {
			if subtype, ok := m["subtype"].(string); ok {
				switch subtype {
				case "error":
					errMsg, _ := m["error"].(string)
					a.uiModel.AddMessage("system", fmt.Sprintf("Error: %s", errMsg))
				case "interrupted":
					a.uiModel.AddMessage("system", "Operation interrupted")
				}
			}
		}
	}
}

// handleResultMessage handles result messages.
func (a *App) handleResultMessage(msg query.ResultMessage) {
	if msg.IsError {
		a.uiModel.AddMessage("system", fmt.Sprintf("Error: %s (subtype: %s)", msg.Result, msg.Subtype))
	} else {
		a.uiModel.AddMessage("system", fmt.Sprintf("Completed in %d turns, %.2fs", msg.NumTurns, float64(msg.DurationMs)/1000))
	}
}

// printSDKMessage prints an SDK message in print mode.
func (a *App) printSDKMessage(msg query.SDKMessage) {
	data, _ := json.MarshalIndent(msg, "", "  ")
	fmt.Println(string(data))
}

// printResultMessage prints a result message in print mode.
func (a *App) printResultMessage(msg query.ResultMessage) {
	fmt.Printf("\n--- Result ---\n")
	fmt.Printf("Status: %s\n", msg.Subtype)
	fmt.Printf("Duration: %.2fs\n", float64(msg.DurationMs)/1000)
	fmt.Printf("Turns: %d\n", msg.NumTurns)
	fmt.Printf("Cost: $%.6f\n", msg.TotalCostUsd)
	fmt.Printf("Tokens: %d input, %d output\n", msg.Usage.InputTokens, msg.Usage.OutputTokens)
}

// registerCommands registers all built-in commands.
func (a *App) registerCommands() {
	a.registry.Register(commands.NewHelpCommand(a.registry))
	a.registry.Register(commands.NewExitCommand())
	a.registry.Register(commands.NewClearCommand())
	a.registry.Register(commands.NewModelCommand())
	a.registry.Register(commands.NewConfigCommand())
	a.registry.Register(commands.NewCostCommand())
	a.registry.Register(commands.NewThemeCommand())
}

// registerTools registers all built-in tools.
func (a *App) registerTools() {
	a.toolRegistry.Register(tools.NewBashTool())
	a.toolRegistry.Register(tools.NewFileReadTool())
	a.toolRegistry.Register(tools.NewFileWriteTool())
	a.toolRegistry.Register(tools.NewGlobTool())
	a.toolRegistry.Register(tools.NewGrepTool())
}

// Shutdown cleans up resources.
func (a *App) Shutdown() {
	if a.cancel != nil {
		a.cancel()
	}
}

// RunWithPrompt runs the app with a specific prompt (for scripting).
func (a *App) RunWithPrompt(prompt string) (string, error) {
	output, err := a.queryEngine.SubmitMessage(a.ctx, prompt)
	if err != nil {
		return "", err
	}

	var results []string
	for msg := range output {
		switch m := msg.(type) {
		case query.ResultMessage:
			results = append(results, m.Result)
		case query.SDKMessage:
			if m.Type == "assistant" {
				if response, ok := m.Message.(*types.Message); ok {
					results = append(results, string(response.Content))
				}
			}
		}
	}

	return strings.Join(results, "\n"), nil
}
