// Package streaming implements the audio broadcast hub.
// Each stream has one broadcaster goroutine that reads audio chunks and
// fans them out to all registered listener channels with zero blocking.
package streaming

import (
	"context"
	"log/slog"
	"sync"
)

// Chunk is an immutable slice of audio bytes from the broadcaster.
type Chunk []byte

// Hub multiplexes one audio source to N concurrent listeners.
type Hub struct {
	mu          sync.RWMutex
	listeners   map[string]chan Chunk
	bufferSize  int
	logger      *slog.Logger
}

func NewHub(bufferSize int, logger *slog.Logger) *Hub {
	return &Hub{
		listeners:  make(map[string]chan Chunk),
		bufferSize: bufferSize,
		logger:     logger,
	}
}

// Subscribe registers a listener and returns its read channel.
// The caller must call Unsubscribe when done to prevent goroutine leaks.
func (h *Hub) Subscribe(listenerID string) <-chan Chunk {
	ch := make(chan Chunk, h.bufferSize)
	h.mu.Lock()
	h.listeners[listenerID] = ch
	h.mu.Unlock()
	h.logger.Info("listener subscribed", slog.String("listener_id", listenerID),
		slog.Int("total", h.count()))
	return ch
}

// Unsubscribe removes the listener and closes its channel.
func (h *Hub) Unsubscribe(listenerID string) {
	h.mu.Lock()
	if ch, ok := h.listeners[listenerID]; ok {
		close(ch)
		delete(h.listeners, listenerID)
	}
	h.mu.Unlock()
	h.logger.Info("listener unsubscribed", slog.String("listener_id", listenerID),
		slog.Int("total", h.count()))
}

// Broadcast sends a chunk to all active listeners.
// Slow listeners whose buffer is full are dropped (non-blocking send).
func (h *Hub) Broadcast(ctx context.Context, chunk Chunk) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, ch := range h.listeners {
		select {
		case ch <- chunk:
		case <-ctx.Done():
			return
		default:
			h.logger.Warn("listener buffer full, dropping chunk",
				slog.String("listener_id", id))
		}
	}
}

// ListenerCount returns the number of currently active listeners.
func (h *Hub) ListenerCount() int {
	return h.count()
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.listeners)
}
