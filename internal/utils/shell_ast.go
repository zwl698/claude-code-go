package utils

import (
	"regexp"
	"strings"
)

// =============================================================================
// AST-based Shell Security Parser
// =============================================================================
//
// This module provides shell command security analysis without Tree-sitter.
// It implements the core security checks from ast.ts in a simplified form.
//
// Key Design Properties:
// 1. FAIL-CLOSED: Any unknown or dangerous construct returns 'too-complex'
// 2. EXPLICIT ALLOWLIST: Only known-safe constructs are allowed
// 3. DEFENSE IN DEPTH: Multiple layers of validation

// Redirect represents a shell redirection.
type Redirect struct {
	Op     string // ">", ">>", "<", "<<", ">&", ">|", "<&", "&>", "&>>", "<<<"
	Target string
	FD     *int // Optional file descriptor
}

// SimpleCommand represents a parsed shell command.
type SimpleCommand struct {
	Argv      []string           // argv[0] is the command name
	EnvVars   []EnvVarAssignment // Leading VAR=val assignments
	Redirects []Redirect         // Output/input redirects
	Text      string             // Original source span
}

// EnvVarAssignment represents a variable assignment.
type EnvVarAssignment struct {
	Name  string
	Value string
}

// ParseForSecurityResult represents the result of security parsing.
type ParseForSecurityResult struct {
	Kind     string          // "simple", "too-complex", "parse-unavailable"
	Commands []SimpleCommand // Valid when Kind == "simple"
	Reason   string          // Valid when Kind == "too-complex"
	NodeType string          // Node type that caused rejection
}

// SemanticCheckResult represents the result of semantic validation.
type SemanticCheckResult struct {
	Ok     bool
	Reason string
}

// =============================================================================
// Constants and Patterns
// =============================================================================

// Placeholder strings for command substitution and tracked variables
const (
	CmdSubPlaceholder = "__CMDSUB_OUTPUT__"
	VarPlaceholder    = "__TRACKED_VAR__"
)

// Structural types that represent composition of commands
var structuralTypes = map[string]bool{
	"program":              true,
	"list":                 true,
	"pipeline":             true,
	"redirected_statement": true,
}

// Separator types between commands
var separatorTypes = map[string]bool{
	"&&": true,
	"||": true,
	"|":  true,
	";":  true,
	"&":  true,
	"|&": true,
	"\n": true,
}

// Dangerous node types that cannot be statically analyzed
var dangerousTypes = map[string]bool{
	"command_substitution": true,
	"process_substitution": true,
	"expansion":            true,
	"simple_expansion":     true,
	"brace_expression":     true,
	"subshell":             true,
	"compound_statement":   true,
	"for_statement":        true,
	"while_statement":      true,
	"until_statement":      true,
	"if_statement":         true,
	"case_statement":       true,
	"function_definition":  true,
	"test_command":         true,
	"ansi_c_string":        true,
	"translated_string":    true,
	"herestring_redirect":  true,
	"heredoc_redirect":     true,
}

// Zsh module builtins - dangerous in BashTool context
var zshDangerousBuiltins = map[string]bool{
	"zmodload": true,
	"emulate":  true,
	"sysopen":  true,
	"sysread":  true,
	"syswrite": true,
	"sysseek":  true,
	"zpty":     true,
	"ztcp":     true,
	"zsocket":  true,
	"zf_rm":    true,
	"zf_mv":    true,
	"zf_ln":    true,
	"zf_chmod": true,
	"zf_chown": true,
	"zf_mkdir": true,
	"zf_rmdir": true,
	"zf_chgrp": true,
}

// Shell builtins that evaluate arguments as code
var evalLikeBuiltins = map[string]bool{
	"eval":      true,
	"source":    true,
	".":         true,
	"exec":      true,
	"command":   true,
	"builtin":   true,
	"fc":        true,
	"coproc":    true,
	"noglob":    true,
	"nocorrect": true,
	"trap":      true,
	"enable":    true,
	"mapfile":   true,
	"readarray": true,
	"hash":      true,
	"bind":      true,
	"complete":  true,
	"compgen":   true,
	"alias":     true,
	"let":       true,
}

