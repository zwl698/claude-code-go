package utils

import (
	"strings"
)

// =============================================================================
// Command Spec Types
// =============================================================================

// CommandSpec represents a command specification for prefix extraction.
type CommandSpec struct {
	// Name is the command name.
	Name string `json:"name"`

	// Description is an optional description.
	Description string `json:"description,omitempty"`

	// Subcommands are nested subcommand specifications.
	Subcommands []CommandSpec `json:"subcommands,omitempty"`

	// Args are the argument specifications.
	Args []Argument `json:"args,omitempty"`

	// Options are the option/flag specifications.
	Options []Option `json:"options,omitempty"`
}

// Argument represents a command argument specification.
type Argument struct {
	// Name is the argument name.
	Name string `json:"name,omitempty"`

	// Description is an optional description.
	Description string `json:"description,omitempty"`

	// IsDangerous indicates if this argument could be used for injection.
	IsDangerous bool `json:"isDangerous,omitempty"`

	// IsVariadic indicates if this argument repeats infinitely (e.g., echo hello world).
	IsVariadic bool `json:"isVariadic,omitempty"`

	// IsOptional indicates if this argument is optional.
	IsOptional bool `json:"isOptional,omitempty"`

	// IsCommand indicates this argument is a wrapper command (e.g., timeout, sudo).
	IsCommand bool `json:"isCommand,omitempty"`

	// IsModule indicates this argument is a module (for python -m and similar).
	IsModule interface{} `json:"isModule,omitempty"` // string | bool

	// IsScript indicates this argument is a script file.
	IsScript bool `json:"isScript,omitempty"`
}

// Option represents a command option/flag specification.
type Option struct {
	// Name is the option name (can be multiple names for aliases).
	Name interface{} `json:"name"` // string | []string

	// Description is an optional description.
	Description string `json:"description,omitempty"`

	// Args are the option argument specifications.
	Args []Argument `json:"args,omitempty"`

	// IsRequired indicates if this option is required.
	IsRequired bool `json:"isRequired,omitempty"`
}

// =============================================================================
// Built-in Command Specs
// =============================================================================

