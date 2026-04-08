package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Voice Recording
// =============================================================================

// RecordingConfig represents voice recording configuration.
type RecordingConfig struct {
	SampleRate       int           `json:"sampleRate"`
	Channels         int           `json:"channels"`
	SilenceDuration  float64       `json:"silenceDuration"`
	SilenceThreshold string        `json:"silenceThreshold"`
	MaxDuration      time.Duration `json:"maxDuration"`
}

// DefaultRecordingConfig returns default recording configuration.
func DefaultRecordingConfig() RecordingConfig {
	return RecordingConfig{
		SampleRate:       16000,
		Channels:         1,
		SilenceDuration:  2.0,
		SilenceThreshold: "3%",
		MaxDuration:      60 * time.Second,
	}
}

// Recorder represents a voice recorder.
type Recorder struct {
	config    RecordingConfig
	cmd       *exec.Cmd
	output    io.Reader
	mu        sync.Mutex
	recording bool
}

// NewRecorder creates a new voice recorder.
func NewRecorder(config RecordingConfig) *Recorder {
	return &Recorder{
		config: config,
	}
}

// Start starts recording audio.
func (r *Recorder) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.recording {
		return fmt.Errorf("already recording")
	}

	// Try different recording methods
	// Priority: sox (rec) > arecord (Linux) > ffmpeg

	var cmd *exec.Cmd
	var err error

	// Try sox first (cross-platform)
	if r.hasCommand("rec") {
		cmd = r.startSoxRecording(ctx)
	} else if r.hasCommand("arecord") {
		cmd = r.startArecordRecording(ctx)
	} else if r.hasCommand("ffmpeg") {
		cmd = r.startFFmpegRecording(ctx)
	} else {
		return fmt.Errorf("no audio recording command found (need sox, arecord, or ffmpeg)")
	}

	// Start the command
	r.output, err = cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	r.cmd = cmd
	r.recording = true

	return nil
}

// Stop stops recording and returns the audio data.
func (r *Recorder) Stop() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.recording || r.cmd == nil {
		return nil, fmt.Errorf("not recording")
	}

	// Kill the recording process
	if err := r.cmd.Process.Kill(); err != nil {
		return nil, fmt.Errorf("failed to stop recording: %w", err)
	}

	// Read all audio data
	data, err := io.ReadAll(r.output)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	r.recording = false
	r.cmd = nil

	return data, nil
}

// IsRecording returns true if currently recording.
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

// hasCommand checks if a command is available.
func (r *Recorder) hasCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// startSoxRecording starts recording with SoX rec command.
func (r *Recorder) startSoxRecording(ctx context.Context) *exec.Cmd {
	args := []string{
		"-t", "raw",
		"-r", fmt.Sprintf("%d", r.config.SampleRate),
		"-e", "signed-integer",
		"-b", "16",
		"-c", fmt.Sprintf("%d", r.config.Channels),
		"-",
		"silence", "1", "0.1", r.config.SilenceThreshold,
		"1", fmt.Sprintf("%.1f", r.config.SilenceDuration), r.config.SilenceThreshold,
	}

	return exec.CommandContext(ctx, "rec", args...)
}

// startArecordRecording starts recording with arecord (Linux ALSA).
func (r *Recorder) startArecordRecording(ctx context.Context) *exec.Cmd {
	args := []string{
		"-f", "S16_LE",
		"-r", fmt.Sprintf("%d", r.config.SampleRate),
		"-c", fmt.Sprintf("%d", r.config.Channels),
		"-t", "raw",
		"-",
	}

	return exec.CommandContext(ctx, "arecord", args...)
}

// startFFmpegRecording starts recording with ffmpeg.
func (r *Recorder) startFFmpegRecording(ctx context.Context) *exec.Cmd {
	args := []string{
		"-f", "alsa",
		"-i", "default",
		"-f", "s16le",
		"-ac", fmt.Sprintf("%d", r.config.Channels),
		"-ar", fmt.Sprintf("%d", r.config.SampleRate),
		"-",
	}

	return exec.CommandContext(ctx, "ffmpeg", args...)
}

