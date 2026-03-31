package utils

import (
	"context"
	"sync"
)

// AbortController manages abort signaling for operations.
type AbortController struct {
	mu      sync.RWMutex
	aborted bool
	reason  error
	signal  *SignalVoid
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewAbortController creates a new AbortController.
func NewAbortController() *AbortController {
	ctx, cancel := context.WithCancel(context.Background())
	return &AbortController{
		signal: NewSignalVoid(),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Abort signals all listeners that the operation should be aborted.
func (a *AbortController) Abort(reason error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.aborted {
		return
	}

	a.aborted = true
	a.reason = reason
	a.cancel()
	a.signal.Emit()
}

// IsAborted returns true if the controller has been aborted.
func (a *AbortController) IsAborted() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aborted
}

// Reason returns the abort reason, or nil if not aborted.
func (a *AbortController) Reason() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.reason
}

// Subscribe adds a listener for abort events.
func (a *AbortController) Subscribe(listener func()) func() {
	return a.signal.Subscribe(listener)
}

// Context returns the context associated with this controller.
func (a *AbortController) Context() context.Context {
	return a.ctx
}

// NewChildAbortController creates a child controller that aborts when the parent aborts.
// Aborting the child does NOT affect the parent.
func NewChildAbortController(parent *AbortController) *AbortController {
	child := NewAbortController()

	// Fast path: parent already aborted
	if parent.IsAborted() {
		child.Abort(parent.Reason())
		return child
	}

	// Subscribe to parent abort
	unsubscribe := parent.Subscribe(func() {
		child.Abort(parent.Reason())
	})

	// Cleanup subscription when child is aborted
	child.Subscribe(func() {
		unsubscribe()
	})

	return child
}
