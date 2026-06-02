package service

import (
	"context"
	"sync"
	"sync/atomic"
)

// Защита по лимитам для отправки сообщений массовых и массово присоединения к комнатам с одного клиента (IP)
type roomRateLimitGuard struct {

	// Реализации защит от атак спамов
	joinRateLimiter *rateLimiter
	postRateLimiter *rateLimiter

	cfg RateLimitConfig
	// Управление жизненным циклом
	closing   atomic.Bool // Флаг: идёт ли shutdown
	closeOnce sync.Once   // Гарантия однократного выполнения Close()
}

// NewRoomManager создает новый менеджер.
func newRoomRateGuard(cfg RateLimitConfig) *roomRateLimitGuard {
	cfg = cfg.withDefaults()
	return &roomRateLimitGuard{
		joinRateLimiter: newRateLimiter(),
		postRateLimiter: newRateLimiter(),
		cfg:             cfg,
	}
}

// Start запускает фоновые процессы менеджера.
// Вызывать один раз после создания, перед обработкой запросов.
// Принимает внешний контекст для корректного graceful shutdown
// Останавливает все что запущено вызывающая сторона путем ctx.Cancel()
func (r *roomRateLimitGuard) Start(ctx context.Context) error {
	go r.joinRateLimiter.StartCleanup(ctx, r.cfg.JoinRateWindow)
	go r.postRateLimiter.StartCleanup(ctx, r.cfg.PostRateWindow)
	return nil
}

// Close gracefully останавливает inMemoryRoomStorage.
// - Запрещает создание новых комнат
// - Немедленно закрывает все существующие комнаты
// - НЕ отменяет фоновые горутины — это ответственность вызывающего кода через контекст
//
// Идемпотентен: безопасен для многократного вызова.
func (r *roomRateLimitGuard) Close() error {
	r.closeOnce.Do(func() {
		// Сигнал: больше не принимаем новые комнаты
		r.closing.Store(true)
	})
	return nil
}

// PreJoinCheck выполняет все проверки перед попыткой присоединения к комнате.
// Возвращает ошибку, если присоединение запрещено.
//
// Взывать ПЕРЕД созданием комнаты
func (r *roomRateLimitGuard) AllowJoin(ip string) error {

	// Если идёт shutdown — отклоняем сразу и ничего тяжелого не запускаем
	if r.closing.Load() {
		return ErrShuttingDown
	}
	// Rate Limit Отсекаем аномально активных клиентов до захвата общих ресурсов
	if !r.joinRateLimiter.Allow(ip, r.cfg.JoinRateLimit) {
		return ErrJoinRateLimited
	}
	return nil
}

// PrePostCheck выполняет все проверки перед попыткой присоединения к комнате.
// Возвращает ошибку, если присоединение запрещено.
//
// Взывать ПЕРЕД отправкой сообщения в чат
func (r *roomRateLimitGuard) AllowPost(ip string) error {

	// Если идёт shutdown — отклоняем сразу
	if r.closing.Load() {
		return ErrShuttingDown
	}
	if !r.postRateLimiter.Allow(ip, r.cfg.PostRateLimit) {
		return ErrPostRateLimited
	}
	return nil
}
