package services

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Compact Prompt Templates
// =============================================================================

// NO_TOOLS_PREAMBLE is the aggressive no-tools preamble for compact requests.
// The cache-sharing fork path inherits the parent's full tool set (required for
// cache-key match), and on Sonnet 4.6+ adaptive-thinking models the model
// sometimes attempts a tool call despite the weaker trailer instruction.
// With maxTurns: 1, a denied tool call means no text output → falls through
// to the streaming fallback. Putting this FIRST and making it explicit about
// rejection consequences prevents the wasted turn.
const NO_TOOLS_PREAMBLE = `CRITICAL: Respond with TEXT ONLY. Do NOT call any tools.

- Do NOT use Read, Bash, Grep, Glob, Edit, Write, or ANY other tool.
- You already have all the context you need in the conversation above.
- Tool calls will be REJECTED and will waste your only turn — you will fail the task.
- Your entire response must be plain text: an <analysis> block followed by a <summary> block.

`

// DETAILED_ANALYSIS_INSTRUCTION_BASE is the base analysis instruction template.
// Two variants: BASE scopes to "the conversation", PARTIAL scopes to "the recent messages".
// The <analysis> block is a drafting scratchpad that formatCompactSummary() strips
// before the summary reaches context.
const DETAILED_ANALYSIS_INSTRUCTION_BASE = `Before providing your final summary, wrap your analysis in <analysis> tags to organize your thoughts and ensure you've covered all necessary points. In your analysis process:

1. Chronologically analyze each message and section of the conversation. For each section thoroughly identify:
   - The user's explicit requests and intents
   - Your approach to addressing the user's requests
   - Key decisions, technical concepts and code patterns
   - Specific details like:
     - file names
     - full code snippets
     - function signatures
     - file edits
   - Errors that you ran into and how you fixed them
   - Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
2. Double-check for technical accuracy and completeness, addressing each required element thoroughly.`

// DETAILED_ANALYSIS_INSTRUCTION_PARTIAL is the partial analysis instruction template
const DETAILED_ANALYSIS_INSTRUCTION_PARTIAL = `Before providing your final summary, wrap your analysis in <analysis> tags to organize your thoughts and ensure you've covered all necessary points. In your analysis process:

1. Analyze the recent messages chronologically. For each section thoroughly identify:
   - The user's explicit requests and intents
   - Your approach to addressing the user's requests
   - Key decisions, technical concepts and code patterns
   - Specific details like:
     - file names
     - full code snippets
     - function signatures
     - file edits
   - Errors that you ran into and how you fixed them
   - Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
2. Double-check for technical accuracy and completeness, addressing each required element thoroughly.`

// BASE_COMPACT_PROMPT is the base compact prompt template
const BASE_COMPACT_PROMPT = `Your task is to create a detailed summary of the conversation so far, paying close attention to the user's explicit requests and your previous actions.
This summary should be thorough in capturing technical details, code patterns, and architectural decisions that would be essential for continuing development work without losing context.

{{DETAILED_ANALYSIS_INSTRUCTION}}

Your summary should include the following sections:

1. Primary Request and Intent: Capture all of the user's explicit requests and intents in detail
2. Key Technical Concepts: List all important technical concepts, technologies, and frameworks discussed.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Pay special attention to the most recent messages and include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List all errors that you ran into, and how you fixed them. Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages that are not tool results. These are critical for understanding the users' feedback and changing intent.
7. Pending Tasks: Outline any pending tasks that you have explicitly been asked to work on.
8. Current Work: Describe in detail precisely what was being worked on immediately before this summary request, paying special attention to the most recent messages from both user and assistant. Include file names and code snippets where applicable.
9. Optional Next Step: List the next step that you will take that is related to the most recent work you were doing. IMPORTANT: ensure that this step is DIRECTLY in line with the user's most recent explicit requests, and the task you were working on immediately before this summary request. If your last task was concluded, then only list next steps if they are explicitly in line with the users request. Do not start on tangential requests or really old requests that were already completed without confirming with the user first.
                       If there is a next step, include direct quotes from the most recent conversation showing exactly what task you were working on and where you left off. This should be verbatim to ensure there's no drift in task interpretation.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]
   - [...]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Summary of the changes made to this file, if any]
      - [Important Code Snippet]
   - [File Name 2]
      - [Important Code Snippet]
   - [...]

4. Errors and fixes:
    - [Detailed description of error 1]:
      - [How you fixed the error]
      - [User feedback on the error if any]
    - [...]

5. Problem Solving:
   [Description of solved problems and ongoing troubleshooting]

6. All user messages:
    - [Detailed non tool use user message]
    - [...]

7. Pending Tasks:
   - [Task 1]
   - [Task 2]
   - [...]

8. Current Work:
   [Precise description of current work]

9. Optional Next Step:
   [Optional Next step to take]

</summary>
</example>

Please provide your summary based on the conversation so far, following this structure and ensuring precision and thoroughness in your response.

There may be additional summarization instructions provided in the included context. If so, remember to follow these instructions when creating your summary. Examples of instructions include:
<example>
## Compact Instructions
When summarizing the conversation focus on typescript code changes and also remember the mistakes you made and how you fixed them.
</example>

<example>
# Summary instructions
When you are using compact - please focus on test output and code changes. Include file reads verbatim.
</example>
`

