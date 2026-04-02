package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Shell Command Parser Types
// =============================================================================

// ParseEntry represents a parsed shell token.
type ParseEntry interface {
	isParseEntry()
}

// StringToken is a simple string token.
type StringToken string

func (StringToken) isParseEntry() {}

// OperatorToken represents a shell operator.
type OperatorToken struct {
	Op string
}

func (OperatorToken) isParseEntry() {}

// GlobToken represents a glob pattern.
type GlobToken struct {
	Pattern string
}

func (GlobToken) isParseEntry() {}

// CommentToken represents a comment.
type CommentToken struct {
	Comment string
}

func (CommentToken) isParseEntry() {}

// =============================================================================
// Heredoc Types
// =============================================================================

// HeredocInfo contains information about a heredoc.
type HeredocInfo struct {
	FullText           string
	Delimiter          string
	OperatorStartIndex int
	OperatorEndIndex   int
	ContentStartIndex  int
	ContentEndIndex    int
}

// HeredocExtractionResult contains the result of heredoc extraction.
type HeredocExtractionResult struct {
	ProcessedCommand string
	Heredocs         map[string]HeredocInfo
}

// =============================================================================
// Placeholder Generation
// =============================================================================

// Placeholders contains unique placeholder strings for safe parsing.
type Placeholders struct {
	SingleQuote       string
	DoubleQuote       string
	NewLine           string
	EscapedOpenParen  string
	EscapedCloseParen string
}

// generatePlaceholders creates unique placeholder strings with random salt.
// Security: This is critical for preventing injection attacks where a command
// could contain literal placeholder strings that would be replaced during parsing.
func generatePlaceholders() Placeholders {
	salt := generateRandomSalt(8)
	return Placeholders{
		SingleQuote:       fmt.Sprintf("__SINGLE_QUOTE_%s__", salt),
		DoubleQuote:       fmt.Sprintf("__DOUBLE_QUOTE_%s__", salt),
		NewLine:           fmt.Sprintf("__NEW_LINE_%s__", salt),
		EscapedOpenParen:  fmt.Sprintf("__ESCAPED_OPEN_PAREN_%s__", salt),
		EscapedCloseParen: fmt.Sprintf("__ESCAPED_CLOSE_PAREN_%s__", salt),
	}
}

// generateRandomSalt generates a random hex string.
func generateRandomSalt(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based salt if random fails
		return fmt.Sprintf("%d", randomCounter())
	}
	return hex.EncodeToString(bytes)
}

// randomCounter provides a simple counter for fallback salt generation.
var counterVal int64

func randomCounter() int64 {
	counterVal++
	return counterVal
}

// =============================================================================
// Heredoc Extraction
// =============================================================================

const (
	heredocPlaceholderPrefix = "__HEREDOC_"
	heredocPlaceholderSuffix = "__"
)

// Heredoc start pattern: <<[-]['"]?DELIM['"]?
// Note: Go regexp doesn't support backreferences, so we use simpler patterns
var heredocStartPattern = regexp.MustCompile(`<<(-)?[ \t]*(?:['"](\w+)['"]|(\w+))`)

// ExtractHeredocs extracts heredocs from a command string and replaces them with placeholders.
// This allows the parser to process the command without mangling heredoc syntax.
func ExtractHeredocs(command string, opts ...HeredocOptions) HeredocExtractionResult {
	options := HeredocOptions{}
	if len(opts) > 0 {
		options = opts[0]
	}

	heredocs := make(map[string]HeredocInfo)

	// Quick check: if no << present, skip processing
	if !strings.Contains(command, "<<") {
		return HeredocExtractionResult{ProcessedCommand: command, Heredocs: heredocs}
	}

	// Security pre-validation: bail out for constructs that could desync quote tracking
	if shouldBailForSecurity(command) {
		return HeredocExtractionResult{ProcessedCommand: command, Heredocs: heredocs}
	}

	// Find all heredoc matches
	matches := findHeredocMatches(command, options)
	if len(matches) == 0 {
		return HeredocExtractionResult{ProcessedCommand: command, Heredocs: heredocs}
	}

	// Filter nested heredocs
	topLevel := filterNestedHeredocs(matches)
	if len(topLevel) == 0 {
		return HeredocExtractionResult{ProcessedCommand: command, Heredocs: heredocs}
	}

	// Check for multiple heredocs sharing the same content start (causes index corruption)
	if hasSharedContentStart(topLevel) {
		return HeredocExtractionResult{ProcessedCommand: command, Heredocs: heredocs}
	}

	// Sort by content end position descending for safe replacement
	sortHeredocsByPosition(topLevel)

	// Generate salt and replace heredocs with placeholders
	salt := generateRandomSalt(8)
	processed := command

	for i, info := range topLevel {
		placeholderIndex := len(topLevel) - 1 - i
		placeholder := fmt.Sprintf("%s%d_%s%s", heredocPlaceholderPrefix, placeholderIndex, salt, heredocPlaceholderSuffix)

		heredocs[placeholder] = info

		// Replace heredoc with placeholder
		processed = processed[:info.OperatorStartIndex] +
			placeholder +
			processed[info.OperatorEndIndex:info.ContentStartIndex] +
			processed[info.ContentEndIndex:]
	}

	return HeredocExtractionResult{ProcessedCommand: processed, Heredocs: heredocs}
}

