package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Shell Quote Types
// =============================================================================

// ShellParseResult represents the result of parsing a shell command.
type ShellParseResult struct {
	Success bool
	Tokens  []ParseEntry
	Error   string
}

// ShellQuoteResult represents the result of quoting shell arguments.
type ShellQuoteResult struct {
	Success bool
	Quoted  string
	Error   string
}

// =============================================================================
// Shell Quote Functions
// =============================================================================

// TryQuoteShellArgs attempts to quote shell arguments safely.
func TryQuoteShellArgs(args []interface{}) ShellQuoteResult {
	var validated []string

	for i, arg := range args {
		if arg == nil {
			validated = append(validated, "nil")
			continue
		}

		switch v := arg.(type) {
		case string:
			validated = append(validated, v)
		case int, int8, int16, int32, int64:
			validated = append(validated, fmt.Sprintf("%d", v))
		case uint, uint8, uint16, uint32, uint64:
			validated = append(validated, fmt.Sprintf("%d", v))
		case float32, float64:
			validated = append(validated, fmt.Sprintf("%v", v))
		case bool:
			validated = append(validated, fmt.Sprintf("%t", v))
		default:
			return ShellQuoteResult{
				Success: false,
				Error:   fmt.Sprintf("Cannot quote argument at index %d: unsupported type %T", i, arg),
			}
		}
	}

	quoted := Quote(validated)
	return ShellQuoteResult{Success: true, Quoted: quoted}
}

// Quote quotes shell arguments safely.
// This is a simplified implementation that handles basic quoting needs.
func Quote(args []string) string {
	var quoted []string
	for _, arg := range args {
		quoted = append(quoted, quoteSingleArg(arg))
	}
	return strings.Join(quoted, " ")
}

// quoteSingleArg quotes a single shell argument.
func quoteSingleArg(arg string) string {
	// Empty string needs quotes
	if arg == "" {
		return "''"
	}

	// Check if the argument is safe without quotes
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\/\@]+$`)
	if safePattern.MatchString(arg) {
		return arg
	}

	// Use single quotes for safety, escaping any single quotes within
	escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// =============================================================================
// Malformed Token Detection
// =============================================================================

// HasMalformedTokens checks if parsed tokens contain malformed entries.
// This detects when shell-quote misinterprets the command, which can happen
// with ambiguous patterns like JSON-like strings with semicolons.
func HasMalformedTokens(command string, parsed []ParseEntry) bool {
	// Check for unterminated quotes in the original command
	if hasUnterminatedQuotes(command) {
		return true
	}

	for _, entry := range parsed {
		str, ok := entry.(StringToken)
		if !ok {
			continue
		}
		s := string(str)

		// Check for unbalanced curly braces
		if !isBalanced(s, '{', '}') {
			return true
		}

		// Check for unbalanced parentheses
		if !isBalanced(s, '(', ')') {
			return true
		}

		// Check for unbalanced square brackets
		if !isBalanced(s, '[', ']') {
			return true
		}

		// Check for unbalanced double quotes
		if !hasBalancedQuotes(s, '"') {
			return true
		}

		// Check for unbalanced single quotes
		if !hasBalancedQuotes(s, '\'') {
			return true
		}
	}

	return false
}

// hasUnterminatedQuotes checks if a command has unterminated quotes.
func hasUnterminatedQuotes(command string) bool {
	inSingle := false
	inDouble := false

	for i := 0; i < len(command); i++ {
		c := command[i]

		if c == '\\' && !inSingle && i+1 < len(command) {
			i++
			continue
		}

		if c == '"' && !inSingle {
			inDouble = !inDouble
		} else if c == '\'' && !inDouble {
			inSingle = !inSingle
		}
	}

	return inSingle || inDouble
}

// isBalanced checks if opening and closing characters are balanced.
func isBalanced(s string, open, close rune) bool {
	count := 0
	for _, c := range s {
		if c == open {
			count++
		} else if c == close {
			count--
		}
	}
	return count == 0
}

// hasBalancedQuotes checks if quotes of a given type are balanced,
// accounting for escape sequences.
func hasBalancedQuotes(s string, quoteChar rune) bool {
	count := 0
	escaped := false

	for _, c := range s {
		if escaped {
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == quoteChar {
			count++
		}
	}

	return count%2 == 0
}

// =============================================================================
// Shell-Quote Single Quote Bug Detection
// =============================================================================

// HasShellQuoteSingleQuoteBug detects commands containing patterns that exploit
// the shell-quote library's incorrect handling of backslashes inside single quotes.
//
// In bash, single quotes preserve ALL characters literally - backslash has no
// special meaning. So '\' is just the string \ (the quote opens, contains \,
// and the next ' closes it). But some parsers incorrectly treat \ as an escape
// character inside single quotes, causing '\' to NOT close the quoted string.
func HasShellQuoteSingleQuoteBug(command string) bool {
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(command); i++ {
		char := command[i]

		// Handle backslash escaping outside of single quotes
		if char == '\\' && !inSingleQuote && i+1 < len(command) {
			i++
			continue
		}

		if char == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if char == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote

			// Check if we just closed a single quote with trailing backslashes
			if !inSingleQuote {
				backslashCount := 0
				j := i - 1
				for j >= 0 && command[j] == '\\' {
					backslashCount++
					j--
				}

				// Odd trailing backslashes = always a bug
				if backslashCount > 0 && backslashCount%2 == 1 {
					return true
				}

				// Even trailing backslashes: only a bug when a later ' exists
				if backslashCount > 0 && backslashCount%2 == 0 {
					if strings.Contains(command[i+1:], "'") {
						return true
					}
				}
			}
			continue
		}
	}

	return false
}

// =============================================================================
// Safe Shell Argument Validation
// =============================================================================

// IsSafeShellArg checks if a string is safe to use as a shell argument
// without additional quoting.
func IsSafeShellArg(arg string) bool {
	if arg == "" {
		return false
	}

	// Check for shell metacharacters
	dangerousChars := []string{
		";", "|", "&", "$", "`", "(", ")", "<", ">",
		"*", "?", "[", "]", "{", "}", "~", "!", "#",
		"\n", "\r", "\t", " ", "'", `"`, "\\",
	}

	for _, char := range dangerousChars {
		if strings.Contains(arg, char) {
			return false
		}
	}

	return true
}

// EscapeShellArg escapes a shell argument if necessary.
func EscapeShellArg(arg string) string {
	if IsSafeShellArg(arg) {
		return arg
	}
	return quoteSingleArg(arg)
}

// =============================================================================
// Shell Command Validation
// =============================================================================

// IsSimpleCommand checks if a command is simple and safe to execute.
// A simple command contains no shell metacharacters or substitutions.
func IsSimpleCommand(command string) bool {
	// Check for dangerous patterns
	dangerousPatterns := []string{
		"$(", "${", "`", ";", "|", "&", "&&", "||",
		">", "<", ">>", "<<", ">>>", "2>", "2>>",
		"!", "#", "\n", "\r",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return false
		}
	}

	return true
}

// ValidateShellCommand performs comprehensive validation on a shell command.
func ValidateShellCommand(command string) error {
	// Check for empty command
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// Check for unterminated quotes
	if hasUnterminatedQuotes(command) {
		return fmt.Errorf("unterminated quotes in command")
	}

	// Check for shell-quote single quote bug
	if HasShellQuoteSingleQuoteBug(command) {
		return fmt.Errorf("potential shell injection via single quote bug")
	}

	return nil
}