// Flags that take a NAME argument and evaluate subscripts arithmetically
var subscriptEvalFlags = map[string]map[string]bool{
	"test":   {"-v": true, "-R": true},
	"[":      {"-v": true, "-R": true},
	"[[":     {"-v": true, "-R": true},
	"printf": {"-v": true},
	"read":   {"-a": true},
	"unset":  {"-v": true},
	"wait":   {"-p": true},
}

// Arithmetic comparison operators in [[ ]]
var testArithCmpOps = map[string]bool{
	"-eq": true,
	"-ne": true,
	"-lt": true,
	"-le": true,
	"-gt": true,
	"-ge": true,
}

// Builtins where every positional arg is a NAME with subscript evaluation
var bareSubscriptNameBuiltins = map[string]bool{
	"read":  true,
	"unset": true,
}

// Read flags whose next argument is data, not a NAME
var readDataFlags = map[string]bool{
	"-p": true,
	"-d": true,
	"-n": true,
	"-N": true,
	"-t": true,
	"-u": true,
	"-i": true,
}

// Safe environment variables (shell-controlled values)
var safeEnvVars = map[string]bool{
	"HOME":         true,
	"PWD":          true,
	"OLDPWD":       true,
	"USER":         true,
	"LOGNAME":      true,
	"SHELL":        true,
	"PATH":         true,
	"HOSTNAME":     true,
	"UID":          true,
	"EUID":         true,
	"PPID":         true,
	"RANDOM":       true,
	"SECONDS":      true,
	"LINENO":       true,
	"TMPDIR":       true,
	"BASH_VERSION": true,
	"BASHPID":      true,
	"SHLVL":        true,
	"HISTFILE":     true,
	"IFS":          true,
}

// Special variable names ($?, $$, $!, $#, $0, $-)
var specialVarNames = map[string]bool{
	"?": true,
	"$": true,
	"!": true,
	"#": true,
	"0": true,
	"-": true,
}

// Shell keywords that should never appear as argv[0]
var shellKeywords = map[string]bool{
	"if":       true,
	"then":     true,
	"else":     true,
	"elif":     true,
	"fi":       true,
	"case":     true,
	"esac":     true,
	"for":      true,
	"select":   true,
	"while":    true,
	"until":    true,
	"do":       true,
	"done":     true,
	"in":       true,
	"function": true,
	"time":     true,
	"coproc":   true,
	"!":        true,
	"{":        true,
	"}":        true,
	"[[":       true,
	"]]":       true,
}

// Regular expressions for pre-checks
var (
	// Control characters that bash drops
	controlCharRE = regexp.MustCompile(`[\x00-\x08\x0B-\x1F\x7F]`)

	// Unicode whitespace beyond ASCII
	unicodeWhitespaceRE = regexp.MustCompile(`[\u00A0\u1680\u2000-\u200B\u2028\u2029\u202F\u205F\u3000\uFEFF]`)

	// Backslash before whitespace
	backslashWhitespaceRE = regexp.MustCompile(`\\[ \t]|[^ \t\n\\]\\\n`)

	// Zsh tilde bracket (~[)
	zshTildeBracketRE = regexp.MustCompile(`~\[`)

	// Zsh equals expansion (=cmd)
	zshEqualsExpansionRE = regexp.MustCompile(`(?:^|[\s;&|])=[a-zA-Z_]`)

	// Brace expansion
	braceExpansionRE = regexp.MustCompile(`\{[^{}\s]*(,|\.\.)[^{}\s]*\}`)

	// Brace with quote
	braceWithQuoteRE = regexp.MustCompile(`\{[^}]*['"]`)

	// Newline followed by #
	newlineHashRE = regexp.MustCompile(`\n[ \t]*#`)

	// /proc/*/environ path
	procEnvironRE = regexp.MustCompile(`/proc/.*/environ`)

	// BARE variable unsafe characters
	bareVarUnsafeRE = regexp.MustCompile(`[ \t\n*?\[]`)

	// Arithmetic leaf pattern
	arithLeafRE = regexp.MustCompile(`^(?:[0-9]+|0[xX][0-9a-fA-F]+|[0-9]+#[0-9a-zA-Z]+|[-+*/%^&|~!<>=?:(),]+|<<|>>|\*\*|&&|\|\||[<>=!]=|\$\(\(|\)\))$`)
)

// =============================================================================
// Core Parsing Functions
// =============================================================================

