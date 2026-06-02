package service

import (
	"context"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{records: make(map[string][]time.Time)}
}

func (l *rateLimiter) Allow(clientID string, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts, ok := l.records[clientID]
	if !ok {
		l.records[clientID] = []time.Time{time.Now()}
		return true
	}
	if len(ts) >= limit {
		return false
	}
	l.records[clientID] = append(ts, time.Now())
	return true
}

func (l *rateLimiter) StartCleanup(ctx context.Context, window time.Duration) {
	ticker := time.NewTicker(window / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-window)
			l.mu.Lock()
			for id, ts := range l.records {
				idx := 0
				for idx < len(ts) && !ts[idx].After(cutoff) {
					idx++
				}
				if idx == len(ts) {
					delete(l.records, id)
				} else if idx > 0 {
					copy(ts, ts[idx:])
					l.records[id] = ts[:len(ts)-idx]
				}
			}
			l.mu.Unlock()
		}
	}
}
