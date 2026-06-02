package service

import (
	"context"
	"log/slog"

	"github.com/pavel/anonymous-chat/internal/domain"
	"github.com/pavel/anonymous-chat/internal/domain/crypto"
)

// Контракт хранилище комнат/ Текущая реализация InMemory
type roomStorage interface {
	Get(ctx context.Context, roomName string) (*domain.Room, error)
	GetOrCreate(ctx context.Context, roomName string) (*domain.Room, error)
	Count() int
}

// RoomManager — оркестратор сессии участника
type RoomManager interface {
	Join(ctx context.Context, req JoinRequest) (*JoinSession, error)
	Stream(ctx context.Context, session *JoinSession, w ChatWriter) error
}

// MessageSender — обработчик входящих сообщений
type MessageSender interface {
	Send(ctx context.Context, req SendRequest) error
}

// WireMessage — контракт для доменных типов, поддерживающих сериализацию в проводной протокол.
// Экспортируется для использования в адаптерах.
type WireMessage interface {
	Marshal() []byte
}

// ChatWriter — контракт для записи данных в транспорт (авто-flush внутри реализации)
type ChatWriter interface {
	Write(ctx context.Context, msg WireMessage) error
}

// rateLimitGuard - Защитник от атак по массовой отправке однотипных команд
type rateLimitGuard interface {
	AllowJoin(ip string) error
	AllowPost(ip string) error
}

// JoinSession — состояние успешного Join, передаётся в Stream.
// Владеет всеми захваченными ресурсами через release.
// release вызывается ровно один раз: либо через Stream (defer),
// либо через Release() если Stream не был вызван.
type JoinSession struct {
	room        *domain.Room
	member      *domain.Member
	snapshot    []domain.Event
	lastSeq     uint64
	privateSeed []byte
	pubKeyB64   string
	release     func()
}

func (s *JoinSession) Release() {
	s.release()
}

type ChatService struct {
	repo roomStorage

	limitGuard   rateLimitGuard
	connTracker  *connectionTracker
	sessionGuard *sessionGuard

	cfg RoomServiceConfig

	crypto crypto.Provider
	logger *slog.Logger
}

// NewChatService создает сервис и возвращает:
// ChatService — интерфейс для бизнес-логики (передаётся в хендлеры)
// ChatServiceLifecycle — интерфейс для управления жизненным циклом (используется в main)
//
// inMemoryRoomStorage создаётся внутри и полностью скрыт от внешнего кода.
func NewChatService(cfg RoomServiceConfig, cp crypto.Provider, logger *slog.Logger) (*ChatService, *ChatServiceLifecycle) {
	cfg = cfg.withDefaults()

	mf := domain.NewMemberFactory(cfg.mng.member)
	rf := domain.NewRoomFactory(cfg.mng.room, mf)
	repo := newInMemoryRoomStorage(cfg.mng, rf)

	rGuard := newRoomRateGuard(cfg.limiter)
	connTr := newConnectionTracker(cfg.MaxConnectionsPerIP)
	sGuard := newSessionGuard(cfg.SessionAbsoluteTTL, cfg.SessionIdleTTL)

	svc := &ChatService{
		repo: repo,
		//	memberFactory: mf,
		limitGuard:   rGuard,
		connTracker:  connTr,
		sessionGuard: sGuard,
		cfg:          cfg,
		crypto:       cp,
		logger:       logger,
	}

	lc := &ChatServiceLifecycle{}
	lc.Register(repo)
	lc.Register(rGuard)
	lc.Register(sGuard)

	return svc, lc
}