// HeredocOptions contains options for heredoc extraction.
type HeredocOptions struct {
	QuotedOnly bool // Only extract quoted/escaped delimiter heredocs
}

// shouldBailForSecurity checks for security-sensitive constructs.
func shouldBailForSecurity(command string) bool {
	// Check for $'...' or $"..." (ANSI-C / locale quoting)
	if regexp.MustCompile(`\$['"]`).MatchString(command) {
		return true
	}

	// Check for backticks before first <<
	firstHeredocPos := strings.Index(command, "<<")
	if firstHeredocPos > 0 {
		beforeHeredoc := command[:firstHeredocPos]
		if strings.Contains(beforeHeredoc, "`") {
			return true
		}
	}

	// Check for unbalanced (( before << (arithmetic context)
	if firstHeredocPos > 0 {
		beforeHeredoc := command[:firstHeredocPos]
		openArith := strings.Count(beforeHeredoc, "((")
		closeArith := strings.Count(beforeHeredoc, "))")
		if openArith > closeArith {
			return true
		}
	}

	return false
}

// findHeredocMatches finds all valid heredoc matches in the command.
func findHeredocMatches(command string, options HeredocOptions) []HeredocInfo {
	var matches []HeredocInfo
	skippedRanges := make([]struct{ start, end int }, 0)

	// Scanner state for incremental quote/comment tracking
	scanPos := 0
	scanInSingleQuote := false
	scanInDoubleQuote := false
	scanInComment := false
	scanDqEscapeNext := false
	scanPendingBackslashes := 0

	// Advance scanner to target position
	advanceScan := func(target int) {
		for i := scanPos; i < target; i++ {
			ch := command[i]

			// Newline clears comment state
			if ch == '\n' {
				scanInComment = false
			}

			if scanInSingleQuote {
				if ch == '\'' {
					scanInSingleQuote = false
				}
				continue
			}

			if scanInDoubleQuote {
				if scanDqEscapeNext {
					scanDqEscapeNext = false
					continue
				}
				if ch == '\\' {
					scanDqEscapeNext = true
					continue
				}
				if ch == '"' {
					scanInDoubleQuote = false
				}
				continue
			}

			// Unquoted context
			if ch == '\\' {
				scanPendingBackslashes++
				continue
			}
			escaped := scanPendingBackslashes%2 == 1
			scanPendingBackslashes = 0
			if escaped {
				continue
			}

			if ch == '\'' {
				scanInSingleQuote = true
			} else if ch == '"' {
				scanInDoubleQuote = true
			} else if !scanInComment && ch == '#' {
				scanInComment = true
			}
		}
		scanPos = target
	}

	// Iterate over all heredoc pattern matches
	heredocPattern := regexp.MustCompile(heredocStartPattern.String())
	allMatches := heredocPattern.FindAllStringSubmatchIndex(command, -1)

	for _, matchIdx := range allMatches {
		startIndex := matchIdx[0]
		advanceScan(startIndex)

		// Skip if inside quoted string
		if scanInSingleQuote || scanInDoubleQuote {
			continue
		}

		// Skip if inside comment
		if scanInComment {
			continue
		}

		// Skip if preceded by odd number of backslashes
		if scanPendingBackslashes%2 == 1 {
			continue
		}

		// Skip if inside a previously skipped heredoc's body
		insideSkipped := false
		for _, r := range skippedRanges {
			if startIndex > r.start && startIndex < r.end {
				insideSkipped = true
				break
			}
		}
		if insideSkipped {
			continue
		}

		// Extract match details
		fullMatch := command[matchIdx[0]:matchIdx[1]]
		isDash := matchIdx[2] != -1 // Group 1: dash prefix

		var delimiter string
		var quoteChar string
		if matchIdx[6] != -1 { // Group 3: quoted delimiter
			delimiter = command[matchIdx[6]:matchIdx[7]]
			if matchIdx[4] != -1 { // Group 2: quote char
				quoteChar = command[matchIdx[4]:matchIdx[5]]
			}
		} else if matchIdx[8] != -1 { // Group 4: unquoted delimiter
			delimiter = command[matchIdx[8]:matchIdx[9]]
		}

		operatorEndIndex := matchIdx[1]

		// Verify quote was matched completely
		if quoteChar != "" && len(command) > operatorEndIndex-1 && string(command[operatorEndIndex-1]) != quoteChar {
			continue
		}

		// Check if delimiter is quoted/escaped
		isQuotedOrEscaped := quoteChar != "" || strings.Contains(fullMatch, "\\")

		// Verify next char is a word terminator
		if operatorEndIndex < len(command) {
			nextChar := string(command[operatorEndIndex])
			if !isWordTerminator(nextChar) {
				continue
			}
		}

		// Find content start (first newline not inside quotes)
		contentStartIndex := findContentStart(command, operatorEndIndex)
		if contentStartIndex == -1 {
			continue
		}

		// Check for trailing backslash-newline continuation
		sameLineContent := command[operatorEndIndex:contentStartIndex]
		if hasTrailingBackslash(sameLineContent) {
			continue
		}

		// Find closing delimiter
		afterNewline := command[contentStartIndex+1:]
		contentLines := strings.Split(afterNewline, "\n")
		closingLineIndex := findClosingDelimiter(contentLines, delimiter, isDash)

		// Handle quotedOnly mode
		if options.QuotedOnly && !isQuotedOrEscaped {
			// Track skipped range
			var skipEnd int
			if closingLineIndex == -1 {
				skipEnd = len(command)
			} else {
				skipEnd = contentStartIndex + 1 + len(strings.Join(contentLines[:closingLineIndex+1], "\n"))
			}
			skippedRanges = append(skippedRanges, struct{ start, end int }{contentStartIndex, skipEnd})
			continue
		}

		if closingLineIndex == -1 {
			continue
		}

		// Calculate end position
		linesUpToClosing := contentLines[:closingLineIndex+1]
		contentLength := len(strings.Join(linesUpToClosing, "\n"))
		contentEndIndex := contentStartIndex + 1 + contentLength

		// Check for overlap with skipped ranges
		overlapsSkipped := false
		for _, r := range skippedRanges {
			if contentStartIndex < r.end && r.start < contentEndIndex {
				overlapsSkipped = true
				break
			}
		}
		if overlapsSkipped {
			continue
		}

		// Build full text
		operatorText := command[startIndex:operatorEndIndex]
		contentText := command[contentStartIndex:contentEndIndex]
		fullText := operatorText + contentText

		matches = append(matches, HeredocInfo{
			FullText:           fullText,
			Delimiter:          delimiter,
			OperatorStartIndex: startIndex,
			OperatorEndIndex:   operatorEndIndex,
			ContentStartIndex:  contentStartIndex,
			ContentEndIndex:    contentEndIndex,
		})
	}

	return matches
}

