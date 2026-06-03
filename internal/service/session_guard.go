package service

import (
	"context"
	"sync"
	"time"
)

type sessionEntry struct {
	ip           string
	createdAt    time.Time
	lastActiveAt time.Time // Обновляется при каждом POST
}

type sessionGuard struct {
	mu          sync.Mutex
	sessions    map[string]sessionEntry
	absoluteTTL time.Duration // Макс время жизни сессии
	idleTTL     time.Duration // Макс время неактивности (нет сообщений)
}

func newSessionGuard(absoluteTTL, idleTTL time.Duration) *sessionGuard {
	return &sessionGuard{
		sessions:    make(map[string]sessionEntry),
		absoluteTTL: absoluteTTL,
		idleTTL:     idleTTL,
	}
}

// Register — вызывается при GET (JoinAndStream) после генерации ключей
func (g *sessionGuard) Register(pubKeyB64, ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	g.sessions[pubKeyB64] = sessionEntry{
		ip:           ip,
		createdAt:    now,
		lastActiveAt: now,
	}
}

// Verify — вызывается при POST (Send)
// Проверяет что pubKey пришёл с того же IP что и при GET
func (g *sessionGuard) Verify(pubKeyB64, ip string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry, ok := g.sessions[pubKeyB64]
	if !ok {
		return ErrSessionNotFound
	}

	now := time.Now()

	// Проверка абсолютного TTL
	if now.Sub(entry.createdAt) > g.absoluteTTL {
		delete(g.sessions, pubKeyB64)
		return ErrSessionExpired
	}

	// Проверка TTL неактивности
	if now.Sub(entry.lastActiveAt) > g.idleTTL {
		delete(g.sessions, pubKeyB64)
		return ErrSessionIdle
	}

	// IP проверка
	if entry.ip != ip {
		return ErrIPMismatch
	}

	// Обновляем время активности
	entry.lastActiveAt = now
	g.sessions[pubKeyB64] = entry
	return nil
}

// Unregister — вызывается при отключении клиента (leave)
func (g *sessionGuard) Unregister(pubKeyB64 string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.sessions, pubKeyB64)
}

// cleanup — периодическая очистка протухших сессий

func (g *sessionGuard) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(g.idleTTL / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.cleanup()
			}
		}
	}()
	return nil
}

func (g *sessionGuard) cleanup() {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for pub, entry := range g.sessions {
		if now.Sub(entry.createdAt) > g.absoluteTTL {
			delete(g.sessions, pub)
			continue
		}
		if now.Sub(entry.lastActiveAt) > g.idleTTL {
			delete(g.sessions, pub)
		}
	}
}
