package chat

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Service управляет жизненным циклом комнат
type Service struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewService() *Service {
	return &Service{rooms: make(map[string]*Room)}
}

// NewSession - реализация логики подготовки к стримингу в комнате
// Под капотом создаёт комнату (если нет), генерирует ключи, регистрирует участника
// Вызывающий обязан вызвать session.Close().
func (s *Service) NewSession(roomName string) (*Session, error) {
	return NewSession(s.getOrCreateRoom(roomName))
}

// getOrCreateRoom создаёт комнату при первом обращении если ее нет (double-checked locking)
func (s *Service) getOrCreateRoom(name string) *Room {
	s.mu.RLock()
	r, ok := s.rooms[name]
	s.mu.RUnlock()
	if ok {
		return r
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if room, ok := s.rooms[name]; ok {
		return room
	}
	room := NewRoom(name)
	s.rooms[name] = room
	return room
}

// PostMessage - реализация отправки сообщения
func (s *Service) PostMessage(roomName, pubB64, sigB64, msg string) error {
	// Валидация формата сообщения
	if err := ValidateMessage(msg); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Получаем комнату куда будем слать сообщение
	room, ok := s.GetRoom(roomName)
	if !ok {
		return ErrRoomNotFound
	}

	// Находим участника по публичному ключу
	member, ok := room.GetMemberByPubKey(pubB64)
	if !ok {
		return ErrUnauthorized
	}

	// Декодируем ключи и проверяем подпись (криптография)
	pubBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return ErrInvalidPubKey
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}

	if !ed25519.Verify(pubBytes, []byte(msg), sigBytes) {
		return ErrForbidden
	}

	// Создаём событие и рассылаем
	event := NewMsgEvent(time.Now().Unix(), member.ID, msg)
	room.AddToHistory(event)
	room.Broadcast(event, 0) // 0 = рассылка всем, включая отправителя (по ТЗ)

	return nil
}

// GetRoom ПОтокобезопасно возвращает комнату только если она уже существует
func (s *Service) GetRoom(name string) (*Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[name]
	return room, ok
}
