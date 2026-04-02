package commands

import (
	"context"
	"fmt"
)

// =============================================================================
// Prompt-based Commands
// These commands generate prompts for the AI to execute specific tasks.
// =============================================================================

// CommitCommand creates a git commit.
type CommitCommand struct{}

func NewCommitCommand() *CommitCommand { return &CommitCommand{} }

func (c *CommitCommand) Name() string        { return "commit" }
func (c *CommitCommand) Description() string { return "Create a git commit" }
func (c *CommitCommand) IsEnabled() bool     { return true }
func (c *CommitCommand) IsHidden() bool      { return false }

func (c *CommitCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := `## Context

- Current git status: !` + "`git status`" + `
- Current git diff (staged and unstaged changes): !` + "`git diff HEAD`" + `
- Current branch: !` + "`git branch --show-current`" + `
- Recent commits: !` + "`git log --oneline -10`" + `

## Git Safety Protocol

- NEVER update the git config
- NEVER skip hooks (--no-verify, --no-gpg-sign, etc) unless the user explicitly requests it
- CRITICAL: ALWAYS create NEW commits. NEVER use git commit --amend, unless the user explicitly requests it
- Do not commit files that likely contain secrets (.env, credentials.json, etc). Warn the user if they specifically request to commit those files
- If there are no changes to commit (i.e., no untracked files and no modifications), do not create an empty commit
- Never use git commands with the -i flag (like git rebase -i or git add -i) since they require interactive input which is not supported

## Your task

Based on the above changes, create a single git commit:

1. Analyze all staged changes and draft a commit message:
   - Look at the recent commits above to follow this repository's commit message style
   - Summarize the nature of the changes (new feature, enhancement, bug fix, refactoring, test, docs, etc.)
   - Ensure the message accurately reflects the changes and their purpose
   - Draft a concise (1-2 sentences) commit message that focuses on the "why" rather than the "what"

2. Stage relevant files and create the commit using HEREDOC syntax:
` + "```" + `
git commit -m "$(cat <<'EOF'
Commit message here.
EOF
)"
` + "```" + `

You have the capability to call multiple tools in a single response. Stage and create the commit using a single message.`

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// ReviewCommand reviews a pull request.
type ReviewCommand struct{}

func NewReviewCommand() *ReviewCommand { return &ReviewCommand{} }

func (c *ReviewCommand) Name() string        { return "review" }
func (c *ReviewCommand) Description() string { return "Review a pull request" }
func (c *ReviewCommand) IsEnabled() bool     { return true }
func (c *ReviewCommand) IsHidden() bool      { return false }

func (c *ReviewCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prNumber := args
	if prNumber == "" {
		prNumber = "no PR number provided"
	}

	prompt := fmt.Sprintf(`You are an expert code reviewer. Follow these steps:

1. If no PR number is provided in the args, run `+"`gh pr list`"+` to show open PRs
2. If a PR number is provided, run `+"`gh pr view <number>`"+` to get PR details
3. Run `+"`gh pr diff <number>`"+` to get the diff
4. Analyze the changes and provide a thorough code review that includes:
   - Overview of what the PR does
   - Analysis of code quality and style
   - Specific suggestions for improvements
   - Any potential issues or risks

Keep your review concise but thorough. Focus on:
- Code correctness
- Following project conventions
- Performance implications
- Test coverage
- Security considerations

Format your review with clear sections and bullet points.

PR number: %s`, prNumber)

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// BriefCommand provides a brief summary.
type BriefCommand struct{}

func NewBriefCommand() *BriefCommand { return &BriefCommand{} }

func (c *BriefCommand) Name() string        { return "brief" }
func (c *BriefCommand) Description() string { return "Provide a brief summary" }
func (c *BriefCommand) IsEnabled() bool     { return true }
func (c *BriefCommand) IsHidden() bool      { return false }

func (c *BriefCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := `Provide a brief, concise summary of the current context and recent changes. Focus on:
- Key files modified
- Main functionality affected
- Important decisions made

Keep the summary under 200 words.`

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// AdvisorCommand provides code advice.
type AdvisorCommand struct{}

func NewAdvisorCommand() *AdvisorCommand { return &AdvisorCommand{} }

func (c *AdvisorCommand) Name() string        { return "advisor" }
func (c *AdvisorCommand) Description() string { return "Get code advice and best practices" }
func (c *AdvisorCommand) IsEnabled() bool     { return true }
func (c *AdvisorCommand) IsHidden() bool      { return false }

func (c *AdvisorCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := fmt.Sprintf(`You are a code advisor. Provide helpful guidance and best practices for:

%s

Focus on:
- Code quality improvements
- Performance optimizations
- Security considerations
- Maintainability enhancements
- Testing recommendations

Provide actionable, specific advice with examples where helpful.`, args)

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// InsightsCommand provides code insights.
type InsightsCommand struct{}

func NewInsightsCommand() *InsightsCommand { return &InsightsCommand{} }

func (c *InsightsCommand) Name() string        { return "insights" }
func (c *InsightsCommand) Description() string { return "Analyze code for insights" }
func (c *InsightsCommand) IsEnabled() bool     { return true }
func (c *InsightsCommand) IsHidden() bool      { return false }

func (c *InsightsCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := `Analyze the current codebase and provide insights:

1. Architecture patterns and design decisions
2. Code complexity hotspots
3. Technical debt areas
4. Dependency relationships
5. Potential refactoring opportunities

Focus on actionable insights that would help improve the codebase quality.`

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// SecurityReviewCommand performs security analysis.
type SecurityReviewCommand struct{}

func NewSecurityReviewCommand() *SecurityReviewCommand { return &SecurityReviewCommand{} }

func (c *SecurityReviewCommand) Name() string        { return "security-review" }
func (c *SecurityReviewCommand) Description() string { return "Perform a security review" }
func (c *SecurityReviewCommand) IsEnabled() bool     { return true }
func (c *SecurityReviewCommand) IsHidden() bool      { return false }

func (c *SecurityReviewCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := `Perform a comprehensive security review of the codebase:

1. **Input Validation**: Check for unvalidated user inputs
2. **Authentication & Authorization**: Review access control mechanisms
3. **Data Protection**: Analyze how sensitive data is handled
4. **Injection Vulnerabilities**: Look for SQL injection, XSS, command injection
5. **Dependencies**: Check for known vulnerable dependencies
6. **Secrets Management**: Identify hardcoded credentials or API keys
7. **Encryption**: Verify proper use of cryptographic functions
8. **Error Handling**: Check for information disclosure in errors

Provide specific findings with severity levels and remediation recommendations.`

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// ExportCommand exports session data.
type ExportCommand struct{}

func NewExportCommand() *ExportCommand { return &ExportCommand{} }

func (c *ExportCommand) Name() string        { return "export" }
func (c *ExportCommand) Description() string { return "Export session data" }
func (c *ExportCommand) IsEnabled() bool     { return true }
func (c *ExportCommand) IsHidden() bool      { return false }

func (c *ExportCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	// TODO: Implement actual export functionality
	return &CommandResult{
		Type:  "text",
		Value: "Export functionality not yet implemented. Will export session transcript and metadata.",
	}, nil
}

// =============================================================================
// FeedbackCommand submits feedback.
type FeedbackCommand struct{}

func NewFeedbackCommand() *FeedbackCommand { return &FeedbackCommand{} }

func (c *FeedbackCommand) Name() string        { return "feedback" }
func (c *FeedbackCommand) Description() string { return "Submit feedback about Claude Code" }
func (c *FeedbackCommand) IsEnabled() bool     { return true }
func (c *FeedbackCommand) IsHidden() bool      { return false }

func (c *FeedbackCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	prompt := `Thank you for providing feedback! Please describe:
- What you were trying to do
- What worked well
- What could be improved
- Any bugs or issues you encountered

Your feedback helps us improve Claude Code.`
	if args != "" {
		prompt += fmt.Sprintf("\n\nFeedback context: %s", args)
	}

	return &CommandResult{
		Type:  "text",
		Value: prompt,
	}, nil
}

// =============================================================================
// VersionCommand shows version information.
type VersionCommand struct {
	version string
}

func NewVersionCommand(version string) *VersionCommand {
	return &VersionCommand{version: version}
}

func (c *VersionCommand) Name() string        { return "version" }
func (c *VersionCommand) Description() string { return "Show version information" }
func (c *VersionCommand) IsEnabled() bool     { return true }
func (c *VersionCommand) IsHidden() bool      { return false }

func (c *VersionCommand) Execute(ctx context.Context, args string, context *CommandContext) (*CommandResult, error) {
	return &CommandResult{
		Type:  "text",
		Value: fmt.Sprintf("Claude Code Go version: %s", c.version),
	}, nil
}