// isWordTerminator checks if a character is a bash word terminator.
func isWordTerminator(ch string) bool {
	return regexp.MustCompile(`^[ \t\n|&;()<>]$`).MatchString(ch)
}

// findContentStart finds the first newline not inside quotes.
func findContentStart(command string, operatorEnd int) int {
	inSingleQuote := false
	inDoubleQuote := false

	for k := operatorEnd; k < len(command); k++ {
		ch := command[k]

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				k++
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		// Count preceding backslashes
		backslashCount := 0
		for j := k - 1; j >= operatorEnd && command[j] == '\\'; j-- {
			backslashCount++
		}

		if ch == '\n' && backslashCount%2 == 0 {
			return k
		}

		if backslashCount%2 == 0 {
			if ch == '\'' {
				inSingleQuote = true
			} else if ch == '"' {
				inDoubleQuote = true
			}
		}
	}

	return -1
}

// hasTrailingBackslash checks if content ends with odd number of backslashes.
func hasTrailingBackslash(content string) bool {
	count := 0
	for i := len(content) - 1; i >= 0 && content[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}

// findClosingDelimiter finds the closing delimiter line.
func findClosingDelimiter(lines []string, delimiter string, isDash bool) int {
	for i, line := range lines {
		var checkLine string
		if isDash {
			// <<- strips leading tabs only
			checkLine = strings.TrimLeft(line, "\t")
		} else {
			checkLine = line
		}

		if checkLine == delimiter {
			return i
		}

		// Check for early closure (metacharacter after delimiter)
		if len(checkLine) > len(delimiter) && strings.HasPrefix(checkLine, delimiter) {
			charAfter := checkLine[len(delimiter)]
			// Check if character after delimiter is a shell metacharacter
			if charAfter == ')' || charAfter == '}' || charAfter == '`' ||
				charAfter == '|' || charAfter == '&' || charAfter == ';' ||
				charAfter == '(' || charAfter == '<' || charAfter == '>' {
				return -1 // Bail out
			}
		}
	}
	return -1
}

// filterNestedHeredocs removes heredocs nested inside other heredocs.
func filterNestedHeredocs(matches []HeredocInfo) []HeredocInfo {
	var result []HeredocInfo

	for _, candidate := range matches {
		isNested := false
		for _, other := range matches {
			if candidate == other {
				continue
			}
			if candidate.OperatorStartIndex > other.ContentStartIndex &&
				candidate.OperatorStartIndex < other.ContentEndIndex {
				isNested = true
				break
			}
		}
		if !isNested {
			result = append(result, candidate)
		}
	}

	return result
}

// hasSharedContentStart checks if multiple heredocs share the same content start.
func hasSharedContentStart(matches []HeredocInfo) bool {
	positions := make(map[int]bool)
	for _, m := range matches {
		if positions[m.ContentStartIndex] {
			return true
		}
		positions[m.ContentStartIndex] = true
	}
	return false
}

// sortHeredocsByPosition sorts heredocs by content end position descending.
func sortHeredocsByPosition(matches []HeredocInfo) {
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].ContentEndIndex > matches[i].ContentEndIndex {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

// RestoreHeredocs restores heredoc placeholders back to their original content.
func RestoreHeredocs(parts []string, heredocs map[string]HeredocInfo) []string {
	if len(heredocs) == 0 {
		return parts
	}

	result := make([]string, len(parts))
	for i, part := range parts {
		for placeholder, info := range heredocs {
			part = strings.ReplaceAll(part, placeholder, info.FullText)
		}
		result[i] = part
	}
	return result
}

// =============================================================================
// Split Command With Operators
// =============================================================================

// Control operators
var commandListSeparators = map[string]bool{
	"&&": true,
	"||": true,
	";":  true,
	";;": true,
	"|":  true,
}

var allSupportedControlOperators = map[string]bool{
	"&&": true,
	"||": true,
	";":  true,
	";;": true,
	"|":  true,
	">&": true,
	">":  true,
	">>": true,
}

// Allowed file descriptors for redirection
var allowedFileDescriptors = map[string]bool{"0": true, "1": true, "2": true}

// SplitCommandWithOperators splits a command string into tokens preserving operators.
// This is the core function for safe shell command parsing.
func SplitCommandWithOperators(command string) []string {
	placeholders := generatePlaceholders()

	// Extract heredocs before parsing
	extraction := ExtractHeredocs(command)
	processedCommand := extraction.ProcessedCommand
	heredocs := extraction.Heredocs

	// Join continuation lines (backslash followed by newline)
	commandWithContinuations := joinContinuationLines(processedCommand)
	commandOriginalJoined := joinContinuationLines(command)

	// Try to parse the command
	tokens, success := tryParseShellCommand(commandWithContinuations, placeholders)
	if !success {
		return []string{commandOriginalJoined}
	}

	if len(tokens) == 0 {
		return []string{}
	}

	// Collapse adjacent strings and convert to string parts
	parts := collapseTokens(tokens, placeholders)

	// Restore heredocs
	return RestoreHeredocs(parts, heredocs)
}

// joinContinuationLines joins backslash-newline continuation lines.
func joinContinuationLines(cmd string) string {
	// Replace \<newline> sequences
	// Odd number of backslashes: continuation (remove last backslash and newline)
	// Even number: literal (keep as-is)
	result := ""
	i := 0
	for i < len(cmd) {
		if cmd[i] == '\\' {
			// Count consecutive backslashes
			backslashCount := 0
			j := i
			for j < len(cmd) && cmd[j] == '\\' {
				backslashCount++
				j++
			}

			// Check if followed by newline
			if j < len(cmd) && cmd[j] == '\n' {
				if backslashCount%2 == 1 {
					// Odd: continuation - remove escaping backslash and newline
					result += strings.Repeat("\\", backslashCount-1)
					i = j + 1
					continue
				}
			}
		}
		result += string(cmd[i])
		i++
	}
	return result
}

// tryParseShellCommand attempts to parse a shell command.
func tryParseShellCommand(cmd string, placeholders Placeholders) ([]ParseEntry, bool) {
	// Apply placeholders to preserve quotes and special chars
	processed := cmd
	processed = strings.ReplaceAll(processed, `"`, `"`+placeholders.DoubleQuote)
	processed = strings.ReplaceAll(processed, `'`, `'`+placeholders.SingleQuote)
	processed = strings.ReplaceAll(processed, "\n", "\n"+placeholders.NewLine+"\n")
	processed = strings.ReplaceAll(processed, `\(`, placeholders.EscapedOpenParen)
	processed = strings.ReplaceAll(processed, `\)`, placeholders.EscapedCloseParen)

	// Simple tokenizer
	tokens, err := tokenize(processed)
	if err != nil {
		return nil, false
	}

	return tokens, true
}

// tokenize performs basic shell tokenization.
func tokenize(cmd string) ([]ParseEntry, error) {
	var tokens []ParseEntry
	var current strings.Builder

	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]

		if escaped {
			escaped = false
			current.WriteByte(ch)
			continue
		}

		if ch == '\\' && !inSingleQuote {
			escaped = true
			current.WriteByte(ch)
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		if !inSingleQuote && !inDoubleQuote {
			// Check for operators
			if isOperatorChar(ch) {
				if current.Len() > 0 {
					tokens = append(tokens, StringToken(current.String()))
					current.Reset()
				}

				// Check for multi-char operators
				op := string(ch)
				if i+1 < len(cmd) {
					twoChar := cmd[i : i+2]
					if isOperator(twoChar) {
						op = twoChar
						i++
					}
				}

				tokens = append(tokens, OperatorToken{Op: op})
				continue
			}

			// Check for whitespace
			if ch == ' ' || ch == '\t' || ch == '\n' {
				if current.Len() > 0 {
					tokens = append(tokens, StringToken(current.String()))
					current.Reset()
				}
				continue
			}
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, StringToken(current.String()))
	}

	return tokens, nil
}

// isOperatorChar checks if a character could start an operator.
func isOperatorChar(ch byte) bool {
	return ch == '&' || ch == '|' || ch == ';' || ch == '>' || ch == '<' || ch == '(' || ch == ')'
}

// isOperator checks if a string is a shell operator.
func isOperator(s string) bool {
	return allSupportedControlOperators[s] || s == "&&" || s == "||" || s == "<<" || s == ">>>" || s == ">&"
}

// collapseTokens collapses adjacent string tokens and converts to string parts.
func collapseTokens(tokens []ParseEntry, placeholders Placeholders) []string {
	var parts []string

	for _, token := range tokens {
		switch t := token.(type) {
		case StringToken:
			str := string(t)
			// Restore placeholders
			str = strings.ReplaceAll(str, placeholders.SingleQuote, "'")
			str = strings.ReplaceAll(str, placeholders.DoubleQuote, `"`)
			str = strings.ReplaceAll(str, "\n"+placeholders.NewLine+"\n", "\n")
			str = strings.ReplaceAll(str, placeholders.EscapedOpenParen, `\(`)
			str = strings.ReplaceAll(str, placeholders.EscapedCloseParen, `\)`)

			// Collapse with previous string
			if len(parts) > 0 && !isOperator(parts[len(parts)-1]) {
				parts[len(parts)-1] += " " + str
			} else {
				parts = append(parts, str)
			}

		case OperatorToken:
			parts = append(parts, t.Op)

		case GlobToken:
			if len(parts) > 0 && !isOperator(parts[len(parts)-1]) {
				parts[len(parts)-1] += " " + t.Pattern
			} else {
				parts = append(parts, t.Pattern)
			}

		case CommentToken:
			// Comments are converted to strings with # prefix
			parts = append(parts, "#"+t.Comment)
		}
	}

	return parts
}

// =============================================================================
// Filter Control Operators
// =============================================================================

// FilterControlOperators removes control operators from the token list.
func FilterControlOperators(parts []string) []string {
	var result []string
	for _, part := range parts {
		if !commandListSeparators[part] {
			result = append(result, part)
		}
	}
	return result
}

// =============================================================================
// Split Command (Deprecated)
// =============================================================================

// SplitCommand splits a command string into individual commands based on operators.
// Deprecated: Use SplitCommandWithOperators for more accurate parsing.
func SplitCommand(command string) []string {
	parts := SplitCommandWithOperators(command)

	// Process redirections
	parts = stripRedirections(parts)

	return FilterControlOperators(parts)
}

// stripRedirections removes safe redirection patterns from parts.
func stripRedirections(parts []string) []string {
	result := make([]string, 0, len(parts))

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if part == ">&" || part == ">" || part == ">>" {
			prev := ""
			if i > 0 {
				prev = strings.TrimSpace(parts[i-1])
			}

			next := ""
			if i+1 < len(parts) {
				next = strings.TrimSpace(parts[i+1])
			}

			// Check for safe redirection patterns
			if shouldStripRedirection(part, prev, next, parts, i) {
				// Remove previous FD if present
				if len(result) > 0 && isFileDescriptor(strings.TrimSpace(result[len(result)-1])) {
					result = result[:len(result)-1]
				}
				i++ // Skip the target
				continue
			}
		}

		result = append(result, part)
	}

	return result
}