// ParseForSecurity parses a bash command string and extracts simple commands.
// Returns 'too-complex' if the command uses features we can't statically analyze.
func ParseForSecurity(cmd string) ParseForSecurityResult {
	// Empty command
	if cmd == "" {
		return ParseForSecurityResult{Kind: "simple", Commands: []SimpleCommand{}}
	}

	// Pre-checks: characters that cause parsing differentials
	if controlCharRE.MatchString(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains control characters",
			NodeType: "pre-check",
		}
	}

	if unicodeWhitespaceRE.MatchString(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains Unicode whitespace",
			NodeType: "pre-check",
		}
	}

	if backslashWhitespaceRE.MatchString(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains backslash-escaped whitespace",
			NodeType: "pre-check",
		}
	}

	if zshTildeBracketRE.MatchString(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains zsh ~[ dynamic directory syntax",
			NodeType: "pre-check",
		}
	}

	if zshEqualsExpansionRE.MatchString(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains zsh =cmd equals expansion",
			NodeType: "pre-check",
		}
	}

	if braceWithQuoteRE.MatchString(maskBracesInQuotedContexts(cmd)) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains brace with quote character (expansion obfuscation)",
			NodeType: "pre-check",
		}
	}

	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return ParseForSecurityResult{Kind: "simple", Commands: []SimpleCommand{}}
	}

	// Check for dangerous constructs
	if hasDangerousConstructs(cmd) {
		return ParseForSecurityResult{
			Kind:     "too-complex",
			Reason:   "Contains dangerous shell constructs",
			NodeType: "dangerous-construct",
		}
	}

	// Parse the command
	commands, result := parseCommands(cmd)
	if result.Kind == "too-complex" {
		return result
	}

	return ParseForSecurityResult{Kind: "simple", Commands: commands}
}

// hasDangerousConstructs checks for shell constructs that cannot be safely analyzed.
func hasDangerousConstructs(cmd string) bool {
	// Check for command substitution
	if strings.Contains(cmd, "$(") || strings.Contains(cmd, "`") {
		return true
	}

	// Check for process substitution
	if strings.Contains(cmd, "<(") || strings.Contains(cmd, ">(") {
		return true
	}

	// Check for variable expansion ${...}
	if regexp.MustCompile(`\$\{`).MatchString(cmd) {
		return true
	}

	// Check for compound commands
	if regexp.MustCompile(`^\s*\(`).MatchString(cmd) {
		return true
	}

	// Check for for/while/until/if/case keywords
	keywords := []string{"for ", "while ", "until ", "if ", "case ", "function "}
	for _, kw := range keywords {
		if regexp.MustCompile(`(?i)^\s*` + kw).MatchString(cmd) {
			return true
		}
	}

	// Check for [[ ]] or [ ] test commands
	if regexp.MustCompile(`\[\[|\]\]`).MatchString(cmd) {
		return true
	}

	// Check for here-strings <<<
	if strings.Contains(cmd, "<<<") {
		return true
	}

	return false
}

// parseCommands parses a command string into simple commands.
func parseCommands(cmd string) ([]SimpleCommand, ParseForSecurityResult) {
	// Use the existing parser infrastructure
	parts := SplitCommandWithOperators(cmd)
	if len(parts) == 0 {
		return []SimpleCommand{}, ParseForSecurityResult{Kind: "simple", Commands: []SimpleCommand{}}
	}

	var commands []SimpleCommand
	var currentCmd *SimpleCommand
	var envVars []EnvVarAssignment

	for _, part := range parts {
		// Check for operators
		if separatorTypes[part] {
			if currentCmd != nil {
				commands = append(commands, *currentCmd)
				currentCmd = nil
			}
			// Reset env vars on ||, |, |&, &
			if part == "||" || part == "|" || part == "|&" || part == "&" {
				envVars = nil
			}
			continue
		}

		// Check for variable assignment
		if isVarAssignment(part) {
			name, value := parseVarAssignment(part)
			envVars = append(envVars, EnvVarAssignment{Name: name, Value: value})
			continue
		}

		// Check for redirection
		if isRedirection(part) {
			// Skip redirections for now (handled elsewhere)
			continue
		}

		// Parse as regular command
		argv := parseArgv(part)
		if len(argv) > 0 {
			if currentCmd != nil {
				commands = append(commands, *currentCmd)
			}
			currentCmd = &SimpleCommand{
				Argv:      argv,
				EnvVars:   envVars,
				Redirects: []Redirect{},
				Text:      part,
			}
			// Env vars are consumed by the command
			envVars = nil
		}
	}

	if currentCmd != nil {
		commands = append(commands, *currentCmd)
	}

	return commands, ParseForSecurityResult{Kind: "simple", Commands: commands}
}

