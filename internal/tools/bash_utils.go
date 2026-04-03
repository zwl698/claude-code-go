package tools

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// =============================================================================
// Bash Tool Utilities
// =============================================================================

const (
	MaxOutputLength   = 50000
	MaxImageFileSize  = 20 * 1024 * 1024 // 20 MB
	MaxCommandTimeout = 600              // seconds
)

// BashCommandOptions represents options for executing a bash command.
type BashCommandOptions struct {
	Command     string
	WorkingDir  string
	Timeout     int
	Env         []string
	Interactive bool
}

// BashCommandResult represents the result of a bash command.
type BashCommandResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Interrupted bool
	IsImage     bool
	ImageData   string
	ImageType   string
}

// ExecuteBashCommand executes a bash command with security checks.
func ExecuteBashCommand(opts BashCommandOptions) (*BashCommandResult, error) {
	// Validate command
	if opts.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Security check
	if err := ValidateBashCommand(opts.Command); err != nil {
		return nil, err
	}

	// Set working directory
	workingDir := opts.WorkingDir
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	// Get shell path
	shell := GetShellPath()

	// Create command
	cmd := exec.Command(shell, "-c", opts.Command)
	cmd.Dir = workingDir

	// Set environment
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	// Execute command
	output, err := cmd.CombinedOutput()

	result := &BashCommandResult{
		Stdout:   string(output),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	// Check for image output
	if IsImageOutput(result.Stdout) {
		result.IsImage = true
		parsed := ParseDataUri(result.Stdout)
		if parsed != nil {
			result.ImageData = parsed.Data
			result.ImageType = parsed.MediaType
		}
	}

	return result, nil
}

// ValidateBashCommand validates a bash command for security.
func ValidateBashCommand(command string) error {
	// Check for dangerous patterns
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{"rm -rf /", "attempting to delete root filesystem"},
		{"rm -rf /*", "attempting to delete root filesystem"},
		{":(){ :|:& };:", "fork bomb detected"},
		{"mkfs", "attempting to format filesystem"},
		{"dd if=/dev/zero", "attempting to overwrite disk"},
		{"> /dev/sda", "attempting to write directly to disk"},
		{"chmod -R 777 /", "attempting to change permissions on root"},
		{"chown -R", "attempting to change ownership recursively"},
	}

	for _, dp := range dangerousPatterns {
		if strings.Contains(command, dp.pattern) {
			return fmt.Errorf("dangerous command blocked: %s", dp.message)
		}
	}

	return nil
}

// =============================================================================
// Output Formatting
// =============================================================================

// FormatOutput formats the output with truncation if needed.
func FormatOutput(content string) (totalLines int, truncatedContent string, isImage bool) {
	isImage = IsImageOutput(content)
	if isImage {
		return 1, content, true
	}

	if len(content) <= MaxOutputLength {
		return CountLines(content), content, false
	}

	truncated := content[:MaxOutputLength]
	remainingLines := CountLines(content[MaxOutputLength:])
	result := fmt.Sprintf("%s\n\n... [%d lines truncated] ...", truncated, remainingLines)

	return CountLines(content), result, false
}

// StripEmptyLines strips leading and trailing empty lines.
func StripEmptyLines(content string) string {
	lines := strings.Split(content, "\n")

	// Find first non-empty line
	startIndex := 0
	for startIndex < len(lines) && strings.TrimSpace(lines[startIndex]) == "" {
		startIndex++
	}

	// Find last non-empty line
	endIndex := len(lines) - 1
	for endIndex >= 0 && strings.TrimSpace(lines[endIndex]) == "" {
		endIndex--
	}

	// If all lines are empty, return empty string
	if startIndex > endIndex {
		return ""
	}

	return strings.Join(lines[startIndex:endIndex+1], "\n")
}

// =============================================================================
// Image Handling
// =============================================================================

// IsImageOutput checks if content is a base64 encoded image data URL.
func IsImageOutput(content string) bool {
	matched, _ := regexp.MatchString(`^data:image/[a-z0-9.+_-]+;base64,`, content)
	return matched
}

// ParsedDataUri represents a parsed data URI.
type ParsedDataUri struct {
	MediaType string
	Data      string
}

