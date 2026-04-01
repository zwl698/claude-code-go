package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"claude-code-go/internal/types"
)

// =============================================================================
// Compact Service - Conversation Compaction
// =============================================================================

const (
	// PostCompactMaxFilesToRestore is the maximum number of files to restore after compaction
	PostCompactMaxFilesToRestore = 5
	// PostCompactTokenBudget is the token budget for post-compact file attachments
	PostCompactTokenBudget = 50000
	// PostCompactMaxTokensPerFile is the max tokens per file for post-compact restoration
	PostCompactMaxTokensPerFile = 5000
	// PostCompactMaxTokensPerSkill is the max tokens per skill for post-compact restoration
	PostCompactMaxTokensPerSkill = 5000
	// PostCompactSkillsTokenBudget is the total token budget for skills
	PostCompactSkillsTokenBudget = 25000
	// MaxCompactStreamingRetries is the max retries for compact streaming
	MaxCompactStreamingRetries = 2
	// MaxPTLRetries is the max retries for prompt-too-long errors
	MaxPTLRetries = 3
)

// Error messages
const (
	ErrorMessageNotEnoughMessages  = "Not enough messages to compact."
	ErrorMessagePromptTooLong      = "Conversation too long. Press esc twice to go up a few messages and try again."
	ErrorMessageUserAbort          = "API Error: Request was aborted."
	ErrorMessageIncompleteResponse = "Compaction interrupted · This may be due to network issues — please try again."
)

// PTLRetryMarker is the marker for prompt-too-long retry
const PTLRetryMarker = "[earlier conversation truncated for compaction retry]"

// CompactDirection represents the direction of partial compaction
type CompactDirection string

const (
	CompactDirectionFrom CompactDirection = "from"
	CompactDirectionUpTo CompactDirection = "up_to"
)

// CompactTrigger represents the trigger type for compaction
type CompactTrigger string

const (
	CompactTriggerAuto   CompactTrigger = "auto"
	CompactTriggerManual CompactTrigger = "manual"
)