// BuiltInCommandSpecs contains the built-in command specifications.
var BuiltInCommandSpecs = []CommandSpec{
	{
		Name:        "timeout",
		Description: "Run a command with a time limit",
		Args: []Argument{
			{Name: "duration", Description: "Duration to wait before timing out", IsOptional: false},
			{Name: "command", Description: "Command to run", IsCommand: true},
		},
	},
	{
		Name:        "sleep",
		Description: "Delay for a specified amount of time",
		Args: []Argument{
			{Name: "duration", Description: "Duration to sleep", IsOptional: false},
		},
	},
	{
		Name:        "alias",
		Description: "Create or display command aliases",
	},
	{
		Name:        "nohup",
		Description: "Run a command immune to hangups",
		Args: []Argument{
			{Name: "command", Description: "Command to run", IsCommand: true},
		},
	},
	{
		Name:        "pyright",
		Description: "Python type checker",
	},
	{
		Name:        "time",
		Description: "Time command execution",
		Args: []Argument{
			{Name: "command", Description: "Command to time", IsCommand: true},
		},
	},
	{
		Name:        "srun",
		Description: "Slurm run command",
	},
	// Additional common commands
	{
		Name:        "sudo",
		Description: "Execute a command as another user",
		Args: []Argument{
			{Name: "command", Description: "Command to run", IsCommand: true},
		},
	},
	{
		Name:        "exec",
		Description: "Replace shell with command",
		Args: []Argument{
			{Name: "command", Description: "Command to execute", IsCommand: true},
		},
	},
	{
		Name:        "nice",
		Description: "Run with modified scheduling priority",
		Args: []Argument{
			{Name: "command", Description: "Command to run", IsCommand: true},
		},
	},
	{
		Name:        "env",
		Description: "Run a command in a modified environment",
		Args: []Argument{
			{Name: "command", Description: "Command to run", IsCommand: true},
		},
	},
	{
		Name:        "xargs",
		Description: "Build and execute command lines from stdin",
	},
	{
		Name:        "find",
		Description: "Search for files in a directory hierarchy",
	},
	{
		Name:        "grep",
		Description: "Print lines matching a pattern",
		Args: []Argument{
			{Name: "pattern", Description: "Pattern to search for", IsOptional: false},
			{Name: "file", Description: "Files to search", IsVariadic: true},
		},
	},
	{
		Name:        "cat",
		Description: "Concatenate and print files",
		Args: []Argument{
			{Name: "file", Description: "Files to concatenate", IsVariadic: true},
		},
	},
	{
		Name:        "echo",
		Description: "Display a line of text",
		Args: []Argument{
			{Name: "string", Description: "Strings to echo", IsVariadic: true},
		},
	},
	{
		Name:        "git",
		Description: "Git version control",
		Subcommands: []CommandSpec{
			{Name: "status"},
			{Name: "diff"},
			{Name: "log"},
			{Name: "show"},
			{Name: "branch"},
			{Name: "checkout"},
			{Name: "commit"},
			{Name: "add"},
			{Name: "rm"},
			{Name: "mv"},
			{Name: "push"},
			{Name: "pull"},
			{Name: "fetch"},
			{Name: "merge"},
			{Name: "rebase"},
			{Name: "stash"},
			{Name: "clone"},
			{Name: "init"},
			{Name: "remote"},
			{Name: "config"},
		},
	},
	{
		Name:        "npm",
		Description: "Node package manager",
		Subcommands: []CommandSpec{
			{Name: "install"},
			{Name: "uninstall"},
			{Name: "update"},
			{Name: "run"},
			{Name: "test"},
			{Name: "build"},
			{Name: "start"},
			{Name: "stop"},
		},
	},
	{
		Name:        "docker",
		Description: "Docker container manager",
		Subcommands: []CommandSpec{
			{Name: "run"},
			{Name: "build"},
			{Name: "ps"},
			{Name: "images"},
			{Name: "exec"},
			{Name: "logs"},
			{Name: "stop"},
			{Name: "rm"},
			{Name: "rmi"},
			{Name: "push"},
			{Name: "pull"},
		},
	},
	{
		Name:        "kubectl",
		Description: "Kubernetes command-line tool",
		Subcommands: []CommandSpec{
			{Name: "get"},
			{Name: "describe"},
			{Name: "create"},
			{Name: "apply"},
			{Name: "delete"},
			{Name: "logs"},
			{Name: "exec"},
			{Name: "port-forward"},
		},
	},
	{
		Name:        "python",
		Description: "Python interpreter",
		Options: []Option{
			{Name: "-c", Args: []Argument{{Name: "command", IsCommand: true}}},
			{Name: "-m", Args: []Argument{{Name: "module", IsModule: true}}},
		},
	},
	{
		Name:        "python3",
		Description: "Python 3 interpreter",
		Options: []Option{
			{Name: "-c", Args: []Argument{{Name: "command", IsCommand: true}}},
			{Name: "-m", Args: []Argument{{Name: "module", IsModule: true}}},
		},
	},
	{
		Name:        "pytest",
		Description: "Python test framework",
	},
	{
		Name:        "go",
		Description: "Go programming language",
		Subcommands: []CommandSpec{
			{Name: "build"},
			{Name: "run"},
			{Name: "test"},
			{Name: "mod"},
			{Name: "get"},
			{Name: "fmt"},
			{Name: "vet"},
			{Name: "doc"},
		},
	},
	{
		Name:        "cargo",
		Description: "Rust package manager",
		Subcommands: []CommandSpec{
			{Name: "build"},
			{Name: "run"},
			{Name: "test"},
			{Name: "doc"},
			{Name: "publish"},
		},
	},
}

// =============================================================================
// Spec Registry
// =============================================================================

// SpecRegistry maintains a registry of command specifications.
type SpecRegistry struct {
	specs map[string]CommandSpec
}

// NewSpecRegistry creates a new specification registry with built-in specs.
func NewSpecRegistry() *SpecRegistry {
	registry := &SpecRegistry{
		specs: make(map[string]CommandSpec),
	}

	// Register built-in specs
	for _, spec := range BuiltInCommandSpecs {
		registry.specs[strings.ToLower(spec.Name)] = spec
	}

	return registry
}