// isVarAssignment checks if a string is a variable assignment.
func isVarAssignment(s string) bool {
	// VAR=value or VAR+=value pattern
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return false
	}

	name := s[:idx]
	// Check if name is a valid variable name
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
		return false
	}

	// Check it's not a comparison operator like ==
	if idx < len(s)-1 && s[idx+1] == '=' {
		return false
	}

	return true
}

// parseVarAssignment parses a variable assignment into name and value.
func parseVarAssignment(s string) (name, value string) {
	// Handle += operator
	if strings.Contains(s, "+=") {
		idx := strings.Index(s, "+=")
		return s[:idx], s[idx+2:]
	}

	idx := strings.Index(s, "=")
	return s[:idx], s[idx+1:]
}

// isRedirection checks if a string is a redirection operator.
func isRedirection(s string) bool {
	return s == ">" || s == ">>" || s == "<" || s == "<<" ||
		s == ">&" || s == ">|" || s == "<&" || s == "&>" ||
		s == "&>>" || s == "<<<"
}

// parseArgv parses a command string into argv array.
func parseArgv(cmd string) []string {
	// Use the existing tokenizer
	tokens, success := tryParseShellCommand(cmd, generatePlaceholders())
	if !success {
		return nil
	}

	var argv []string
	for _, token := range tokens {
		if st, ok := token.(StringToken); ok {
			argv = append(argv, string(st))
		}
	}

	return argv
}

// =============================================================================
// Semantic Checks
// =============================================================================

// CheckSemantics performs post-argv semantic validation.
func CheckSemantics(commands []SimpleCommand) SemanticCheckResult {
	for _, cmd := range commands {
		// Strip safe wrapper commands
		argv := stripSafeWrappers(cmd.Argv)

		if len(argv) == 0 {
			continue
		}

		name := argv[0]

		// Empty command name check
		if name == "" {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Empty command name — argv[0] may not reflect what bash runs",
			}
		}

		// Placeholder check
		if strings.Contains(name, CmdSubPlaceholder) || strings.Contains(name, VarPlaceholder) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Command name is runtime-determined (placeholder argv[0])",
			}
		}

		// Fragment check
		if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "|") || strings.HasPrefix(name, "&") {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Command appears to be an incomplete fragment",
			}
		}

		// Subscript evaluation check
		if result := checkSubscriptEvaluation(name, argv); !result.Ok {
			return result
		}

		// Shell keyword check
		if shellKeywords[name] {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Shell keyword '" + name + "' as command name — tree-sitter mis-parse",
			}
		}

		// Newline-hash check
		if result := checkNewlineHash(cmd); !result.Ok {
			return result
		}

		// jq system() check
		if name == "jq" {
			if result := checkJqCommand(argv); !result.Ok {
				return result
			}
		}

		// Zsh dangerous builtins
		if zshDangerousBuiltins[name] {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Zsh builtin '" + name + "' can bypass security checks",
			}
		}

		// Eval-like builtins
		if evalLikeBuiltins[name] {
			if result := checkEvalLikeBuiltin(name, argv); !result.Ok {
				return result
			}
		}

		// /proc/*/environ check
		if result := checkProcEnviron(cmd); !result.Ok {
			return result
		}
	}

	return SemanticCheckResult{Ok: true}
}

// stripSafeWrappers removes safe wrapper commands like nohup, time, timeout.
func stripSafeWrappers(argv []string) []string {
	for len(argv) > 0 {
		name := argv[0]

		if name == "time" || name == "nohup" {
			argv = argv[1:]
			continue
		}

		if name == "timeout" {
			argv = stripTimeoutFlags(argv)
			continue
		}

		if name == "nice" {
			argv = stripNiceFlags(argv)
			continue
		}

		if name == "env" {
			argv = stripEnvFlags(argv)
			continue
		}

		if name == "stdbuf" {
			argv = stripStdbufFlags(argv)
			continue
		}

		break
	}

	return argv
}