// CompactMessage represents a message in the compact system
type CompactMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
	IsMeta    bool            `json:"isMeta,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Timestamp float64         `json:"timestamp,omitempty"`
}

// CompactionResult represents the result of a compaction operation
type CompactionResult struct {
	BoundaryMarker            *CompactMessage
	SummaryMessages           []*CompactMessage
	Attachments               []*CompactMessage
	HookResults               []*CompactMessage
	MessagesToKeep            []*CompactMessage
	UserDisplayMessage        string
	PreCompactTokenCount      int
	PostCompactTokenCount     int
	TruePostCompactTokenCount int
	CompactionUsage           *TokenUsage
}

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// RecompactionInfo contains diagnosis context for auto-compact
type RecompactionInfo struct {
	IsRecompactionInChain     bool
	TurnsSincePreviousCompact int
	PreviousCompactTurnID     string
	AutoCompactThreshold      int
	QuerySource               string
}

// CompactService handles conversation compaction
type CompactService struct {
	mu                sync.RWMutex
	readFileState     map[string]*FileState
	loadedMemoryPaths map[string]bool
	sessionTranscript SessionTranscriptWriter
	hookExecutor      HookExecutor
	tokenEstimator    TokenEstimator
}

// FileState tracks file read state
type FileState struct {
	Content   string
	Timestamp int64
}

// SessionTranscriptWriter is an interface for writing session transcripts
type SessionTranscriptWriter interface {
	WriteSegment(messages []*CompactMessage) error
}

// HookExecutor is an interface for executing hooks
type HookExecutor interface {
	ExecutePreCompact(ctx context.Context, trigger CompactTrigger, customInstructions string) (*PreCompactHookResult, error)
	ExecutePostCompact(ctx context.Context, trigger CompactTrigger, summary string) (*PostCompactHookResult, error)
	ExecuteSessionStart(ctx context.Context, model string) ([]*CompactMessage, error)
}

// PreCompactHookResult is the result of pre-compact hooks
type PreCompactHookResult struct {
	NewCustomInstructions string
	UserDisplayMessage    string
}

// PostCompactHookResult is the result of post-compact hooks
type PostCompactHookResult struct {
	UserDisplayMessage string
}

// TokenEstimator estimates token counts
type TokenEstimator interface {
	EstimateTokens(content string) int
	EstimateTokensForMessages(messages []*CompactMessage) int
}

// NewCompactService creates a new compact service
func NewCompactService() *CompactService {
	return &CompactService{
		readFileState:     make(map[string]*FileState),
		loadedMemoryPaths: make(map[string]bool),
	}
}

// CompactConversation creates a compact version of a conversation
func (s *CompactService) CompactConversation(
	ctx context.Context,
	messages []*CompactMessage,
	customInstructions string,
	isAutoCompact bool,
	recompactionInfo *RecompactionInfo,
	onProgress func(progress interface{}),
) (*CompactionResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf(ErrorMessageNotEnoughMessages)
	}

	preCompactTokenCount := s.tokenEstimator.EstimateTokensForMessages(messages)

	// Execute pre-compact hooks
	if s.hookExecutor != nil {
		hookResult, err := s.hookExecutor.ExecutePreCompact(ctx, CompactTriggerManual, customInstructions)
		if err != nil {
			return nil, fmt.Errorf("pre-compact hook failed: %w", err)
		}
		if hookResult != nil {
			customInstructions = mergeHookInstructions(customInstructions, hookResult.NewCustomInstructions)
		}
	}

	// Strip images from messages for compaction
	strippedMessages := stripImagesFromMessages(messages)
	strippedMessages = stripReinjectedAttachments(strippedMessages)

	// Generate compact summary
	summary, err := s.streamCompactSummary(ctx, strippedMessages, customInstructions)
	if err != nil {
		return nil, err
	}

	if summary == "" {
		return nil, fmt.Errorf("failed to generate conversation summary")
	}

	// Clear file state cache
	s.mu.Lock()
	s.readFileState = make(map[string]*FileState)
	s.loadedMemoryPaths = make(map[string]bool)
	s.mu.Unlock()

	// Create post-compact attachments
	fileAttachments := s.createPostCompactFileAttachments(preCompactTokenCount)

	// Create boundary marker
	boundaryMarker := createCompactBoundaryMessage(
		CompactTriggerManual,
		preCompactTokenCount,
		messages[len(messages)-1].UUID,
	)

	// Create summary message
	summaryMessages := []*CompactMessage{
		createCompactUserMessage(summary, true, true),
	}

	// Execute session start hooks
	var hookMessages []*CompactMessage
	if s.hookExecutor != nil {
		hookMessages, _ = s.hookExecutor.ExecuteSessionStart(ctx, "default")
	}

	// Execute post-compact hooks
	var userDisplayMessage string
	if s.hookExecutor != nil {
		postResult, _ := s.hookExecutor.ExecutePostCompact(ctx, CompactTriggerManual, summary)
		if postResult != nil {
			userDisplayMessage = postResult.UserDisplayMessage
		}
	}

	return &CompactionResult{
		BoundaryMarker:        boundaryMarker,
		SummaryMessages:       summaryMessages,
		Attachments:           fileAttachments,
		HookResults:           hookMessages,
		UserDisplayMessage:    userDisplayMessage,
		PreCompactTokenCount:  preCompactTokenCount,
		PostCompactTokenCount: s.tokenEstimator.EstimateTokens(summary),
	}, nil
}

// PartialCompactConversation performs a partial compaction around a pivot index
func (s *CompactService) PartialCompactConversation(
	ctx context.Context,
	allMessages []*CompactMessage,
	pivotIndex int,
	direction CompactDirection,
	userFeedback string,
	customInstructions string,
	onProgress func(progress interface{}),
) (*CompactionResult, error) {
	var messagesToSummarize []*CompactMessage
	var messagesToKeep []*CompactMessage

	if direction == CompactDirectionUpTo {
		messagesToSummarize = allMessages[:pivotIndex]
		// Filter out old compact boundaries and summaries for 'up_to' direction
		for _, m := range allMessages[pivotIndex:] {
			if m.Type != "progress" && !isCompactBoundaryMessage(m) && !isCompactSummaryMessage(m) {
				messagesToKeep = append(messagesToKeep, m)
			}
		}
	} else {
		messagesToSummarize = allMessages[pivotIndex:]
		for _, m := range allMessages[:pivotIndex] {
			if m.Type != "progress" {
				messagesToKeep = append(messagesToKeep, m)
			}
		}
	}

	if len(messagesToSummarize) == 0 {
		return nil, fmt.Errorf("nothing to summarize %s the selected message",
			map[CompactDirection]string{
				CompactDirectionUpTo: "before",
				CompactDirectionFrom: "after",
			}[direction])
	}

	preCompactTokenCount := s.tokenEstimator.EstimateTokensForMessages(allMessages)

	// Merge user feedback with custom instructions
	if userFeedback != "" {
		if customInstructions != "" {
			customInstructions = fmt.Sprintf("%s\n\nUser context: %s", customInstructions, userFeedback)
		} else {
			customInstructions = fmt.Sprintf("User context: %s", userFeedback)
		}
	}

	// Generate partial compact summary
	summary, err := s.streamCompactSummary(ctx, messagesToSummarize, customInstructions)
	if err != nil {
		return nil, err
	}

	// Create boundary marker
	boundaryMarker := createCompactBoundaryMessage(
		CompactTriggerManual,
		preCompactTokenCount,
		findLastNonProgressUUID(allMessages, pivotIndex, direction),
	)

	// Create summary message
	summaryMessages := []*CompactMessage{
		createCompactUserMessage(summary, true, len(messagesToKeep) > 0),
	}

	return &CompactionResult{
		BoundaryMarker:       boundaryMarker,
		SummaryMessages:      summaryMessages,
		MessagesToKeep:       messagesToKeep,
		PreCompactTokenCount: preCompactTokenCount,
	}, nil
}

// streamCompactSummary streams the compact summary
func (s *CompactService) streamCompactSummary(
	ctx context.Context,
	messages []*CompactMessage,
	customInstructions string,
) (string, error) {
	// Build compact prompt
	prompt := buildCompactPrompt(customInstructions, messages)

	// In a real implementation, this would call the API
	// For now, we return a placeholder
	summary := fmt.Sprintf("Summary of %d messages. %s", len(messages), prompt[:min(100, len(prompt))])

	return summary, nil
}

// createPostCompactFileAttachments creates file attachments for post-compact restoration
func (s *CompactService) createPostCompactFileAttachments(tokenBudget int) []*CompactMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var attachments []*CompactMessage
	usedTokens := 0

	// Get recent files sorted by timestamp
	recentFiles := make([]struct {
		path      string
		content   string
		timestamp int64
	}, 0, len(s.readFileState))

	for path, state := range s.readFileState {
		recentFiles = append(recentFiles, struct {
			path      string
			content   string
			timestamp int64
		}{path, state.Content, state.Timestamp})
	}

	// Sort by timestamp descending
	for i := 0; i < len(recentFiles); i++ {
		for j := i + 1; j < len(recentFiles); j++ {
			if recentFiles[j].timestamp > recentFiles[i].timestamp {
				recentFiles[i], recentFiles[j] = recentFiles[j], recentFiles[i]
			}
		}
	}

	// Create attachments within budget
	for i := 0; i < len(recentFiles) && i < PostCompactMaxFilesToRestore; i++ {
		tokens := s.tokenEstimator.EstimateTokens(recentFiles[i].content)
		if usedTokens+tokens <= tokenBudget {
			attachments = append(attachments, &CompactMessage{
				Type:     "attachment",
				Content:  json.RawMessage(fmt.Sprintf(`"%s"`, recentFiles[i].content)),
				Metadata: json.RawMessage(fmt.Sprintf(`{"file_path": "%s"}`, recentFiles[i].path)),
			})
			usedTokens += tokens
		}
	}

	return attachments
}

// SetReadFileState sets the file read state
func (s *CompactService) SetReadFileState(path, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readFileState[path] = &FileState{
		Content:   content,
		Timestamp: time.Now().UnixNano(),
	}
}

// GetReadFileState gets the file read state
func (s *CompactService) GetReadFileState(path string) (*FileState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.readFileState[path]
	return state, ok
}

// ClearReadFileState clears all file read state
func (s *CompactService) ClearReadFileState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readFileState = make(map[string]*FileState)
}

// SetHookExecutor sets the hook executor
func (s *CompactService) SetHookExecutor(executor HookExecutor) {
	s.hookExecutor = executor
}

// SetTokenEstimator sets the token estimator
func (s *CompactService) SetTokenEstimator(estimator TokenEstimator) {
	s.tokenEstimator = estimator
}

// SetSessionTranscript sets the session transcript writer
func (s *CompactService) SetSessionTranscript(writer SessionTranscriptWriter) {
	s.sessionTranscript = writer
}

// ConvertToCompactMessages converts types.Message to CompactMessage
func ConvertToCompactMessages(messages []*types.Message) []*CompactMessage {
	result := make([]*CompactMessage, len(messages))
	for i, msg := range messages {
		result[i] = &CompactMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			ID:        msg.Id,
			Timestamp: msg.Timestamp,
		}
	}
	return result
}

// ConvertFromCompactMessages converts CompactMessage to types.Message
func ConvertFromCompactMessages(messages []*CompactMessage) []*types.Message {
	result := make([]*types.Message, len(messages))
	for i, msg := range messages {
		result[i] = &types.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Id:        msg.ID,
			Timestamp: msg.Timestamp,
		}
	}
	return result
}

// =============================================================================
// Helper Functions
// =============================================================================

// stripImagesFromMessages strips image blocks from messages
func stripImagesFromMessages(messages []*CompactMessage) []*CompactMessage {
	result := make([]*CompactMessage, len(messages))
	for i, msg := range messages {
		if msg.Role != "user" {
			result[i] = msg
			continue
		}

		// Process content to strip images
		var contentStr string
		if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
			result[i] = msg
			continue
		}

		var contentArr []interface{}
		if err := json.Unmarshal(msg.Content, &contentArr); err == nil {
			newContent := make([]interface{}, 0, len(contentArr))
			for _, block := range contentArr {
				if blockMap, ok := block.(map[string]interface{}); ok {
					if blockType, ok := blockMap["type"].(string); ok {
						if blockType == "image" {
							newContent = append(newContent, map[string]interface{}{
								"type": "text",
								"text": "[image]",
							})
							continue
						}
						if blockType == "document" {
							newContent = append(newContent, map[string]interface{}{
								"type": "text",
								"text": "[document]",
							})
							continue
						}
					}
				}
				newContent = append(newContent, block)
			}
			if data, err := json.Marshal(newContent); err == nil {
				newMsg := *msg
				newMsg.Content = json.RawMessage(data)
				result[i] = &newMsg
				continue
			}
		}

		result[i] = msg
	}
	return result
}

// stripReinjectedAttachments strips attachments that will be re-injected
func stripReinjectedAttachments(messages []*CompactMessage) []*CompactMessage {
	result := make([]*CompactMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Type == "attachment" {
			var metadata map[string]interface{}
			if err := json.Unmarshal(msg.Metadata, &metadata); err == nil {
				if attType, ok := metadata["type"].(string); ok {
					if attType == "skill_discovery" || attType == "skill_listing" {
						continue
					}
				}
			}
		}
		result = append(result, msg)
	}
	return result
}

// truncateHeadForPTLRetry truncates the head of messages for PTL retry
func truncateHeadForPTLRetry(messages []*CompactMessage, ptlResponse *CompactMessage) []*CompactMessage {
	if len(messages) < 2 {
		return nil
	}

	// Strip PTL retry marker if present
	input := messages
	if len(messages) > 0 && messages[0].Role == "user" {
		var contentStr string
		if err := json.Unmarshal(messages[0].Content, &contentStr); err == nil && contentStr == PTLRetryMarker {
			input = messages[1:]
		}
	}

	// Group messages by API round
	groups := groupMessagesByApiRound(input)
	if len(groups) < 2 {
		return nil
	}

	// Calculate drop count based on token gap
	dropCount := 1
	if len(groups) > 5 {
		dropCount = len(groups) / 5
	}

	// Keep at least one group
	if dropCount >= len(groups) {
		dropCount = len(groups) - 1
	}
	if dropCount < 1 {
		return nil
	}

	sliced := make([]*CompactMessage, 0)
	for _, g := range groups[dropCount:] {
		sliced = append(sliced, g...)
	}

	// Prepend synthetic user marker if assistant-first
	if len(sliced) > 0 && sliced[0].Role == "assistant" {
		sliced = append([]*CompactMessage{{
			Role:    "user",
			Content: json.RawMessage(fmt.Sprintf(`"%s"`, PTLRetryMarker)),
			IsMeta:  true,
		}}, sliced...)
	}

	return sliced
}

// groupMessagesByApiRound groups messages by API round
func groupMessagesByApiRound(messages []*CompactMessage) [][]*CompactMessage {
	groups := make([][]*CompactMessage, 0)
	currentGroup := make([]*CompactMessage, 0)

	for _, msg := range messages {
		currentGroup = append(currentGroup, msg)
		if msg.Role == "assistant" {
			groups = append(groups, currentGroup)
			currentGroup = make([]*CompactMessage, 0)
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// mergeHookInstructions merges user and hook instructions
func mergeHookInstructions(userInstructions, hookInstructions string) string {
	if hookInstructions == "" {
		return userInstructions
	}
	if userInstructions == "" {
		return hookInstructions
	}
	return userInstructions + "\n\n" + hookInstructions
}

// createCompactBoundaryMessage creates a compact boundary message
func createCompactBoundaryMessage(trigger CompactTrigger, tokenCount int, lastUUID string) *CompactMessage {
	content := map[string]interface{}{
		"type":                    "compact_boundary",
		"trigger":                 string(trigger),
		"pre_compact_token_count": tokenCount,
		"last_message_uuid":       lastUUID,
	}
	contentBytes, _ := json.Marshal(content)
	return &CompactMessage{
		Type:    "system",
		Content: contentBytes,
		UUID:    generateUUID(),
	}
}

// createCompactUserMessage creates a user message
func createCompactUserMessage(content string, isCompactSummary bool, isVisibleInTranscriptOnly bool) *CompactMessage {
	msg := &CompactMessage{
		Role:    "user",
		Content: json.RawMessage(fmt.Sprintf(`"%s"`, content)),
		UUID:    generateUUID(),
	}
	if isCompactSummary {
		metadata := map[string]interface{}{
			"is_compact_summary":            true,
			"is_visible_in_transcript_only": isVisibleInTranscriptOnly,
		}
		metadataBytes, _ := json.Marshal(metadata)
		msg.Metadata = metadataBytes
	}
	return msg
}

// isCompactBoundaryMessage checks if a message is a compact boundary
func isCompactBoundaryMessage(msg *CompactMessage) bool {
	if msg.Type != "system" {
		return false
	}
	var content map[string]interface{}
	if err := json.Unmarshal(msg.Content, &content); err == nil {
		if t, ok := content["type"].(string); ok && t == "compact_boundary" {
			return true
		}
	}
	return false
}

// isCompactSummaryMessage checks if a message is a compact summary
func isCompactSummaryMessage(msg *CompactMessage) bool {
	if msg.Role != "user" {
		return false
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(msg.Metadata, &metadata); err == nil {
		if isSummary, ok := metadata["is_compact_summary"].(bool); ok && isSummary {
			return true
		}
	}
	return false
}

// findLastNonProgressUUID finds the last non-progress message UUID
func findLastNonProgressUUID(messages []*CompactMessage, pivotIndex int, direction CompactDirection) string {
	if direction == CompactDirectionUpTo {
		for i := pivotIndex - 1; i >= 0; i-- {
			if messages[i].Type != "progress" {
				return messages[i].UUID
			}
		}
		return ""
	}

	// For 'from' direction
	for i := pivotIndex; i < len(messages); i++ {
		if messages[i].Type != "progress" {
			return messages[i].UUID
		}
	}
	return messages[len(messages)-1].UUID
}

// buildCompactPrompt builds the compact prompt
func buildCompactPrompt(customInstructions string, messages []*CompactMessage) string {
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation concisely:\n\n")

	if customInstructions != "" {
		sb.WriteString("Instructions: ")
		sb.WriteString(customInstructions)
		sb.WriteString("\n\n")
	}

	for i, msg := range messages {
		sb.WriteString(fmt.Sprintf("[%d] %s: ", i, msg.Role))
		var contentStr string
		if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
			if len(contentStr) > 200 {
				sb.WriteString(contentStr[:200])
				sb.WriteString("...")
			} else {
				sb.WriteString(contentStr)
			}
		} else {
			sb.WriteString(string(msg.Content))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateUUID generates a UUID
func generateUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Auto Compact Service
// =============================================================================

// AutoCompactService handles automatic compaction
type AutoCompactService struct {
	compactService *CompactService
	threshold      int
	enabled        bool
	mu             sync.RWMutex
}

// NewAutoCompactService creates a new auto compact service
func NewAutoCompactService(compactService *CompactService, threshold int) *AutoCompactService {
	return &AutoCompactService{
		compactService: compactService,
		threshold:      threshold,
		enabled:        true,
	}
}

// ShouldAutoCompact checks if auto compaction should be triggered
func (s *AutoCompactService) ShouldAutoCompact(tokenCount int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled && tokenCount >= s.threshold
}

// AutoCompact performs automatic compaction
func (s *AutoCompactService) AutoCompact(
	ctx context.Context,
	messages []*CompactMessage,
	recompactionInfo *RecompactionInfo,
	onProgress func(progress interface{}),
) (*CompactionResult, error) {
	return s.compactService.CompactConversation(
		ctx,
		messages,
		"",
		true,
		recompactionInfo,
		onProgress,
	)
}

// SetThreshold sets the auto compact threshold
func (s *AutoCompactService) SetThreshold(threshold int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threshold = threshold
}

// SetEnabled enables or disables auto compact
func (s *AutoCompactService) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
}

// =============================================================================
// Token Estimator Implementation
// =============================================================================

// DefaultTokenEstimator is a simple token estimator
type DefaultTokenEstimator struct {
	// Roughly 4 characters per token for English
	charsPerToken float64
}

// NewDefaultTokenEstimator creates a new default token estimator
func NewDefaultTokenEstimator() *DefaultTokenEstimator {
	return &DefaultTokenEstimator{
		charsPerToken: 4.0,
	}
}

// EstimateTokens estimates tokens for a string
func (e *DefaultTokenEstimator) EstimateTokens(content string) int {
	return int(float64(len(content)) / e.charsPerToken)
}

// EstimateTokensForMessages estimates tokens for messages
func (e *DefaultTokenEstimator) EstimateTokensForMessages(messages []*CompactMessage) int {
	total := 0
	for _, msg := range messages {
		total += e.EstimateTokens(string(msg.Content))
	}
	return total
}

// =============================================================================
// Micro Compact Service
// =============================================================================

// MicroCompactService handles micro compaction for small token savings
type MicroCompactService struct {
	compactService *CompactService
	config         *MicroCompactConfig
}

// MicroCompactConfig holds micro compact configuration
type MicroCompactConfig struct {
	Enabled            bool
	MinTokensToSave    int
	MaxMessagesToCheck int
}

// NewMicroCompactService creates a new micro compact service
func NewMicroCompactService(compactService *CompactService, config *MicroCompactConfig) *MicroCompactService {
	if config == nil {
		config = &MicroCompactConfig{
			Enabled:            true,
			MinTokensToSave:    1000,
			MaxMessagesToCheck: 50,
		}
	}
	return &MicroCompactService{
		compactService: compactService,
		config:         config,
	}
}

// PerformMicroCompact performs micro compaction
func (s *MicroCompactService) PerformMicroCompact(messages []*CompactMessage) ([]*CompactMessage, int, error) {
	if !s.config.Enabled || len(messages) < 2 {
		return messages, 0, nil
	}

	// Find and remove redundant content
	savedTokens := 0
	result := make([]*CompactMessage, 0, len(messages))

	for _, msg := range messages {
		// Skip system messages and keep them
		if msg.Type == "system" {
			result = append(result, msg)
			continue
		}

		// Keep user and assistant messages
		result = append(result, msg)
	}

	return result, savedTokens, nil
}
