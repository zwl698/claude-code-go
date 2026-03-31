package utils

import "sync"

// Signal is a tiny listener-set primitive for pure event signals (no stored state).
// Use this when subscribers only need to know "something happened",
// optionally with event args, not "what is the current value".
type Signal[T any] struct {
	listeners []func(T)
	mu        sync.RWMutex
}

// NewSignal creates a new signal.
func NewSignal[T any]() *Signal[T] {
	return &Signal[T]{
		listeners: make([]func(T), 0),
	}
}

// Subscribe adds a listener and returns an unsubscribe function.
func (s *Signal[T]) Subscribe(listener func(T)) func() {
	s.mu.Lock()
	s.listeners = append(s.listeners, listener)
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, l := range s.listeners {
			// Compare function pointers
			if &l == &listener {
				s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
				return
			}
		}
	}
}

// Emit calls all subscribed listeners with the given argument.
func (s *Signal[T]) Emit(arg T) {
	s.mu.RLock()
	listeners := make([]func(T), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.RUnlock()

	for _, listener := range listeners {
		listener(arg)
	}
}

// Clear removes all listeners.
func (s *Signal[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = make([]func(T), 0)
}

// SignalVoid is a signal without arguments.
type SignalVoid struct {
	listeners []func()
	mu        sync.RWMutex
}

// NewSignalVoid creates a new void signal.
func NewSignalVoid() *SignalVoid {
	return &SignalVoid{
		listeners: make([]func(), 0),
	}
}

// Subscribe adds a listener and returns an unsubscribe function.
func (s *SignalVoid) Subscribe(listener func()) func() {
	s.mu.Lock()
	s.listeners = append(s.listeners, listener)
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, l := range s.listeners {
			if &l == &listener {
				s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
				return
			}
		}
	}
}

// Emit calls all subscribed listeners.
func (s *SignalVoid) Emit() {
	s.mu.RLock()
	listeners := make([]func(), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.RUnlock()

	for _, listener := range listeners {
		listener()
	}
}

// Clear removes all listeners.
func (s *SignalVoid) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = make([]func(), 0)
}
