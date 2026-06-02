package chat

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
)

// Session хранит всё, что нужно для одного подключения (стриминга)
// комнату, участника, приватный ключ и логику безопасного отключения.
type Session struct {
	room             *Room
	member           *Member
	memberPrivateKey ed25519.PrivateKey
	once             sync.Once
}

func NewSession(room *Room) (*Session, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	member, err := room.Register(publicKey)
	if err != nil {
		return nil, err
	}

	return &Session{
		room:             room,
		member:           member,
		memberPrivateKey: privateKey,
	}, nil
}

// Close гарантирует однократное завершение сессии.
// Потокобезопасен, может вызываться многократно (например, panic + defer).
func (s *Session) Close() {
	s.once.Do(func() {
		s.member.Close()            // 1. Закрываем Done → разблокируем select в хендлере
		s.room.Unregister(s.member) // 2. Удаляем из комнаты → рассылаем "left"
	})
}

func (s *Session) GetMemberID() int64 {
	return s.member.ID
}

func (s *Session) GetHistorySnapshot() []Event {
	return s.room.GetHistorySnapshot()
}

func (s *Session) AddToHistory(event Event) {
	s.room.AddToHistory(event)
}

// Broadcast рассылка Event-а всем кроме себя
func (s *Session) Broadcast(event Event) {
	s.room.Broadcast(event, s.member.ID)
}

func (s *Session) GetEvents() chan Event {
	return s.member.Events
}

func (s *Session) Done() chan struct{} {
	return s.member.Done
}

func (s *Session) GetWelcomeString() string {
	const seedSize = 32
	seedB64 := base64.StdEncoding.EncodeToString(s.memberPrivateKey[:seedSize])

	return fmt.Sprintf("%d %s\n", s.GetMemberID(), seedB64)
}
