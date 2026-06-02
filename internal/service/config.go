package service

import (
	"time"

	"github.com/pavel/anonymous-chat/internal/domain"
)

type RoomServiceConfig struct {

	// Тайм-аут на отправку всех необходимых действий новому присоединившемуся клиенту
	// Если не удалось, клиент не присоеденился значит - у него проблемы. Дефолт 5 сек.
	HandshakeTimeout time.Duration

	// Лимиты устойчивости сервера
	MaxMembersPerRoom int // Макс. участников в комнате (дефолт 100)
	MaxGlobalRooms    int // Жёсткий лимит комнат в процессе (дефолт 1000)

	MaxConnectionsPerIP int // Cколько SSE соединений с одного IP одновременно. дефолт 5

	mng     RoomManagerConfig
	limiter RateLimitConfig

	SessionAbsoluteTTL time.Duration //Время жизни сессии. Дефолт - 24 часа
	SessionIdleTTL     time.Duration // Время жизни без сообщений. Время зависания. Дефолт 3 часа

}

type RoomManagerConfig struct {
	RoomTTL          time.Duration // Время жизни пустой комнаты. Дефолт 30 минут
	SweeperRunPeriod time.Duration // Период запуска очистителя мертвых комнат. Дефолт 10 минут

	room   domain.RoomConfig
	member domain.MemberConfig
}

type RateLimitConfig struct {
	JoinRateLimit  int           // Присоединений в окно (дефолт 5)
	JoinRateWindow time.Duration // Окно rate-limit (дефолт 1 мин)
	PostRateLimit  int           // Отправок сообщений в окно (дефолт 10)
	PostRateWindow time.Duration // Окно rate-limit (дефолт 1 мин)
}

func (c RoomServiceConfig) withDefaults() RoomServiceConfig {
	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = time.Second * 5
	}
	if c.MaxGlobalRooms == 0 {
		c.MaxGlobalRooms = 1_000
	}
	if c.MaxMembersPerRoom == 0 {
		c.MaxMembersPerRoom = 100
	}
	if c.MaxConnectionsPerIP == 0 {
		c.MaxConnectionsPerIP = 3
	}
	if c.SessionAbsoluteTTL == 0 {
		c.SessionAbsoluteTTL = 24 * time.Hour
	}
	if c.SessionIdleTTL == 0 {
		c.SessionIdleTTL = 3 * time.Hour
	}
	return c
}

func (c RoomManagerConfig) withDefaults() RoomManagerConfig {
	if c.RoomTTL == 0 {
		c.RoomTTL = time.Minute * 30
	}
	if c.SweeperRunPeriod == 0 {
		c.SweeperRunPeriod = 10 * time.Minute
	}
	return c
}

func (c RateLimitConfig) withDefaults() RateLimitConfig {
	if c.JoinRateLimit == 0 {
		c.JoinRateLimit = 5
	}
	if c.JoinRateWindow == 0 {
		c.JoinRateWindow = time.Second * 60
	}
	if c.PostRateLimit == 0 {
		c.PostRateLimit = 10
	}
	if c.PostRateWindow == 0 {
		c.PostRateWindow = time.Second * 60
	}
	return c
}