// stripTimeoutFlags strips timeout flags and duration.
func stripTimeoutFlags(argv []string) []string {
	i := 1
	for i < len(argv) {
		arg := argv[i]

		// Long flags with no value
		if arg == "--foreground" || arg == "--preserve-status" || arg == "--verbose" {
			i++
			continue
		}

		// Long flags with fused value
		if regexp.MustCompile(`^--(?:kill-after|signal)=`).MatchString(arg) {
			i++
			continue
		}

		// Long flags with space-separated value
		if (arg == "--kill-after" || arg == "--signal") && i+1 < len(argv) {
			i += 2
			continue
		}

		// Unknown long flag
		if strings.HasPrefix(arg, "--") {
			return []string{"timeout"} // Fail closed
		}

		// Short flags
		if arg == "-v" {
			i++
			continue
		}

		if (arg == "-k" || arg == "-s") && i+1 < len(argv) {
			i += 2
			continue
		}

		if regexp.MustCompile(`^-[ks][A-Za-z0-9_.+-]+$`).MatchString(arg) {
			i++
			continue
		}

		// Unknown short flag
		if strings.HasPrefix(arg, "-") {
			return []string{"timeout"} // Fail closed
		}

		// Duration
		if regexp.MustCompile(`^\d+(?:\.\d+)?[smhd]?$`).MatchString(arg) {
			return argv[i+1:]
		}

		// Unknown duration format
		return []string{"timeout"} // Fail closed
	}

	return []string{"timeout"}
}

// stripNiceFlags strips nice flags.
func stripNiceFlags(argv []string) []string {
	if len(argv) < 2 {
		return argv
	}

	if argv[1] == "-n" && len(argv) > 2 && regexp.MustCompile(`^-?\d+$`).MatchString(argv[2]) {
		return argv[3:]
	}

	if regexp.MustCompile(`^-\d+$`).MatchString(argv[1]) {
		return argv[2:]
	}

	if regexp.MustCompile(`^-\d+$`).MatchString(argv[1]) || strings.Contains(argv[1], "$") {
		return []string{"nice"} // Fail closed
	}

	return argv[1:]
}

// stripEnvFlags strips env flags and VAR=val assignments.
func stripEnvFlags(argv []string) []string {
	i := 1
	for i < len(argv) {
		arg := argv[i]

		// VAR=val assignment
		if strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-") {
			i++
			continue
		}

		// Flags with no argument
		if arg == "-i" || arg == "-0" || arg == "-v" {
			i++
			continue
		}

		// -u NAME
		if arg == "-u" && i+1 < len(argv) {
			i += 2
			continue
		}

		// Unknown flag
		if strings.HasPrefix(arg, "-") {
			return []string{"env"} // Fail closed
		}

		break
	}

	return argv[i:]
}

// stripStdbufFlags strips stdbuf flags.
func stripStdbufFlags(argv []string) []string {
	shortSepRE := regexp.MustCompile(`^-[ioe]$`)
	shortFusedRE := regexp.MustCompile(`^-[ioe].`)
	longRE := regexp.MustCompile(`^--(input|output|error)=`)

	i := 1
	for i < len(argv) {
		arg := argv[i]

		if shortSepRE.MatchString(arg) && i+1 < len(argv) {
			i += 2
			continue
		}

		if shortFusedRE.MatchString(arg) {
			i++
			continue
		}

		if longRE.MatchString(arg) {
			i++
			continue
		}

		if strings.HasPrefix(arg, "-") {
			return []string{"stdbuf"} // Fail closed
		}

		break
	}

	if i > 1 && i < len(argv) {
		return argv[i:]
	}

	return []string{"stdbuf"}
}