// GetSpec retrieves a command specification by name.
func (r *SpecRegistry) GetSpec(command string) *CommandSpec {
	spec, ok := r.specs[strings.ToLower(command)]
	if !ok {
		return nil
	}
	return &spec
}

// RegisterSpec adds a command specification to the registry.
func (r *SpecRegistry) RegisterSpec(spec CommandSpec) {
	r.specs[strings.ToLower(spec.Name)] = spec
}

// IsKnownSubcommand checks if an argument matches a known subcommand.
func (r *SpecRegistry) IsKnownSubcommand(arg string, spec *CommandSpec) bool {
	if spec == nil || len(spec.Subcommands) == 0 {
		return false
	}

	argLower := strings.ToLower(arg)
	for _, sub := range spec.Subcommands {
		if strings.ToLower(sub.Name) == argLower {
			return true
		}
	}
	return false
}

// FlagTakesArg checks if a flag takes an argument based on spec.
func (r *SpecRegistry) FlagTakesArg(flag string, nextArg string, spec *CommandSpec) bool {
	if spec == nil {
		return false
	}

	// Check if flag is in spec options
	for _, opt := range spec.Options {
		optNames := getOptionNames(opt)
		for _, name := range optNames {
			if name == flag {
				return len(opt.Args) > 0
			}
		}
	}

	// Heuristic: if next arg isn't a flag and isn't a known subcommand, assume it's a flag value
	if len(spec.Subcommands) > 0 && nextArg != "" && !strings.HasPrefix(nextArg, "-") {
		return !r.IsKnownSubcommand(nextArg, spec)
	}

	return false
}

// getOptionNames extracts option names from an Option.
func getOptionNames(opt Option) []string {
	switch v := opt.Name.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, name := range v {
			if s, ok := name.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// FindFirstSubcommand finds the first subcommand by skipping flags and their values.
func (r *SpecRegistry) FindFirstSubcommand(args []string, spec *CommandSpec) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}

		if strings.HasPrefix(arg, "-") {
			if r.FlagTakesArg(arg, safeGet(args, i+1), spec) {
				i++ // Skip the flag argument
			}
			continue
		}

		if spec == nil || len(spec.Subcommands) == 0 {
			return arg
		}

		if r.IsKnownSubcommand(arg, spec) {
			return arg
		}
	}
	return ""
}

// safeGet safely retrieves an element from a slice.
func safeGet(args []string, i int) string {
	if i >= 0 && i < len(args) {
		return args[i]
	}
	return ""
}

// =============================================================================
// Spec-based Prefix Building
// =============================================================================

// BuildPrefixWithSpec builds a command prefix using a command specification.
func BuildPrefixWithSpec(command string, args []string, spec *CommandSpec) string {
	maxDepth := calculateDepthWithSpec(command, args, spec)
	parts := []string{command}
	hasSubcommands := spec != nil && len(spec.Subcommands) > 0
	foundSubcommand := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" || len(parts) >= maxDepth {
			break
		}

		if strings.HasPrefix(arg, "-") {
			// Special case: python -c should stop after -c
			if arg == "-c" && (strings.ToLower(command) == "python" || strings.ToLower(command) == "python3") {
				break
			}

			// Check for isCommand/isModule flags that should be included in prefix
			if spec != nil {
				for _, opt := range spec.Options {
					optNames := getOptionNames(opt)
					for _, name := range optNames {
						if name == arg && len(opt.Args) > 0 {
							for _, argSpec := range opt.Args {
								if argSpec.IsCommand || argSpec.IsModule != nil {
									parts = append(parts, arg)
									continue
								}
							}
						}
					}
				}
			}

			// For commands with subcommands, skip global flags to find the subcommand
			if hasSubcommands && !foundSubcommand {
				if globalSpecRegistry.FlagTakesArg(arg, safeGet(args, i+1), spec) {
					i++
				}
				continue
			}
			break // Stop at flags (original behavior)
		}

		if shouldStopAtArgForSpec(arg, args[:i], spec) {
			break
		}

		if hasSubcommands && !foundSubcommand {
			foundSubcommand = globalSpecRegistry.IsKnownSubcommand(arg, spec)
		}
		parts = append(parts, arg)
	}

	return strings.Join(parts, " ")
}