// =============================================================================
// Speech-to-Text
// =============================================================================

// Transcriber represents a speech-to-text service.
type Transcriber interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

// MockTranscriber is a mock transcriber for testing.
type MockTranscriber struct {
	Response string
}

// Transcribe returns a mock transcription.
func (t *MockTranscriber) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	return t.Response, nil
}

// =============================================================================
// Voice Service
// =============================================================================

// Service represents the voice service.
type Service struct {
	recorder    *Recorder
	transcriber Transcriber
	mu          sync.Mutex
	enabled     bool
}

// NewService creates a new voice service.
func NewService(config RecordingConfig, transcriber Transcriber) *Service {
	return &Service{
		recorder:    NewRecorder(config),
		transcriber: transcriber,
		enabled:     false,
	}
}

// Enable enables voice input.
func (s *Service) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
}

// Disable disables voice input.
func (s *Service) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
}

// IsEnabled returns true if voice input is enabled.
func (s *Service) IsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

// StartRecording starts voice recording.
func (s *Service) StartRecording(ctx context.Context) error {
	if !s.IsEnabled() {
		return fmt.Errorf("voice input is disabled")
	}
	return s.recorder.Start(ctx)
}

// StopRecording stops recording and transcribes the audio.
func (s *Service) StopRecording(ctx context.Context) (string, error) {
	audioData, err := s.recorder.Stop()
	if err != nil {
		return "", err
	}

	if len(audioData) == 0 {
		return "", fmt.Errorf("no audio data recorded")
	}

	// Transcribe the audio
	return s.transcriber.Transcribe(ctx, audioData)
}

// IsRecording returns true if currently recording.
func (s *Service) IsRecording() bool {
	return s.recorder.IsRecording()
}

// =============================================================================
// Voice Commands
// =============================================================================

// CommandType represents the type of voice command.
type CommandType string

const (
	CommandTypeNavigation CommandType = "navigation"
	CommandTypeEdit       CommandType = "edit"
	CommandTypeSearch     CommandType = "search"
	CommandTypeSystem     CommandType = "system"
	CommandTypeInput      CommandType = "input"
	CommandTypeCode       CommandType = "code"
	CommandTypeFormat     CommandType = "format"
)

// ActionType represents the action of a voice command.
type ActionType string

const (
	ActionGoto      ActionType = "goto"
	ActionOpen      ActionType = "open"
	ActionClose     ActionType = "close"
	ActionDelete    ActionType = "delete"
	ActionRemove    ActionType = "remove"
	ActionCopy      ActionType = "copy"
	ActionCut       ActionType = "cut"
	ActionPaste     ActionType = "paste"
	ActionUndo      ActionType = "undo"
	ActionRedo      ActionType = "redo"
	ActionFind      ActionType = "find"
	ActionReplace   ActionType = "replace"
	ActionSave      ActionType = "save"
	ActionQuit      ActionType = "quit"
	ActionRun       ActionType = "run"
	ActionBuild     ActionType = "build"
	ActionTest      ActionType = "test"
	ActionFormat    ActionType = "format"
	ActionComment   ActionType = "comment"
	ActionUncomment ActionType = "uncomment"
	ActionIndent    ActionType = "indent"
	ActionDedent    ActionType = "dedent"
	ActionText      ActionType = "text"
	ActionRefactor  ActionType = "refactor"
	ActionRename    ActionType = "rename"
	ActionExtract   ActionType = "extract"
)

// VoiceCommand represents a voice command.
type VoiceCommand struct {
	Type       CommandType            `json:"type"`
	Action     ActionType             `json:"action"`
	Params     map[string]interface{} `json:"params,omitempty"`
	RawText    string                 `json:"rawText"`
	Confidence float64                `json:"confidence,omitempty"`
}

// commandPattern represents a pattern for matching voice commands.
type commandPattern struct {
	keywords  []string
	cmdType   CommandType
	action    ActionType
	extractor func(string) map[string]interface{}
}