// PARTIAL_COMPACT_PROMPT is the partial compact prompt template for 'from' direction
const PARTIAL_COMPACT_PROMPT = `Your task is to create a detailed summary of the RECENT portion of the conversation — the messages that follow earlier retained context. The earlier messages are being kept intact and do NOT need to be summarized. Focus your summary on what was discussed, learned, and accomplished in the recent messages only.

{{DETAILED_ANALYSIS_INSTRUCTION}}

Your summary should include the following sections:

1. Primary Request and Intent: Capture the user's explicit requests and intents from the recent messages
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks discussed recently.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List errors encountered and how they were fixed.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages from the recent portion that are not tool results.
7. Pending Tasks: Outline any pending tasks from the recent messages.
8. Current Work: Describe precisely what was being worked on immediately before this summary request.
9. Optional Next Step: List the next step related to the most recent work. Include direct quotes from the most recent conversation.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Important Code Snippet]

4. Errors and fixes:
    - [Error description]:
      - [How you fixed it]

5. Problem Solving:
   [Description]

6. All user messages:
    - [Detailed non tool use user message]

7. Pending Tasks:
   - [Task 1]

8. Current Work:
   [Precise description of current work]

9. Optional Next Step:
   [Optional Next step to take]

</summary>
</example>

Please provide your summary based on the RECENT messages only (after the retained earlier context), following this structure and ensuring precision and thoroughness in your response.
`

// PARTIAL_COMPACT_UP_TO_PROMPT is the partial compact prompt for 'up_to' direction.
// 'up_to': model sees only the summarized prefix (cache hit). Summary will
// precede kept recent messages, hence "Context for Continuing Work" section.
const PARTIAL_COMPACT_UP_TO_PROMPT = `Your task is to create a detailed summary of this conversation. This summary will be placed at the start of a continuing session; newer messages that build on this context will follow after your summary (you do not see them here). Summarize thoroughly so that someone reading only your summary and then the newer messages can fully understand what happened and continue the work.

{{DETAILED_ANALYSIS_INSTRUCTION}}

Your summary should include the following sections:

1. Primary Request and Intent: Capture the user's explicit requests and intents in detail
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks discussed.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List errors encountered and how they were fixed.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages that are not tool results.
7. Pending Tasks: Outline any pending tasks.
8. Work Completed: Describe what was accomplished by the end of this portion.
9. Context for Continuing Work: Summarize any context, decisions, or state that would be needed to understand and continue the work in subsequent messages.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Important Code Snippet]

4. Errors and fixes:
    - [Error description]:
      - [How you fixed it]

5. Problem Solving:
   [Description]

6. All user messages:
    - [Detailed non tool use user message]

7. Pending Tasks:
   - [Task 1]

8. Work Completed:
   [Description of what was accomplished]

9. Context for Continuing Work:
   [Key context, decisions, or state needed to continue the work]

</summary>
</example>

Please provide your summary following this structure, ensuring precision and thoroughness in your response.
`

// NO_TOOLS_TRAILER is the reminder at the end of compact prompts
const NO_TOOLS_TRAILER = `

REMINDER: Do NOT call any tools. Respond with plain text only — an <analysis> block followed by a <summary> block. Tool calls will be rejected and you will fail the task.`

// GetPartialCompactPrompt returns the partial compact prompt based on direction
func GetPartialCompactPrompt(customInstructions *string, direction CompactDirection) string {
	template := PARTIAL_COMPACT_PROMPT
	if direction == CompactDirectionUpTo {
		template = PARTIAL_COMPACT_UP_TO_PROMPT
	}

	// Replace placeholder with appropriate analysis instruction
	template = strings.Replace(template, "{{DETAILED_ANALYSIS_INSTRUCTION}}", DETAILED_ANALYSIS_INSTRUCTION_PARTIAL, 1)

	prompt := NO_TOOLS_PREAMBLE + template

	if customInstructions != nil && strings.TrimSpace(*customInstructions) != "" {
		prompt += fmt.Sprintf("\n\nAdditional Instructions:\n%s", *customInstructions)
	}

	prompt += NO_TOOLS_TRAILER

	return prompt
}