// checkSubscriptEvaluation checks for subscript evaluation vulnerabilities.
func checkSubscriptEvaluation(name string, argv []string) SemanticCheckResult {
	// Check for flags that take NAME argument
	dangerFlags, hasDangerFlags := subscriptEvalFlags[name]
	if hasDangerFlags {
		for i := 1; i < len(argv); i++ {
			arg := argv[i]

			// Separate form: -v then NAME in next arg
			if dangerFlags[arg] && i+1 < len(argv) && strings.Contains(argv[i+1], "[") {
				return SemanticCheckResult{
					Ok:     false,
					Reason: "'" + name + " " + arg + "' operand contains array subscript — bash evaluates $(cmd) in subscripts",
				}
			}

			// Fused form: -vNAME in one arg
			for flag := range dangerFlags {
				if len(flag) == 2 && strings.HasPrefix(arg, flag) && len(arg) > 2 && strings.Contains(arg, "[") {
					return SemanticCheckResult{
						Ok:     false,
						Reason: "'" + name + " " + flag + "' (fused) operand contains array subscript — bash evaluates $(cmd) in subscripts",
					}
				}
			}
		}
	}

	// Check for [[ ]] arithmetic comparison
	if name == "[[" {
		for i := 2; i < len(argv); i++ {
			if testArithCmpOps[argv[i]] {
				if i > 0 && strings.Contains(argv[i-1], "[") {
					return SemanticCheckResult{
						Ok:     false,
						Reason: "'[[ ... " + argv[i] + " ... ]]' operand contains array subscript — bash arithmetically evaluates $(cmd) in subscripts",
					}
				}
				if i+1 < len(argv) && strings.Contains(argv[i+1], "[") {
					return SemanticCheckResult{
						Ok:     false,
						Reason: "'[[ ... " + argv[i] + " ... ]]' operand contains array subscript — bash arithmetically evaluates $(cmd) in subscripts",
					}
				}
			}
		}
	}

	// Check for read/unset positional NAME args
	if bareSubscriptNameBuiltins[name] {
		skipNext := false
		for i := 1; i < len(argv); i++ {
			arg := argv[i]

			if skipNext {
				skipNext = false
				continue
			}

			if strings.HasPrefix(arg, "-") {
				if name == "read" && readDataFlags[arg] {
					skipNext = true
				}
				continue
			}

			if strings.Contains(arg, "[") {
				return SemanticCheckResult{
					Ok:     false,
					Reason: "'" + name + "' positional NAME '" + arg + "' contains array subscript — bash evaluates $(cmd) in subscripts",
				}
			}
		}
	}

	return SemanticCheckResult{Ok: true}
}

// checkNewlineHash checks for newline followed by # in arguments.
func checkNewlineHash(cmd SimpleCommand) SemanticCheckResult {
	for _, arg := range cmd.Argv {
		if strings.Contains(arg, "\n") && newlineHashRE.MatchString(arg) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Newline followed by # inside a quoted argument can hide arguments from path validation",
			}
		}
	}

	for _, ev := range cmd.EnvVars {
		if strings.Contains(ev.Value, "\n") && newlineHashRE.MatchString(ev.Value) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Newline followed by # inside an env var value can hide arguments from path validation",
			}
		}
	}

	for _, r := range cmd.Redirects {
		if strings.Contains(r.Target, "\n") && newlineHashRE.MatchString(r.Target) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Newline followed by # inside a redirect target can hide arguments from path validation",
			}
		}
	}

	return SemanticCheckResult{Ok: true}
}

// checkJqCommand checks for dangerous jq constructs.
func checkJqCommand(argv []string) SemanticCheckResult {
	for _, arg := range argv {
		if regexp.MustCompile(`\bsystem\s*\(`).MatchString(arg) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "jq command contains system() function which executes arbitrary commands",
			}
		}
	}

	// Check for dangerous flags
	dangerousFlagRE := regexp.MustCompile(`^(?:-[fL](?:$|[^A-Za-z])|--(?:from-file|rawfile|slurpfile|library-path)(?:$|=))`)
	for _, arg := range argv {
		if dangerousFlagRE.MatchString(arg) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "jq command contains dangerous flags that could execute code or read arbitrary files",
			}
		}
	}

	return SemanticCheckResult{Ok: true}
}

// checkEvalLikeBuiltin checks for safe forms of eval-like builtins.
func checkEvalLikeBuiltin(name string, argv []string) SemanticCheckResult {
	// command -v/-V are safe
	if name == "command" && len(argv) > 1 && (argv[1] == "-v" || argv[1] == "-V") {
		return SemanticCheckResult{Ok: true}
	}

	// fc without -e/-s is safe (lists history)
	if name == "fc" {
		for _, arg := range argv[1:] {
			if regexp.MustCompile(`^-[^-]*[es]`).MatchString(arg) {
				return SemanticCheckResult{
					Ok:     false,
					Reason: "'" + name + "' evaluates arguments as shell code",
				}
			}
		}
		return SemanticCheckResult{Ok: true}
	}

	// compgen without -C/-F/-W is safe
	if name == "compgen" {
		for _, arg := range argv[1:] {
			if regexp.MustCompile(`^-[^-]*[CFW]`).MatchString(arg) {
				return SemanticCheckResult{
					Ok:     false,
					Reason: "'" + name + "' evaluates arguments as shell code",
				}
			}
		}
		return SemanticCheckResult{Ok: true}
	}

	return SemanticCheckResult{
		Ok:     false,
		Reason: "'" + name + "' evaluates arguments as shell code",
	}
}