// commandPatterns defines the patterns for voice command recognition.
var commandPatterns = []commandPattern{
	// Navigation commands
	{
		keywords:  []string{"go to", "goto", "jump to", "navigate to"},
		cmdType:   CommandTypeNavigation,
		action:    ActionGoto,
		extractor: extractTargetParams,
	},
	{
		keywords:  []string{"open file", "open"},
		cmdType:   CommandTypeNavigation,
		action:    ActionOpen,
		extractor: extractTargetParams,
	},
	{
		keywords:  []string{"close file", "close"},
		cmdType:   CommandTypeNavigation,
		action:    ActionClose,
		extractor: extractTargetParams,
	},

	// Edit commands - deletion
	{
		keywords:  []string{"delete line", "delete", "remove"},
		cmdType:   CommandTypeEdit,
		action:    ActionDelete,
		extractor: extractRangeParams,
	},
	{
		keywords:  []string{"cut"},
		cmdType:   CommandTypeEdit,
		action:    ActionCut,
		extractor: nil,
	},
	{
		keywords:  []string{"copy"},
		cmdType:   CommandTypeEdit,
		action:    ActionCopy,
		extractor: nil,
	},
	{
		keywords:  []string{"paste"},
		cmdType:   CommandTypeEdit,
		action:    ActionPaste,
		extractor: nil,
	},
	{
		keywords:  []string{"undo"},
		cmdType:   CommandTypeEdit,
		action:    ActionUndo,
		extractor: nil,
	},
	{
		keywords:  []string{"redo"},
		cmdType:   CommandTypeEdit,
		action:    ActionRedo,
		extractor: nil,
	},

	// Search commands
	{
		keywords:  []string{"search for", "search", "find", "look for"},
		cmdType:   CommandTypeSearch,
		action:    ActionFind,
		extractor: extractQueryParams,
	},
	{
		keywords:  []string{"replace", "substitute"},
		cmdType:   CommandTypeSearch,
		action:    ActionReplace,
		extractor: extractReplaceParams,
	},

	// System commands
	{
		keywords:  []string{"save", "save file"},
		cmdType:   CommandTypeSystem,
		action:    ActionSave,
		extractor: nil,
	},
	{
		keywords:  []string{"quit", "exit", "close app"},
		cmdType:   CommandTypeSystem,
		action:    ActionQuit,
		extractor: nil,
	},
	{
		keywords:  []string{"run", "execute"},
		cmdType:   CommandTypeSystem,
		action:    ActionRun,
		extractor: extractTargetParams,
	},
	{
		keywords:  []string{"build", "compile"},
		cmdType:   CommandTypeSystem,
		action:    ActionBuild,
		extractor: nil,
	},
	{
		keywords:  []string{"run tests", "test"},
		cmdType:   CommandTypeSystem,
		action:    ActionTest,
		extractor: nil,
	},

	// Code commands
	{
		keywords:  []string{"refactor"},
		cmdType:   CommandTypeCode,
		action:    ActionRefactor,
		extractor: extractTargetParams,
	},
	{
		keywords:  []string{"rename"},
		cmdType:   CommandTypeCode,
		action:    ActionRename,
		extractor: extractTargetParams,
	},
	{
		keywords:  []string{"extract function", "extract method", "extract"},
		cmdType:   CommandTypeCode,
		action:    ActionExtract,
		extractor: nil,
	},

	// Format commands
	{
		keywords:  []string{"format code", "format", "prettify"},
		cmdType:   CommandTypeFormat,
		action:    ActionFormat,
		extractor: nil,
	},
	{
		keywords:  []string{"comment"},
		cmdType:   CommandTypeFormat,
		action:    ActionComment,
		extractor: nil,
	},
	{
		keywords:  []string{"uncomment"},
		cmdType:   CommandTypeFormat,
		action:    ActionUncomment,
		extractor: nil,
	},
	{
		keywords:  []string{"indent"},
		cmdType:   CommandTypeFormat,
		action:    ActionIndent,
		extractor: nil,
	},
	{
		keywords:  []string{"dedent", "unindent"},
		cmdType:   CommandTypeFormat,
		action:    ActionDedent,
		extractor: nil,
	},
}