// GetCompactPrompt returns the base compact prompt
func GetCompactPrompt(customInstructions *string) string {
	// Replace placeholder with base analysis instruction
	template := strings.Replace(BASE_COMPACT_PROMPT, "{{DETAILED_ANALYSIS_INSTRUCTION}}", DETAILED_ANALYSIS_INSTRUCTION_BASE, 1)

	prompt := NO_TOOLS_PREAMBLE + template

	if customInstructions != nil && strings.TrimSpace(*customInstructions) != "" {
		prompt += fmt.Sprintf("\n\nAdditional Instructions:\n%s", *customInstructions)
	}

	prompt += NO_TOOLS_TRAILER

	return prompt
}

// FormatCompactSummary formats the compact summary by stripping the <analysis>
// drafting scratchpad and replacing <summary> XML tags with readable section headers.
func FormatCompactSummary(summary string) string {
	formattedSummary := summary

	// Strip analysis section — it's a drafting scratchpad that improves summary
	// quality but has no informational value once the summary is written.
	analysisRegex := regexp.MustCompile(`(?s)<analysis>.*?</analysis>`)
	formattedSummary = analysisRegex.ReplaceAllString(formattedSummary, "")

	// Extract and format summary section
	summaryRegex := regexp.MustCompile(`(?s)<summary>(.*?)</summary>`)
	if match := summaryRegex.FindStringSubmatch(formattedSummary); match != nil {
		content := strings.TrimSpace(match[1])
		formattedSummary = summaryRegex.ReplaceAllString(formattedSummary, fmt.Sprintf("Summary:\n%s", content))
	}

	// Clean up extra whitespace between sections
	whitespaceRegex := regexp.MustCompile(`\n\n+`)
	formattedSummary = whitespaceRegex.ReplaceAllString(formattedSummary, "\n\n")

	return strings.TrimSpace(formattedSummary)
}

// GetCompactUserSummaryMessage creates the user-facing summary message
func GetCompactUserSummaryMessage(
	summary string,
	suppressFollowUpQuestions bool,
	transcriptPath *string,
	recentMessagesPreserved bool,
) string {
	formattedSummary := FormatCompactSummary(summary)

	baseSummary := fmt.Sprintf(`This session is being continued from a previous conversation that ran out of context. The summary below covers the earlier portion of the conversation.

%s`, formattedSummary)

	if transcriptPath != nil && *transcriptPath != "" {
		baseSummary += fmt.Sprintf("\n\nIf you need specific details from before compaction (like exact code snippets, error messages, or content you generated), read the full transcript at: %s", *transcriptPath)
	}

	if recentMessagesPreserved {
		baseSummary += "\n\nRecent messages are preserved verbatim."
	}

	if suppressFollowUpQuestions {
		continuation := fmt.Sprintf(`%s
Continue the conversation from where it left off without asking the user any further questions. Resume directly — do not acknowledge the summary, do not recap what was happening, do not preface with "I'll continue" or similar. Pick up the last task as if the break never happened.`, baseSummary)

		return continuation
	}

	return baseSummary
}

// =============================================================================
// Compact Message Builder
// =============================================================================

// CompactPromptBuilder builds compact prompts with various options
type CompactPromptBuilder struct {
	customInstructions   *string
	direction            CompactDirection
	isPartial            bool
	includeAnalysisBlock bool
}

// NewCompactPromptBuilder creates a new compact prompt builder
func NewCompactPromptBuilder() *CompactPromptBuilder {
	return &CompactPromptBuilder{
		includeAnalysisBlock: true,
	}
}

// WithCustomInstructions sets custom instructions
func (b *CompactPromptBuilder) WithCustomInstructions(instructions string) *CompactPromptBuilder {
	b.customInstructions = &instructions
	return b
}

// WithDirection sets the compact direction
func (b *CompactPromptBuilder) WithDirection(direction CompactDirection) *CompactPromptBuilder {
	b.direction = direction
	b.isPartial = true
	return b
}

// WithPartial sets whether this is a partial compact
func (b *CompactPromptBuilder) WithPartial(isPartial bool) *CompactPromptBuilder {
	b.isPartial = isPartial
	return b
}

// Build builds the compact prompt
func (b *CompactPromptBuilder) Build() string {
	if b.isPartial {
		return GetPartialCompactPrompt(b.customInstructions, b.direction)
	}
	return GetCompactPrompt(b.customInstructions)
}