// shouldStripRedirection determines if a redirection should be stripped.
func shouldStripRedirection(op, prev, next string, parts []string, i int) bool {
	// 2>&1 style
	if op == ">&" && allowedFileDescriptors[next] {
		return true
	}

	// > &1 style with space
	if op == ">" && next == "&" && i+2 < len(parts) && allowedFileDescriptors[parts[i+2]] {
		return true
	}

	// > &1 style (no space after &)
	if op == ">" && strings.HasPrefix(next, "&") && len(next) > 1 && allowedFileDescriptors[next[1:]] {
		return true
	}

	// General file redirection with static target
	if (op == ">" || op == ">>") && isStaticRedirectTarget(next) {
		return true
	}

	return false
}

// isFileDescriptor checks if a string is a file descriptor number.
func isFileDescriptor(s string) bool {
	return s == "0" || s == "1" || s == "2"
}

// isStaticRedirectTarget checks if a target is a simple static file path.
func isStaticRedirectTarget(target string) bool {
	// Reject targets with whitespace or quotes
	if regexp.MustCompile(`[\s'"]`).MatchString(target) {
		return false
	}

	// Reject empty string
	if target == "" {
		return false
	}

	// Reject comment-prefixed
	if strings.HasPrefix(target, "#") {
		return false
	}

	// Reject dynamic content
	if strings.HasPrefix(target, "!") ||
		strings.HasPrefix(target, "=") ||
		strings.Contains(target, "$") ||
		strings.Contains(target, "`") ||
		strings.Contains(target, "*") ||
		strings.Contains(target, "?") ||
		strings.Contains(target, "[") ||
		strings.Contains(target, "{") ||
		strings.Contains(target, "~") ||
		strings.Contains(target, "(") ||
		strings.Contains(target, "<") ||
		strings.HasPrefix(target, "&") {
		return false
	}

	return true
}