// calculateDepthWithSpec calculates the maximum depth for prefix extraction.
func calculateDepthWithSpec(command string, args []string, spec *CommandSpec) int {
	// Get global registry
	registry := globalSpecRegistry

	// Find first subcommand by skipping flags and their values
	firstSubcommand := registry.FindFirstSubcommand(args, spec)

	commandLower := strings.ToLower(command)
	key := commandLower
	if firstSubcommand != "" {
		key = commandLower + " " + strings.ToLower(firstSubcommand)
	}

	// Check depth rules
	if depth, ok := DepthRules[key]; ok {
		return depth
	}
	if depth, ok := DepthRules[commandLower]; ok {
		return depth
	}

	if spec == nil {
		return 2
	}

	// Check for options with isCommand/isModule
	if len(spec.Options) > 0 {
		for _, arg := range args {
			if !strings.HasPrefix(arg, "-") {
				continue
			}
			for _, opt := range spec.Options {
				optNames := getOptionNames(opt)
				for _, name := range optNames {
					if name == arg && len(opt.Args) > 0 {
						for _, argSpec := range opt.Args {
							if argSpec.IsCommand || argSpec.IsModule != nil {
								return 3
							}
						}
					}
				}
			}
		}
	}

	// Check subcommand spec
	if firstSubcommand != "" && len(spec.Subcommands) > 0 {
		for _, sub := range spec.Subcommands {
			if strings.ToLower(sub.Name) == strings.ToLower(firstSubcommand) {
				if len(sub.Args) > 0 {
					for _, argSpec := range sub.Args {
						if argSpec.IsCommand {
							return 3
						}
						if argSpec.IsVariadic {
							return 2
						}
					}
				}
				if len(sub.Subcommands) > 0 {
					return 4
				}
				// Leaf subcommand with NO args declared
				if len(sub.Args) == 0 {
					return 2
				}
				return 3
			}
		}
	}

	// Check main args
	if len(spec.Args) > 0 {
		for _, argSpec := range spec.Args {
			if argSpec.IsCommand {
				return 2
			}
		}

		if len(spec.Subcommands) == 0 {
			for _, argSpec := range spec.Args {
				if argSpec.IsVariadic {
					return 1
				}
			}
			if spec.Args[0].Name != "" && !spec.Args[0].IsOptional {
				return 2
			}
		}
	}

	// Check for dangerous args
	for _, argSpec := range spec.Args {
		if argSpec.IsDangerous {
			return 3
		}
	}

	return 2
}

// shouldStopAtArgForSpec determines if we should stop at this argument.
func shouldStopAtArgForSpec(arg string, previousArgs []string, spec *CommandSpec) bool {
	if strings.HasPrefix(arg, "-") {
		return true
	}

	// Check for URL protocols
	for _, proto := range []string{"http://", "https://", "ftp://"} {
		if strings.HasPrefix(arg, proto) {
			return true
		}
	}

	// Check for file extension
	dotIndex := strings.LastIndex(arg, ".")
	hasExtension := dotIndex > 0 && dotIndex < len(arg)-1 && !strings.Contains(arg[dotIndex+1:], ":")
	hasFile := strings.Contains(arg, "/") || hasExtension
	hasURL := false
	for _, proto := range []string{"http://", "https://", "ftp://"} {
		if strings.HasPrefix(arg, proto) {
			hasURL = true
			break
		}
	}

	if !hasFile && !hasURL {
		return false
	}

	// Check if we're after a -m flag for python modules
	if spec != nil && len(spec.Options) > 0 && len(previousArgs) > 0 && previousArgs[len(previousArgs)-1] == "-m" {
		for _, opt := range spec.Options {
			optNames := getOptionNames(opt)
			for _, name := range optNames {
				if name == "-m" && len(opt.Args) > 0 {
					for _, argSpec := range opt.Args {
						if argSpec.IsModule != nil {
							return false // Don't stop at module names
						}
					}
				}
			}
		}
	}

	// For actual files/URLs, always stop
	return true
}

// Global spec registry instance
var globalSpecRegistry = NewSpecRegistry()

// GetSpecRegistry returns the global spec registry.
func GetSpecRegistry() *SpecRegistry {
	return globalSpecRegistry
}
