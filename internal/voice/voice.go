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

// VoiceCommand represents a voice command.
type VoiceCommand struct {
	Type   string                 `json:"type"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// ParseCommand parses a voice command from transcribed text.
func ParseCommand(text string) *VoiceCommand {
	// Simple command parsing
	// TODO: Implement more sophisticated NLP-based parsing

	text = strings.ToLower(text)

	// Navigation commands
	if strings.Contains(text, "go to") || strings.Contains(text, "open") {
		return &VoiceCommand{
			Type:   "navigation",
			Action: "goto",
			Params: map[string]interface{}{
				"target": extractTarget(text),
			},
		}
	}

	// Edit commands
	if strings.Contains(text, "delete") || strings.Contains(text, "remove") {
		return &VoiceCommand{
			Type:   "edit",
			Action: "delete",
		}
	}

	if strings.Contains(text, "copy") {
		return &VoiceCommand{
			Type:   "edit",
			Action: "copy",
		}
	}

	if strings.Contains(text, "paste") {
		return &VoiceCommand{
			Type:   "edit",
			Action: "paste",
		}
	}

	if strings.Contains(text, "undo") {
		return &VoiceCommand{
			Type:   "edit",
			Action: "undo",
		}
	}

	if strings.Contains(text, "redo") {
		return &VoiceCommand{
			Type:   "edit",
			Action: "redo",
		}
	}

	// Search commands
	if strings.Contains(text, "search") || strings.Contains(text, "find") {
		return &VoiceCommand{
			Type:   "search",
			Action: "find",
			Params: map[string]interface{}{
				"query": extractQuery(text),
			},
		}
	}

	// Default: treat as text input
	return &VoiceCommand{
		Type:   "input",
		Action: "text",
		Params: map[string]interface{}{
			"text": text,
		},
	}
}

func extractTarget(text string) string {
	// Simple extraction: take words after "go to" or "open"
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

func extractQuery(text string) string {
	// Simple extraction: take words after "search" or "find"
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