// =============================================================================
// Is Help Command
// =============================================================================

// IsHelpCommand checks if a command is a simple help command.
func IsHelpCommand(command string) bool {
	trimmed := strings.TrimSpace(command)

	// Must end with --help
	if !strings.HasSuffix(trimmed, "--help") {
		return false
	}

	// Reject commands with quotes
	if strings.Contains(trimmed, `"`) || strings.Contains(trimmed, `'`) {
		return false
	}

	// Parse and check tokens
	tokens, success := tryParseShellCommand(trimmed, generatePlaceholders())
	if !success {
		return false
	}

	alphanumericPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	foundHelp := false

	for _, token := range tokens {
		if str, ok := token.(StringToken); ok {
			s := string(str)
			if strings.HasPrefix(s, "-") {
				if s == "--help" {
					foundHelp = true
				} else {
					return false
				}
			} else {
				if !alphanumericPattern.MatchString(s) {
					return false
				}
			}
		}
	}

	return foundHelp
}

// =============================================================================
// Extract Output Redirections
// =============================================================================

// OutputRedirection represents an output redirection.
type OutputRedirection struct {
	Target   string
	Operator string // ">" or ">>"
}

// ExtractOutputRedirections extracts output redirections from a command.
func ExtractOutputRedirections(cmd string) (commandWithoutRedirections string, redirections []OutputRedirection, hasDangerousRedirection bool) {
	// Extract heredocs first
	extraction := ExtractHeredocs(cmd)
	processedCmd := extraction.ProcessedCommand
	heredocs := extraction.Heredocs

	// Join continuation lines
	processedCmd = joinContinuationLines(processedCmd)

	// Tokenize
	tokens, success := tryParseShellCommand(processedCmd, generatePlaceholders())
	if !success {
		return cmd, nil, true // Fail closed
	}

	// Find redirected subshells
	redirectedSubshells := findRedirectedSubshells(tokens)

	// Process tokens and extract redirections
	var kept []ParseEntry
	redirections = []OutputRedirection{}
	hasDangerousRedirection = false
	cmdSubDepth := 0

	for i := 0; i < len(tokens); i++ {
		part := tokens[i]
		var prev, next ParseEntry
		if i > 0 {
			prev = tokens[i-1]
		}
		if i+1 < len(tokens) {
			next = tokens[i+1]
		}

		// Skip redirected subshell parens
		if isOperatorTokenForOp(part, "(") || isOperatorTokenForOp(part, ")") {
			if _, ok := redirectedSubshells[i]; ok {
				continue
			}
		}

		// Track command substitution depth
		if isOperatorTokenForOp(part, "(") && endsWithDollar(prev) {
			cmdSubDepth++
		} else if isOperatorTokenForOp(part, ")") && cmdSubDepth > 0 {
			cmdSubDepth--
		}

		// Extract redirections outside command substitutions
		if cmdSubDepth == 0 {
			skip, dangerous := handleRedirection(part, prev, next, tokens, i, &redirections, &kept)
			if dangerous {
				hasDangerousRedirection = true
			}
			if skip > 0 {
				i += skip
				continue
			}
		}

		kept = append(kept, part)
	}

	// Reconstruct command
	commandWithoutRedirections = reconstructCommand(kept, processedCmd)

	// Restore heredocs
	commandWithoutRedirections = RestoreHeredocs([]string{commandWithoutRedirections}, heredocs)[0]

	return commandWithoutRedirections, redirections, hasDangerousRedirection
}

