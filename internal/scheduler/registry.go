package scheduler

import "sync"

// ConnRegistry tracks active SSE connections per thread_id.
// When a reminder fires, the scheduler looks up the thread's channel and pushes.
type ConnRegistry struct {
	mu sync.RWMutex
	ch map[string]map[chan ReminderEvent]struct{} // threadID -> notification channels
}

var DefaultRegistry = NewConnRegistry()

func NewConnRegistry() *ConnRegistry {
	return &ConnRegistry{ch: make(map[string]map[chan ReminderEvent]struct{})}
}

// Register adds a notification channel for a thread.
// The handler calls this when an SSE stream opens.
func (r *ConnRegistry) Register(threadID string) chan ReminderEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan ReminderEvent, 16)
	if r.ch[threadID] == nil {
		r.ch[threadID] = make(map[chan ReminderEvent]struct{})
	}
	r.ch[threadID][ch] = struct{}{}
	return ch
}

// Unregister removes a thread's channel (called on SSE stream close).
func (r *ConnRegistry) Unregister(threadID string, ch chan ReminderEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	channels, ok := r.ch[threadID]
	if !ok {
		return
	}
	if _, ok := channels[ch]; ok {
		close(ch)
		delete(channels, ch)
	}
	if len(channels) == 0 {
		delete(r.ch, threadID)
	}
}

// Push sends a message to a thread's active SSE connection.
// Returns false if the thread is not connected.
func (r *ConnRegistry) Push(threadID string, event ReminderEvent) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	channels, ok := r.ch[threadID]
	if !ok {
		return false
	}
	delivered := false
	for ch := range channels {
		select {
		case ch <- event:
			delivered = true
		default:
		}
	}
	return delivered
}