// checkProcEnviron checks for /proc/*/environ access.
func checkProcEnviron(cmd SimpleCommand) SemanticCheckResult {
	for _, arg := range cmd.Argv {
		if strings.Contains(arg, "/proc/") && procEnvironRE.MatchString(arg) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Accesses /proc/*/environ which may expose secrets",
			}
		}
	}

	for _, r := range cmd.Redirects {
		if strings.Contains(r.Target, "/proc/") && procEnvironRE.MatchString(r.Target) {
			return SemanticCheckResult{
				Ok:     false,
				Reason: "Accesses /proc/*/environ which may expose secrets",
			}
		}
	}

	return SemanticCheckResult{Ok: true}
}

// =============================================================================
// Helper Functions
// =============================================================================

// maskBracesInQuotedContexts masks { characters inside quoted contexts.
func maskBracesInQuotedContexts(cmd string) string {
	if !strings.Contains(cmd, "{") {
		return cmd
	}

	var result strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			if c == '{' {
				result.WriteByte(' ')
			} else {
				result.WriteByte(c)
			}
			continue
		}

		if inDouble {
			if c == '\\' && i+1 < len(cmd) && (cmd[i+1] == '"' || cmd[i+1] == '\\') {
				result.WriteByte(c)
				result.WriteByte(cmd[i+1])
				i++
				continue
			}
			if c == '"' {
				inDouble = false
			}
			if c == '{' {
				result.WriteByte(' ')
			} else {
				result.WriteByte(c)
			}
			continue
		}

		// Unquoted context
		if c == '\\' && i+1 < len(cmd) {
			result.WriteByte(c)
			result.WriteByte(cmd[i+1])
			i++
			continue
		}

		if c == '\'' {
			inSingle = true
		} else if c == '"' {
			inDouble = true
		}

		result.WriteByte(c)
	}

	return result.String()
}

// ContainsAnyPlaceholder checks if a value contains any placeholder.
func ContainsAnyPlaceholder(value string) bool {
	return strings.Contains(value, CmdSubPlaceholder) || strings.Contains(value, VarPlaceholder)
}

// NodeTypeId returns a numeric ID for a node type (for analytics).
func NodeTypeId(nodeType string) int {
	if nodeType == "" {
		return -2
	}
	if nodeType == "ERROR" {
		return -1
	}

	types := []string{
		"command_substitution",
		"process_substitution",
		"expansion",
		"simple_expansion",
		"brace_expression",
		"subshell",
		"compound_statement",
		"for_statement",
		"while_statement",
		"until_statement",
		"if_statement",
		"case_statement",
		"function_definition",
		"test_command",
		"ansi_c_string",
		"translated_string",
		"herestring_redirect",
		"heredoc_redirect",
	}

	for i, t := range types {
		if t == nodeType {
			return i + 1
		}
	}

	return 0
}

// IsSafeEnvVar checks if a variable name is in the safe env vars list.
func IsSafeEnvVar(name string) bool {
	return safeEnvVars[name]
}

// IsSpecialVarName checks if a name is a special variable name.
func IsSpecialVarName(name string) bool {
	return specialVarNames[name] || regexp.MustCompile(`^[0-9]+$`).MatchString(name)
}

// IsShellKeyword checks if a name is a shell keyword.
func IsShellKeyword(name string) bool {
	return shellKeywords[name]
}

// IsEvalLikeBuiltin checks if a name is an eval-like builtin.
func IsEvalLikeBuiltin(name string) bool {
	return evalLikeBuiltins[name]
}

// IsZshDangerousBuiltin checks if a name is a zsh dangerous builtin.
func IsZshDangerousBuiltin(name string) bool {
	return zshDangerousBuiltins[name]
}