// ParseDataUri parses a data-URI string.
func ParseDataUri(s string) *ParsedDataUri {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`^data:([^;]+);base64,(.+)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil
	}
	return &ParsedDataUri{
		MediaType: matches[1],
		Data:      matches[2],
	}
}

// BuildImageToolResult builds an image tool result from stdout.
func BuildImageToolResult(stdout string, toolUseID string) map[string]interface{} {
	parsed := ParseDataUri(stdout)
	if parsed == nil {
		return nil
	}

	return map[string]interface{}{
		"tool_use_id": toolUseID,
		"type":        "tool_result",
		"content": []map[string]interface{}{
			{
				"type": "image",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": parsed.MediaType,
					"data":       parsed.Data,
				},
			},
		},
	}
}

// =============================================================================
// Content Summary
// =============================================================================

// CreateContentSummary creates a summary of content blocks.
func CreateContentSummary(content []map[string]interface{}) string {
	textCount := 0
	imageCount := 0
	var previews []string

	for _, block := range content {
		if block["type"] == "image" {
			imageCount++
		} else if block["type"] == "text" {
			if text, ok := block["text"].(string); ok {
				textCount++
				preview := text
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				previews = append(previews, preview)
			}
		}
	}

	var summary []string
	if imageCount > 0 {
		summary = append(summary, fmt.Sprintf("[%d image(s)]", imageCount))
	}
	if textCount > 0 {
		summary = append(summary, fmt.Sprintf("[%d text block(s)]", textCount))
	}

	result := "MCP Result: " + strings.Join(summary, ", ")
	if len(previews) > 0 {
		result += "\n\n" + strings.Join(previews, "\n\n")
	}
	return result
}

// =============================================================================
// Path Utilities
// =============================================================================

// GetShellPath returns the path to the shell.
func GetShellPath() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/bash"
}

// ExpandPath expands environment variables and ~ in a path.
func ExpandPath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// Expand environment variables
	return os.ExpandEnv(path)
}

// IsValidPath checks if a path is valid and safe.
func IsValidPath(path string) error {
	// Expand the path
	path = ExpandPath(path)

	// Check for path traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected")
	}

	// Check if path exists
	_, err := os.Stat(cleanPath)
	return err
}

// ResetCwdIfOutsideProject resets the current working directory if outside project.
func ResetCwdIfOutsideProject(currentCwd, originalCwd string, allowedPaths []string) bool {
	// Check if current directory is in allowed paths
	for _, path := range allowedPaths {
		if strings.HasPrefix(currentCwd, path) {
			return false
		}
	}

	// Reset to original directory
	if currentCwd != originalCwd {
		return true
	}
	return false
}

// =============================================================================
// Helper Functions
// =============================================================================

// CountLines counts the number of lines in a string.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// EncodeBase64 encodes data to base64.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 data.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// SplitCommand splits a command into parts for execution.
func SplitCommand(command string) (string, []string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// IsBuiltinCommand checks if a command is a shell builtin.
func IsBuiltinCommand(command string) bool {
	builtins := map[string]bool{
		"cd":        true,
		"echo":      true,
		"exit":      true,
		"export":    true,
		"pwd":       true,
		"read":      true,
		"return":    true,
		"shift":     true,
		"test":      true,
		"true":      true,
		"false":     true,
		"alias":     true,
		"unalias":   true,
		"bg":        true,
		"fg":        true,
		"jobs":      true,
		"kill":      true,
		"wait":      true,
		"set":       true,
		"unset":     true,
		"source":    true,
		".":         true,
		"exec":      true,
		"eval":      true,
		"trap":      true,
		"type":      true,
		"hash":      true,
		"help":      true,
		"local":     true,
		"logout":    true,
		"printf":    true,
		"enable":    true,
		"mapfile":   true,
		"readarray": true,
		"caller":    true,
		"compgen":   true,
		"complete":  true,
		"compopt":   true,
		"declare":   true,
		"typeset":   true,
		"dirs":      true,
		"disown":    true,
		"getopts":   true,
		"history":   true,
		"let":       true,
		"popd":      true,
		"pushd":     true,
		"shopt":     true,
		"suspend":   true,
		"times":     true,
		"ulimit":    true,
		"umask":     true,
	}
	return builtins[command]
}

// GetCommandPrefix extracts the command prefix (the actual command).
func GetCommandPrefix(command string) string {
	// Remove leading/trailing whitespace
	command = strings.TrimSpace(command)

	// Handle pipes and redirects
	if idx := strings.IndexAny(command, "|&;<>"); idx > 0 {
		command = command[:idx]
	}

	// Get the first word
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}
