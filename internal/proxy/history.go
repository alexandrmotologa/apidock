package proxy

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// HistoryEntry records a proxied execution for auditing and replay.
type HistoryEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Request   Request   `json:"request"`
	Response  Response  `json:"response"`
}

// HistoryRecorder maintains a thread-safe ring buffer of recent requests.
type HistoryRecorder struct {
	mu       sync.RWMutex
	capacity int
	entries  []HistoryEntry
}

// NewHistoryRecorder creates a recorder with the given capacity limit.
func NewHistoryRecorder(capacity int) *HistoryRecorder {
	if capacity <= 0 {
		capacity = 50
	}
	return &HistoryRecorder{
		capacity: capacity,
		entries:  make([]HistoryEntry, 0, capacity),
	}
}

// Record saves a new execution entry to history.
func (h *HistoryRecorder) Record(req Request, resp Response) HistoryEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry := HistoryEntry{
		ID:        generateID(),
		Timestamp: time.Now().UTC(),
		Request:   req,
		Response:  resp,
	}

	// Prepend for newest-first order
	h.entries = append([]HistoryEntry{entry}, h.entries...)
	if len(h.entries) > h.capacity {
		h.entries = h.entries[:h.capacity]
	}

	return entry
}

// GetAll returns a copy of all recorded history entries.
func (h *HistoryRecorder) GetAll() []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]HistoryEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

// Clear wipes all recorded history.
func (h *HistoryRecorder) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make([]HistoryEntry, 0, h.capacity)
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("hist_%x", b)
}
