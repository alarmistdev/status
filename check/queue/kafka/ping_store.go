package kafka

import (
	"context"
	"sync"
	"time"
)

// PingStore persists timestamps of processed ping messages.
type PingStore interface {
	// SetProcessed stores the timestamp of a processed ping message.
	SetProcessed(ctx context.Context, ts time.Time) error
	// LastProcessed returns the timestamp of the most recently processed ping message.
	LastProcessed(ctx context.Context) (time.Time, error)
}

// InMemoryPingStore keeps ping timestamps in memory with concurrency safety.
type InMemoryPingStore struct {
	mu   sync.RWMutex
	last time.Time
}

// NewInMemoryPingStore constructs a new in-memory ping store.
func NewInMemoryPingStore() *InMemoryPingStore {
	return &InMemoryPingStore{}
}

// SetProcessed stores the provided timestamp as the latest processed ping.
func (s *InMemoryPingStore) SetProcessed(_ context.Context, ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.last = ts

	return nil
}

// LastProcessed returns the timestamp of the latest processed ping.
func (s *InMemoryPingStore) LastProcessed(_ context.Context) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.last, nil
}