// findRedirectedSubshells finds subshells that are redirected.
func findRedirectedSubshells(tokens []ParseEntry) map[int]bool {
	result := make(map[int]bool)
	var stack []struct {
		index int
		start bool
	}

	for i, token := range tokens {
		if isOperatorTokenForOp(token, "(") {
			var prev ParseEntry
			if i > 0 {
				prev = tokens[i-1]
			}
			isStart := i == 0 || (isOperatorTokenForOp(prev, "&&") || isOperatorTokenForOp(prev, "||") || isOperatorTokenForOp(prev, ";") || isOperatorTokenForOp(prev, "|"))
			stack = append(stack, struct {
				index int
				start bool
			}{i, isStart})
		} else if isOperatorTokenForOp(token, ")") && len(stack) > 0 {
			opening := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if opening.start && i+1 < len(tokens) {
				next := tokens[i+1]
				if isOperatorTokenForOp(next, ">") || isOperatorTokenForOp(next, ">>") {
					result[opening.index] = true
					result[i] = true
				}
			}
		}
	}

	return result
}

// handleRedirection processes a redirection operator.
func handleRedirection(part, prev, next ParseEntry, tokens []ParseEntry, i int, redirections *[]OutputRedirection, kept *[]ParseEntry) (skip int, dangerous bool) {
	if !isOperatorTokenForOp(part, ">") && !isOperatorTokenForOp(part, ">>") {
		return 0, false
	}

	op := getOperator(part)
	prevStr := getString(prev)
	nextStr := getString(next)

	// File descriptor redirection
	if isFileDescriptor(prevStr) {
		// Handle various FD redirection patterns
		if nextStr == "!" && i+2 < len(tokens) {
			nextNext := getString(tokens[i+2])
			if isSimpleTarget(nextNext) {
				*redirections = append(*redirections, OutputRedirection{nextNext, op})
				return 2, false
			}
			if hasDangerousExpansion(nextNext) {
				return 0, true
			}
		}

		if isSimpleTarget(nextStr) {
			*redirections = append(*redirections, OutputRedirection{nextStr, op})
			return 1, false
		}

		if hasDangerousExpansion(nextStr) {
			return 0, true
		}
	}

	// Standard stdout redirection
	if isSimpleTarget(nextStr) {
		*redirections = append(*redirections, OutputRedirection{nextStr, op})
		return 1, false
	}

	if hasDangerousExpansion(nextStr) {
		return 0, true
	}

	return 0, false
}