// ParseCommand parses a voice command from transcribed text.
// It uses pattern matching to identify commands and extract parameters.
func ParseCommand(text string) *VoiceCommand {
	originalText := text
	text = strings.ToLower(strings.TrimSpace(text))

	// Try to match against known patterns
	for _, pattern := range commandPatterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(text, keyword) {
				cmd := &VoiceCommand{
					Type:    pattern.cmdType,
					Action:  pattern.action,
					RawText: originalText,
				}

				// Extract parameters if available
				if pattern.extractor != nil {
					cmd.Params = pattern.extractor(text)
				}

				return cmd
			}
		}
	}

	// Check for numbered commands (e.g., "go to line 42")
	if cmd := parseNumberedCommand(text, originalText); cmd != nil {
		return cmd
	}

	// Check for selection commands (e.g., "select all", "select word")
	if cmd := parseSelectionCommand(text, originalText); cmd != nil {
		return cmd
	}

	// Default: treat as text input
	return &VoiceCommand{
		Type:    CommandTypeInput,
		Action:  ActionText,
		RawText: originalText,
		Params: map[string]interface{}{
			"text": originalText,
		},
	}
}

// parseNumberedCommand parses commands with numbers (e.g., "go to line 42").
func parseNumberedCommand(text, originalText string) *VoiceCommand {
	// Pattern: "go to line X" or "goto line X"
	if strings.Contains(text, "line") {
		words := strings.Fields(text)
		for i, word := range words {
			if word == "line" && i+1 < len(words) {
				// Try to parse the next word as a number
				lineNum := parseNumber(words[i+1])
				if lineNum > 0 {
					return &VoiceCommand{
						Type:    CommandTypeNavigation,
						Action:  ActionGoto,
						RawText: originalText,
						Params: map[string]interface{}{
							"line": lineNum,
						},
					}
				}
			}
		}
	}

	return nil
}

// parseSelectionCommand parses selection-related commands.
func parseSelectionCommand(text, originalText string) *VoiceCommand {
	if strings.Contains(text, "select all") {
		return &VoiceCommand{
			Type:    CommandTypeEdit,
			Action:  ActionText,
			RawText: originalText,
			Params: map[string]interface{}{
				"selection": "all",
			},
		}
	}

	if strings.Contains(text, "select word") {
		return &VoiceCommand{
			Type:    CommandTypeEdit,
			Action:  ActionText,
			RawText: originalText,
			Params: map[string]interface{}{
				"selection": "word",
			},
		}
	}

	if strings.Contains(text, "select line") {
		return &VoiceCommand{
			Type:    CommandTypeEdit,
			Action:  ActionText,
			RawText: originalText,
			Params: map[string]interface{}{
				"selection": "line",
			},
		}
	}

	return nil
}

// extractTargetParams extracts target parameters from text.
func extractTargetParams(text string) map[string]interface{} {
	words := strings.Fields(text)
	for i, word := range words {
		// Look for target after keywords
		if word == "to" || word == "open" || word == "run" || word == "rename" || word == "refactor" {
			if i+1 < len(words) {
				return map[string]interface{}{
					"target": strings.Join(words[i+1:], " "),
				}
			}
		}
	}
	return nil
}

// extractQueryParams extracts query parameters from text.
func extractQueryParams(text string) map[string]interface{} {
	words := strings.Fields(text)
	for i, word := range words {
		if word == "for" || word == "search" || word == "find" || word == "look" {
			// Find the start of the query
			startIdx := i + 1
			if word == "look" && startIdx < len(words) && words[startIdx] == "for" {
				startIdx++
			}
			if startIdx < len(words) {
				return map[string]interface{}{
					"query": strings.Join(words[startIdx:], " "),
				}
			}
		}
	}
	return nil
}

