package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// AnalyticsService handles event logging and tracking
type AnalyticsService struct {
	mu          sync.RWMutex
	sink        AnalyticsSink
	eventQueue  []*QueuedEvent
	enabled     bool
	sampleRate  float64
	userType    string
	sessionID   string
	environment string
}

// AnalyticsSink is the interface for analytics backends
type AnalyticsSink interface {
	LogEvent(eventName string, metadata EventMetadata)
	LogEventAsync(ctx context.Context, eventName string, metadata EventMetadata) error
}

// EventMetadata represents metadata for analytics events
type EventMetadata map[string]interface{}

// QueuedEvent represents an event in the queue before sink is attached
type QueuedEvent struct {
	EventName string
	Metadata  EventMetadata
	Async     bool
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{
		eventQueue:  make([]*QueuedEvent, 0),
		enabled:     true,
		sampleRate:  1.0,
		environment: "production",
	}
}

// AttachSink attaches an analytics sink
func (s *AnalyticsService) AttachSink(sink AnalyticsSink) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sink != nil {
		return
	}
	s.sink = sink

	// Drain queued events
	if len(s.eventQueue) > 0 {
		queuedEvents := s.eventQueue
		s.eventQueue = make([]*QueuedEvent, 0)

		go func() {
			for _, event := range queuedEvents {
				if event.Async {
					_ = sink.LogEventAsync(context.Background(), event.EventName, event.Metadata)
				} else {
					sink.LogEvent(event.EventName, event.Metadata)
				}
			}
		}()
	}
}

// LogEvent logs an analytics event (synchronous)
func (s *AnalyticsService) LogEvent(eventName string, metadata EventMetadata) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.enabled {
		return
	}

	if metadata == nil {
		metadata = make(EventMetadata)
	}
	metadata["session_id"] = s.sessionID
	metadata["timestamp"] = time.Now().Unix()

	if s.sink == nil {
		s.eventQueue = append(s.eventQueue, &QueuedEvent{
			EventName: eventName,
			Metadata:  metadata,
			Async:     false,
		})
		return
	}

	s.sink.LogEvent(eventName, metadata)
}

// LogEventAsync logs an analytics event (asynchronous)
func (s *AnalyticsService) LogEventAsync(ctx context.Context, eventName string, metadata EventMetadata) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.enabled {
		return nil
	}

	if metadata == nil {
		metadata = make(EventMetadata)
	}
	metadata["session_id"] = s.sessionID
	metadata["timestamp"] = time.Now().Unix()

	if s.sink == nil {
		s.eventQueue = append(s.eventQueue, &QueuedEvent{
			EventName: eventName,
			Metadata:  metadata,
			Async:     true,
		})
		return nil
	}

	return s.sink.LogEventAsync(ctx, eventName, metadata)
}

// SetEnabled enables or disables analytics
func (s *AnalyticsService) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
}

// SetSessionID sets the session ID
func (s *AnalyticsService) SetSessionID(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = sessionID
}

// ConsoleAnalyticsSink logs events to console for development
type ConsoleAnalyticsSink struct {
	prefix string
}

// NewConsoleAnalyticsSink creates a new console analytics sink
func NewConsoleAnalyticsSink(prefix string) *ConsoleAnalyticsSink {
	return &ConsoleAnalyticsSink{prefix: prefix}
}

// LogEvent logs an event to console
func (s *ConsoleAnalyticsSink) LogEvent(eventName string, metadata EventMetadata) {
	data, _ := json.Marshal(metadata)
	println(s.prefix, eventName, string(data))
}

// LogEventAsync logs an event to console asynchronously
func (s *ConsoleAnalyticsSink) LogEventAsync(ctx context.Context, eventName string, metadata EventMetadata) error {
	s.LogEvent(eventName, metadata)
	return nil
}

// MultiAnalyticsSink sends events to multiple sinks
type MultiAnalyticsSink struct {
	sinks []AnalyticsSink
}

// NewMultiAnalyticsSink creates a new multi sink
func NewMultiAnalyticsSink(sinks ...AnalyticsSink) *MultiAnalyticsSink {
	return &MultiAnalyticsSink{sinks: sinks}
}

// LogEvent logs an event to all sinks
func (s *MultiAnalyticsSink) LogEvent(eventName string, metadata EventMetadata) {
	for _, sink := range s.sinks {
		sink.LogEvent(eventName, metadata)
	}
}

// LogEventAsync logs an event to all sinks asynchronously
func (s *MultiAnalyticsSink) LogEventAsync(ctx context.Context, eventName string, metadata EventMetadata) error {
	for _, sink := range s.sinks {
		_ = sink.LogEventAsync(ctx, eventName, metadata)
	}
	return nil
}

// StripProtoFields strips _PROTO_* keys from metadata
func StripProtoFields(metadata EventMetadata) EventMetadata {
	result := make(EventMetadata)
	for k, v := range metadata {
		if len(k) < 7 || k[:7] != "_PROTO_" {
			result[k] = v
		}
	}
	return result
}

// Global instance
var (
	globalAnalytics *AnalyticsService
	once            sync.Once
)

// GetAnalytics returns the global analytics instance
func GetAnalytics() *AnalyticsService {
	once.Do(func() {
		globalAnalytics = NewAnalyticsService()
	})
	return globalAnalytics
}
