package components

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Tool Result Display Components
// =============================================================================

// Styles
var (
	toolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	toolErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	toolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	toolOutputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	filePathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81"))

	lineNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	diffContextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	codeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	imageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("183"))

	truncationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
)

// ToolResultDisplay renders tool results for display.
type ToolResultDisplay struct {
	ToolName    string
	ToolUseID   string
	Output      interface{}
	Error       error
	IsTruncated bool
	FilePath    string
	LineCount   int
}

// RenderToolResult renders a tool result.
func RenderToolResult(result ToolResultDisplay, width int) string {
	var b strings.Builder

	// Tool header
	icon := "✓"
	style := toolSuccessStyle
	if result.Error != nil {
		icon = "✗"
		style = toolErrorStyle
	}

	header := fmt.Sprintf("%s %s", icon, result.ToolName)
	b.WriteString(style.Render(header) + "\n")

	// File path if present
	if result.FilePath != "" {
		b.WriteString(filePathStyle.Render("📁 "+result.FilePath) + "\n")
	}

	// Output content
	if result.Error != nil {
		b.WriteString(toolErrorStyle.Render(result.Error.Error()) + "\n")
	} else {
		b.WriteString(renderToolOutput(result.Output, width-4))
	}

	// Truncation notice
	if result.IsTruncated {
		b.WriteString(truncationStyle.Render("... (output truncated, see file for full content)") + "\n")
	}

	// Line count if relevant
	if result.LineCount > 0 {
		b.WriteString(lineNumStyle.Render(fmt.Sprintf("%d lines", result.LineCount)) + "\n")
	}

	return b.String()
}

// renderToolOutput renders the output content based on type.
func renderToolOutput(output interface{}, width int) string {
	switch v := output.(type) {
	case string:
		return renderTextOutput(v, width)
	case map[string]interface{}:
		return renderJSONOutput(v, width)
	case []interface{}:
		return renderJSONArray(v, width)
	default:
		return fmt.Sprintf("%v", output)
	}
}

// renderTextOutput renders text output with syntax detection.
func renderTextOutput(text string, width int) string {
	// Detect content type and render accordingly
	if strings.HasPrefix(text, "diff --git") || strings.HasPrefix(text, "--- ") {
		return renderDiffOutput(text, width)
	}
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		// Try to pretty print JSON
		var js interface{}
		if err := json.Unmarshal([]byte(text), &js); err == nil {
			if pretty, err := json.MarshalIndent(js, "", "  "); err == nil {
				return toolOutputStyle.Render(string(pretty))
			}
		}
	}
	return toolOutputStyle.Render(wrapText(text, width))
}

// renderDiffOutput renders git diff output.
func renderDiffOutput(diff string, width int) string {
	var b strings.Builder
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			b.WriteString(diffContextStyle.Render(line) + "\n")
		case strings.HasPrefix(line, "+"):
			b.WriteString(diffAddStyle.Render(line) + "\n")
		case strings.HasPrefix(line, "-"):
			b.WriteString(diffRemoveStyle.Render(line) + "\n")
		case strings.HasPrefix(line, "@@"):
			b.WriteString(lineNumStyle.Render(line) + "\n")
		default:
			b.WriteString(toolOutputStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

// renderJSONOutput renders JSON output.
func renderJSONOutput(data map[string]interface{}, width int) string {
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return toolOutputStyle.Render(string(pretty))
}

// renderJSONArray renders a JSON array.
func renderJSONArray(data []interface{}, width int) string {
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return toolOutputStyle.Render(string(pretty))
}

// =============================================================================
// Tool Use Summary
// =============================================================================

// ToolUseSummary renders a brief summary of a tool use.
type ToolUseSummary struct {
	ToolName string
	Input    map[string]interface{}
}

// Render renders a one-line summary.
func (s *ToolUseSummary) Render() string {
	switch s.ToolName {
	case "Read":
		file, _ := s.Input["target_file"].(string)
		return fmt.Sprintf("📖 Read: %s", filepath.Base(file))
	case "Write", "Edit":
		file, _ := s.Input["file_path"].(string)
		return fmt.Sprintf("✏️ %s: %s", s.ToolName, filepath.Base(file))
	case "Bash":
		cmd, _ := s.Input["command"].(string)
		if len(cmd) > 50 {
			cmd = cmd[:50] + "..."
		}
		return fmt.Sprintf("🔧 Bash: %s", cmd)
	case "Grep":
		pattern, _ := s.Input["pattern"].(string)
		return fmt.Sprintf("🔍 Grep: %s", pattern)
	case "Glob":
		pattern, _ := s.Input["glob_pattern"].(string)
		return fmt.Sprintf("📁 Glob: %s", pattern)
	default:
		return fmt.Sprintf("🔧 %s", s.ToolName)
	}
}

// =============================================================================
// File Preview
// =============================================================================

// FilePreview renders a file preview.
type FilePreview struct {
	Path      string
	Content   string
	StartLine int
	EndLine   int
	Language  string
}

// Render renders a file preview with line numbers.
func (p *FilePreview) Render(width int) string {
	var b strings.Builder

	// Header
	b.WriteString(filePathStyle.Render("📄 "+p.Path) + "\n")

	// Content with line numbers
	lines := strings.Split(p.Content, "\n")
	lineNumWidth := len(fmt.Sprintf("%d", p.EndLine))

	for i, line := range lines {
		lineNum := p.StartLine + i
		if lineNum > p.EndLine {
			break
		}

		lineNumStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)
		b.WriteString(lineNumStyle.Render(lineNumStr+"│") + " " + line + "\n")
	}

	return codeStyle.Render(b.String())
}

// =============================================================================
// Image Preview
// =============================================================================

// ImagePreview renders an image preview placeholder.
type ImagePreview struct {
	Path   string
	Width  int
	Height int
	Format string
}

// Render renders an image preview.
func (p *ImagePreview) Render() string {
	var b strings.Builder

	b.WriteString(imageStyle.Render("🖼️ Image: "+p.Path) + "\n")
	b.WriteString(imageStyle.Render(fmt.Sprintf("  %dx%d %s", p.Width, p.Height, p.Format)) + "\n")

	// ASCII art placeholder
	asciiArt := `
    ┌─────────────────────┐
    │                     │
    │     [ Image ]       │
    │                     │
    └─────────────────────┘
`
	b.WriteString(imageStyle.Render(asciiArt))

	return b.String()
}
