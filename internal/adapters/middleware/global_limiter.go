package middleware

import (
	"net/http"
	"sync"
	"time"
)

func GlobalLimiter(max int) func(http.Handler) http.Handler {
	limiter := newGlobalLimiter(max)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
				return // ← дальше не идём вообще
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Глобальный счётчик запросов
type globalLimiter struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
	max     int // Максимум запросов на весь сервер в секунду
}

func (l *globalLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now() // ← один раз
	if now.After(l.resetAt) {
		l.count = 0
		l.resetAt = now.Add(time.Second)
	}
	if l.count >= l.max {
		return false
	}
	l.count++
	return true
}

func newGlobalLimiter(max int) *globalLimiter {
	return &globalLimiter{
		max:     max,
		resetAt: time.Now().Add(time.Second),
	}
}
