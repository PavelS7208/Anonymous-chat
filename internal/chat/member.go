package chat

import (
	"sync"
)

type MemberID int64

// Member подключённый участник
type Member struct {
	id           MemberID
	publicB64Key string // base64 закодированный pubKey
	events       chan Event
	done         chan struct{}
	closeOnce    sync.Once
}

func NewMember(id MemberID, pubB64Key string) *Member {
	return &Member{
		id:           id,
		publicB64Key: pubB64Key,
		events:       make(chan Event, cfg.eventChannelBuf),
		done:         make(chan struct{}),
	}
}

func (m *Member) ID() MemberID         { return m.id }
func (m *Member) PublicB64Key() string { return m.publicB64Key }
func (m *Member) Events() chan Event   { return m.events }

// Close закрывает канал done, сигнализируя об отключении
// Безопасен для конкурентных вызовов
func (m *Member) Close() {
	m.closeOnce.Do(func() {
		close(m.done)
		close(m.events)
	})
}
func (m *Member) Done() <-chan struct{} { return m.done }
