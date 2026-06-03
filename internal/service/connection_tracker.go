package service

import (
	"sync"
)

type connectionTracker struct {
	mu          sync.Mutex
	connections map[string]int // IP → кол-во активных соединений
	max         int
}

func newConnectionTracker(max int) *connectionTracker {
	return &connectionTracker{
		connections: make(map[string]int),
		max:         max,
	}
}

// Acquire — занять слот. Вернёт ошибку если лимит исчерпан.
func (t *connectionTracker) Acquire(ip string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.connections[ip] >= t.max {
		return ErrTooManyConnections
	}
	t.connections[ip]++
	return nil
}

// Release — освободить слот. Вызывать через defer — всегда!
func (t *connectionTracker) Release(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connections[ip]--
	if t.connections[ip] <= 0 {
		delete(t.connections, ip) // не копим пустые записи
	}
}
