package scheduler

import "sync"

// ConnRegistry tracks active SSE connections per thread_id.
// When a reminder fires, the scheduler looks up the thread's channel and pushes.
type ConnRegistry struct {
	mu sync.RWMutex
	ch map[string]chan ReminderEvent // threadID → notification channel
}

var DefaultRegistry = &ConnRegistry{ch: make(map[string]chan ReminderEvent)}

// Register adds a notification channel for a thread.
// The handler calls this when an SSE stream opens.
func (r *ConnRegistry) Register(threadID string) chan ReminderEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan ReminderEvent, 16)
	r.ch[threadID] = ch
	return ch
}

// Unregister removes a thread's channel (called on SSE stream close).
func (r *ConnRegistry) Unregister(threadID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch, ok := r.ch[threadID]; ok {
		close(ch)
		delete(r.ch, threadID)
	}
}

// Push sends a message to a thread's active SSE connection.
// Returns false if the thread is not connected.
func (r *ConnRegistry) Push(threadID string, event ReminderEvent) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.ch[threadID]
	if !ok {
		return false
	}
	select {
	case ch <- event:
		return true
	default:
		return false
	}
}
