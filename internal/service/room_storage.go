package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pavel/anonymous-chat/internal/domain"
)

// inMemoryRoomStorage управляет жизненным циклом комнат
type roomEntry struct {
	room       *domain.Room
	lastActive time.Time
}

// Храним и управляем комнатами в памяти
type inMemoryRoomStorage struct {
	mu    sync.RWMutex
	rooms map[string]*roomEntry

	cfg         RoomManagerConfig
	roomFactory domain.RoomFactory

	// Управление жизненным циклом
	closing   atomic.Bool // Флаг: идёт ли shutdown
	closeOnce sync.Once   // Гарантия однократного выполнения Close()
}

// NewRoomManager создает новый менеджер.
func newInMemoryRoomStorage(cfg RoomManagerConfig, rf domain.RoomFactory) *inMemoryRoomStorage {
	cfg = cfg.withDefaults()
	return &inMemoryRoomStorage{
		rooms:       make(map[string]*roomEntry),
		cfg:         cfg,
		roomFactory: rf,
	}
}

// Start запускает фоновые процессы менеджера.
// Вызывать один раз после создания, перед обработкой запросов.
// Принимает внешний контекст для корректного graceful shutdown
// Останавливает все что запущено вызывающая сторона путем ctx.Cancel()
func (rm *inMemoryRoomStorage) Start(ctx context.Context) error {
	go rm.startSweeper(ctx)
	return nil
}

// Close gracefully останавливает inMemoryRoomStorage.
// - Запрещает создание новых комнат
// - Немедленно закрывает все существующие комнаты
// - НЕ отменяет фоновые горутины — это ответственность вызывающего кода через контекст
//
// Идемпотентен: безопасен для многократного вызова.
func (rm *inMemoryRoomStorage) Close() error {
	rm.closeOnce.Do(func() {
		// Сигнал: больше не принимаем новые комнаты
		rm.closing.Store(true)

		// Собираем комнаты под локом (чтобы не держать его во время Close())
		roomsToClose := make([]*domain.Room, 0, len(rm.rooms))
		rm.mu.Lock()
		for _, re := range rm.rooms {
			roomsToClose = append(roomsToClose, re.room)
		}
		// Очищаем мапу — новые запросы не найдут комнаты
		clear(rm.rooms)
		rm.mu.Unlock()

		// Закрываем комнаты ВНЕ локов (безопасно, т.к. ссылки скопированы)
		// Room.Close() идемпотентен, поэтому повторные вызовы безопасны
		for _, room := range roomsToClose {
			room.Close()
		}
	})
	return nil
}

func (rm *inMemoryRoomStorage) GetOrCreate(_ context.Context, name string) (*domain.Room, error) {

	// Проверка: если идёт shutdown — отказываем
	if rm.closing.Load() {
		return nil, ErrShuttingDown
	}

	rm.mu.RLock()
	e, ok := rm.rooms[name]
	rm.mu.RUnlock()
	if ok {
		e.lastActive = time.Now()
		return e.room, nil
	}

	rm.mu.Lock()
	if e, ok = rm.rooms[name]; ok {
		rm.mu.Unlock()
		e.lastActive = time.Now()
		return e.room, nil
	}

	r, err := rm.roomFactory.NewRoom(name)
	if err != nil {
		return nil, err
	}
	rm.rooms[name] = &roomEntry{room: r, lastActive: time.Now()}
	rm.mu.Unlock()
	return r, nil
}

// Get находит существующую комнату. Возвращает ErrRoomNotFound, если её нет.
// Обновляет lastActive, чтобы sweeper не удалил активную комнату.
func (rm *inMemoryRoomStorage) Get(_ context.Context, name string) (*domain.Room, error) {
	if rm.closing.Load() {
		return nil, ErrShuttingDown
	}

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	e, ok := rm.rooms[name]
	if !ok {
		return nil, ErrRoomNotFound
	}
	e.lastActive = time.Now()
	return e.room, nil
}

func (rm *inMemoryRoomStorage) Count() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// startSweeper запускает фоновую очистку пустых комнат по истечении TTL
func (rm *inMemoryRoomStorage) startSweeper(ctx context.Context) {
	ticker := time.NewTicker(rm.cfg.SweeperRunPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.sweep()
		}
	}
}

func (rm *inMemoryRoomStorage) sweep() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	now := time.Now()
	ttl := rm.cfg.RoomTTL

	for name, re := range rm.rooms {
		room := re.room
		// Double-check: если зашли в этот момент, счетчик > 0
		if room.MemberCount() > 0 {
			re.lastActive = now
			continue
		}
		if now.Sub(re.lastActive) > ttl {
			room.Close()
			delete(rm.rooms, name)
		}
	}
}
