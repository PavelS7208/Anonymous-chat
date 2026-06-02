package chat

import (
	"crypto/ed25519"
	"sync"
)

// Member подключённый участник
// Экспортирован, так как используется в слоях выше
type Member struct {
	ID        int64
	PublicKey ed25519.PublicKey
	Events    chan Event
	Done      chan struct{}
	once      sync.Once
}

func NewMember(id int64, publicKey ed25519.PublicKey) *Member {
	return &Member{
		ID:        id,
		PublicKey: publicKey,
		Events:    make(chan Event, EventChannelBuf),
		Done:      make(chan struct{}),
	}
}

// Close закрывает канал Done, сигнализируя хендлеру об отключении.
// Безопасен для конкурентных вызовов.
func (m *Member) Close() {
	m.once.Do(func() { close(m.Done) })
}