// isOperatorTokenForOp checks if a token is a specific operator string.
func isOperatorTokenForOp(token ParseEntry, op string) bool {
	if ot, ok := token.(OperatorToken); ok {
		return ot.Op == op
	}
	return false
}

// getOperator returns the operator string from a token.
func getOperator(token ParseEntry) string {
	if ot, ok := token.(OperatorToken); ok {
		return ot.Op
	}
	return ""
}

// getString returns the string value from a token.
func getString(token ParseEntry) string {
	if st, ok := token.(StringToken); ok {
		return string(st)
	}
	return ""
}

// isOperatorToken checks if a ParseEntry token is an operator.
func isOperatorToken(token ParseEntry) bool {
	_, ok := token.(OperatorToken)
	return ok
}

// endsWithDollar checks if a string token ends with $.
func endsWithDollar(token ParseEntry) bool {
	if st, ok := token.(StringToken); ok {
		return strings.HasSuffix(string(st), "$")
	}
	return false
}

// isSimpleTarget checks if a target is safe for path validation.
func isSimpleTarget(target string) bool {
	if target == "" {
		return false
	}
	return !strings.Contains(target, "$") &&
		!strings.Contains(target, "%") &&
		!strings.Contains(target, "`") &&
		!strings.Contains(target, "*") &&
		!strings.Contains(target, "?") &&
		!strings.Contains(target, "[") &&
		!strings.Contains(target, "{") &&
		!strings.HasPrefix(target, "!") &&
		!strings.HasPrefix(target, "=") &&
		!strings.HasPrefix(target, "~")
}

// hasDangerousExpansion checks if a target has dangerous shell expansion.
func hasDangerousExpansion(target string) bool {
	if target == "" {
		return false
	}
	return strings.Contains(target, "$") ||
		strings.Contains(target, "%") ||
		strings.Contains(target, "`") ||
		strings.Contains(target, "*") ||
		strings.Contains(target, "?") ||
		strings.Contains(target, "[") ||
		strings.Contains(target, "{") ||
		strings.HasPrefix(target, "!") ||
		strings.HasPrefix(target, "=") ||
		strings.HasPrefix(target, "~")
}

// reconstructCommand rebuilds a command from tokens.
func reconstructCommand(tokens []ParseEntry, originalCmd string) string {
	if len(tokens) == 0 {
		return originalCmd
	}

	var result strings.Builder
	for i, token := range tokens {
		if i > 0 {
			result.WriteString(" ")
		}
		switch t := token.(type) {
		case StringToken:
			result.WriteString(string(t))
		case OperatorToken:
			result.WriteString(t.Op)
		case GlobToken:
			result.WriteString(t.Pattern)
		}
	}

	return result.String()
}