// extractReplaceParams extracts replace parameters from text.
func extractReplaceParams(text string) map[string]interface{} {
	// Pattern: "replace X with Y" or "substitute X with Y"
	if strings.Contains(text, " with ") {
		parts := strings.SplitN(text, " with ", 2)
		if len(parts) == 2 {
			// Extract what to replace from the first part
			searchPart := parts[0]
			var search string
			for _, keyword := range []string{"replace", "substitute"} {
				if idx := strings.Index(searchPart, keyword); idx != -1 {
					search = strings.TrimSpace(searchPart[idx+len(keyword):])
					break
				}
			}
			return map[string]interface{}{
				"search":  search,
				"replace": strings.TrimSpace(parts[1]),
			}
		}
	}
	return nil
}

// extractRangeParams extracts range parameters from text.
func extractRangeParams(text string) map[string]interface{} {
	params := make(map[string]interface{})

	// Check for line range (e.g., "delete lines 5 to 10")
	if strings.Contains(text, "lines") {
		words := strings.Fields(text)
		for i, word := range words {
			if word == "lines" && i+1 < len(words) {
				// Try to parse line numbers
				startLine := parseNumber(words[i+1])
				if startLine > 0 {
					params["startLine"] = startLine
					// Check for range (e.g., "lines 5 to 10")
					if i+3 < len(words) && words[i+2] == "to" {
						endLine := parseNumber(words[i+3])
						if endLine > 0 {
							params["endLine"] = endLine
						}
					}
				}
				break
			}
		}
	}

	// Check for word range (e.g., "delete word", "delete 5 words")
	if strings.Contains(text, "word") || strings.Contains(text, "words") {
		params["unit"] = "word"
		words := strings.Fields(text)
		for i, word := range words {
			if num := parseNumber(word); num > 0 && i+1 < len(words) && strings.HasPrefix(words[i+1], "word") {
				params["count"] = num
				break
			}
		}
	}

	return params
}

// parseNumber converts a word to a number, handling both digits and words.
func parseNumber(word string) int {
	// Try direct parsing
	var num int
	if _, err := fmt.Sscanf(word, "%d", &num); err == nil {
		return num
	}

	// Try word-to-number conversion
	wordNums := map[string]int{
		"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4,
		"five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9,
		"ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13,
		"fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17,
		"eighteen": 18, "nineteen": 19, "twenty": 20,
	}

	if num, ok := wordNums[word]; ok {
		return num
	}

	return 0
}

// extractTarget extracts a target from text (legacy function for compatibility).
func extractTarget(text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		if word == "to" || word == "open" {
			if i+1 < len(words) {
				return strings.Join(words[i+1:], " ")
			}
		}
	}
	return ""
}

// extractQuery extracts a query from text (legacy function for compatibility).
func extractQuery(text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		if word == "search" || word == "find" || word == "for" {
			if i+1 < len(words) {
				return strings.Join(words[i+1:], " ")
			}
		}
	}
	return ""
}

// =============================================================================
// Audio Encoding
// =============================================================================

// EncodeWAV encodes raw audio data to WAV format.
func EncodeWAV(data []byte, sampleRate, channels int) []byte {
	// WAV header (44 bytes)
	header := make([]byte, 44)

	// RIFF header
	copy(header[0:4], []byte("RIFF"))
	// File size (placeholder, will be filled later)
	// copy(header[4:8], ...)
	copy(header[8:12], []byte("WAVE"))

	// fmt chunk
	copy(header[12:16], []byte("fmt "))
	// Chunk size (16 for PCM)
	header[16] = 16
	// Audio format (1 for PCM)
	header[20] = 1
	// Number of channels
	header[22] = byte(channels)
	// Sample rate
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	// Byte rate
	byteRate := sampleRate * channels * 2
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	// Block align
	blockAlign := channels * 2
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)
	// Bits per sample
	header[34] = 16
	header[35] = 0

	// data chunk
	copy(header[36:40], []byte("data"))
	// Data size
	dataSize := len(data)
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	// File size
	fileSize := uint32(36 + dataSize)
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	// Combine header and data
	result := make([]byte, 0, 44+len(data))
	result = append(result, header...)
	result = append(result, data...)

	return result
}

// EncodeBase64 encodes audio data to base64.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
