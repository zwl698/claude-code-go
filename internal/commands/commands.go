package commands

import (
	"context"
	"fmt"
	"strings"
)

// CommandResult represents the result of command execution.
type CommandResult struct {
	Type    string `json:"type"` // text | compact | skip
	Value   string `json:"value,omitempty"`
	Display string `json:"display,omitempty"` // skip | system | user
}

// CommandContext provides context for command execution.
type CommandContext struct {
	Cwd           string
	Args          string
	IsInteractive bool
	Verbose       bool
	Debug         bool
}

// CommandHandler defines the interface for command handlers.
type CommandHandler interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error)
	IsEnabled() bool
	IsHidden() bool
}

// Registry manages all available commands.
type Registry struct {
	commands map[string]CommandHandler
	aliases  map[string]string
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]CommandHandler),
		aliases:  make(map[string]string),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(handler CommandHandler) {
	r.commands[handler.Name()] = handler
}

// RegisterAlias adds an alias for a command.
func (r *Registry) RegisterAlias(alias, commandName string) {
	r.aliases[alias] = commandName
}

// Get retrieves a command by name or alias.
func (r *Registry) Get(name string) (CommandHandler, bool) {
	// Check direct name first
	if cmd, ok := r.commands[name]; ok {
		return cmd, true
	}
	// Check aliases
	if aliasTarget, ok := r.aliases[name]; ok {
		if cmd, ok := r.commands[aliasTarget]; ok {
			return cmd, true
		}
	}
	return nil, false
}

// List returns all registered commands.
func (r *Registry) List() []CommandHandler {
	result := make([]CommandHandler, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}
	return result
}

// ListEnabled returns all enabled commands.
func (r *Registry) ListEnabled() []CommandHandler {
	result := make([]CommandHandler, 0)
	for _, cmd := range r.commands {
		if cmd.IsEnabled() {
			result = append(result, cmd)
		}
	}
	return result
}

// ParseCommand parses a command string into name and arguments.
func ParseCommand(input string) (name string, args string) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 {
		return "", ""
	}
	name = strings.TrimPrefix(parts[0], "/")
	if len(parts) > 1 {
		args = parts[1]
	}
	return name, args
}

// ========================================
// Built-in Commands
// ========================================

// HelpCommand shows available commands.
type HelpCommand struct {
	registry *Registry
}

func NewHelpCommand(registry *Registry) *HelpCommand {
	return &HelpCommand{registry: registry}
}

func (c *HelpCommand) Name() string        { return "help" }
func (c *HelpCommand) Description() string { return "Show available commands" }
func (c *HelpCommand) IsEnabled() bool     { return true }
func (c *HelpCommand) IsHidden() bool      { return false }

func (c *HelpCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	var sb strings.Builder
	sb.WriteString("Available commands:\n\n")

	commands := c.registry.ListEnabled()
	for _, cmd := range commands {
		if !cmd.IsHidden() {
			sb.WriteString(fmt.Sprintf("  /%-15s %s\n", cmd.Name(), cmd.Description()))
		}
	}

	return &CommandResult{
		Type:  "text",
		Value: sb.String(),
	}, nil
}

// ExitCommand exits the application.
type ExitCommand struct{}

func NewExitCommand() *ExitCommand { return &ExitCommand{} }

func (c *ExitCommand) Name() string        { return "exit" }
func (c *ExitCommand) Description() string { return "Exit the application" }
func (c *ExitCommand) IsEnabled() bool     { return true }
func (c *ExitCommand) IsHidden() bool      { return false }

func (c *ExitCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	return &CommandResult{
		Type:  "text",
		Value: "Goodbye!",
	}, nil
}

// ClearCommand clears the screen.
type ClearCommand struct{}

func NewClearCommand() *ClearCommand { return &ClearCommand{} }

func (c *ClearCommand) Name() string        { return "clear" }
func (c *ClearCommand) Description() string { return "Clear the screen" }
func (c *ClearCommand) IsEnabled() bool     { return true }
func (c *ClearCommand) IsHidden() bool      { return false }

func (c *ClearCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	// Clear screen escape sequence
	return &CommandResult{
		Type:  "text",
		Value: "\033[2J\033[H",
	}, nil
}

// ModelCommand manages model selection.
type ModelCommand struct {
	currentModel string
}

func NewModelCommand() *ModelCommand {
	return &ModelCommand{currentModel: "claude-sonnet-4-20250514"}
}

func (c *ModelCommand) Name() string        { return "model" }
func (c *ModelCommand) Description() string { return "Switch or show the current model" }
func (c *ModelCommand) IsEnabled() bool     { return true }
func (c *ModelCommand) IsHidden() bool      { return false }

func (c *ModelCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return &CommandResult{
			Type:  "text",
			Value: fmt.Sprintf("Current model: %s", c.currentModel),
		}, nil
	}

	// Validate and set model
	validModels := []string{
		"claude-sonnet-4-20250514",
		"claude-opus-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
	}

	for _, model := range validModels {
		if args == model {
			c.currentModel = model
			return &CommandResult{
				Type:  "text",
				Value: fmt.Sprintf("Model set to: %s", model),
			}, nil
		}
	}

	return &CommandResult{
		Type:  "text",
		Value: fmt.Sprintf("Invalid model. Available: %s", strings.Join(validModels, ", ")),
	}, nil
}

// ConfigCommand manages configuration.
type ConfigCommand struct{}

func NewConfigCommand() *ConfigCommand { return &ConfigCommand{} }

func (c *ConfigCommand) Name() string        { return "config" }
func (c *ConfigCommand) Description() string { return "Manage configuration settings" }
func (c *ConfigCommand) IsEnabled() bool     { return true }
func (c *ConfigCommand) IsHidden() bool      { return false }

func (c *ConfigCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	// TODO: Implement config management
	return &CommandResult{
		Type:  "text",
		Value: "Configuration management not yet implemented",
	}, nil
}

// CostCommand shows cost tracking.
type CostCommand struct{}

func NewCostCommand() *CostCommand { return &CostCommand{} }

func (c *CostCommand) Name() string        { return "cost" }
func (c *CostCommand) Description() string { return "Show current session cost" }
func (c *CostCommand) IsEnabled() bool     { return true }
func (c *CostCommand) IsHidden() bool      { return false }

func (c *CostCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	// TODO: Implement cost tracking
	return &CommandResult{
		Type:  "text",
		Value: "Cost tracking: $0.00 (not yet implemented)",
	}, nil
}

// ThemeCommand manages theme settings.
type ThemeCommand struct {
	currentTheme string
}

func NewThemeCommand() *ThemeCommand {
	return &ThemeCommand{currentTheme: "dark"}
}

func (c *ThemeCommand) Name() string        { return "theme" }
func (c *ThemeCommand) Description() string { return "Set or show the current theme" }
func (c *ThemeCommand) IsEnabled() bool     { return true }
func (c *ThemeCommand) IsHidden() bool      { return false }

func (c *ThemeCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return &CommandResult{
			Type:  "text",
			Value: fmt.Sprintf("Current theme: %s", c.currentTheme),
		}, nil
	}

	validThemes := []string{"dark", "light", "ansi"}
	for _, theme := range validThemes {
		if args == theme {
			c.currentTheme = theme
			return &CommandResult{
				Type:  "text",
				Value: fmt.Sprintf("Theme set to: %s", theme),
			}, nil
		}
	}

	return &CommandResult{
		Type:  "text",
		Value: fmt.Sprintf("Invalid theme. Available: %s", strings.Join(validThemes, ", ")),
	}, nil
}
