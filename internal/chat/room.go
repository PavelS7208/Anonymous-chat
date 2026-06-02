package chat

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Room struct {
	name     string
	mu       sync.RWMutex
	members  map[int64]*Member
	byPubKey map[string]*Member
	nextID   atomic.Int64
	history  []Event
}

func NewRoom(name string) *Room {
	log.Printf("room with name=\"%s\" created", name)
	return &Room{
		name:     name,
		members:  make(map[int64]*Member),
		byPubKey: make(map[string]*Member),
		history:  make([]Event, 0, InitialHistoryCount),
	}
}

// Register добавляет участника в комнату
func (r *Room) Register(publicKey ed25519.PublicKey) (*Member, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nextID.Load() == MaxMemberID {
		return nil, fmt.Errorf("memberID overflow")
	}
	id := r.nextID.Add(1)

	member := NewMember(id, publicKey)
	r.members[id] = member
	r.byPubKey[base64.StdEncoding.EncodeToString(publicKey)] = member

	log.Printf("member with memberID=%d registered for room=\"%s\"", member.ID, r.name)
	return member, nil
}

// Unregister удаляет участника, рассылает событие left и закрывает каналы
func (r *Room) Unregister(m *Member) {
	leftEvent := NewLeftEvent(time.Now().Unix(), m.ID)
	r.mu.Lock()

	r.history = append(r.history, leftEvent)
	if len(r.history) > MaxHistoryStorage {
		r.history = r.history[len(r.history)-MaxHistoryStorage:]
	}

	for id, mem := range r.members {
		if id == m.ID {
			continue
		}
		select {
		case mem.Events <- leftEvent:
		default:
			mem.Close()
		}
	}

	delete(r.members, m.ID)
	delete(r.byPubKey, base64.StdEncoding.EncodeToString(m.PublicKey))
	r.mu.Unlock()

	m.Close()
	log.Printf("member with memberID=%d unregistered in room=\"%s\"", m.ID, r.name)
}

// Broadcast рассылает событие всем, кроме excludeID
func (r *Room) Broadcast(e Event, excludeID int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, m := range r.members {
		if id == excludeID {
			continue
		}
		select {
		case m.Events <- e:
			log.Printf("Sent event (%s) to member id =%d", e.String(), id)
		default:
			m.Close()
		}
	}
}

// ------  Потокобезопасные утилиты работы с комнатой --------------------

// GetMemberByPubKey ищет участника по base64 публичному ключу
func (r *Room) GetMemberByPubKey(b64 string) (*Member, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byPubKey[b64]
	return p, ok
}

// AddToHistory добавляет событие в историю с защитой от OOM
func (r *Room) AddToHistory(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = append(r.history, e)
	if len(r.history) > MaxHistoryStorage {
		r.history = r.history[len(r.history)-MaxHistoryStorage:]
	}
}

// GetHistorySnapshot возвращает копию последних InitialHistoryCount событий
func (r *Room) GetHistorySnapshot() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := len(r.history)
	if total <= InitialHistoryCount {
		snapshot := make([]Event, total)
		copy(snapshot, r.history)
		return snapshot
	}
	start := total - InitialHistoryCount
	snapshot := make([]Event, InitialHistoryCount)
	copy(snapshot, r.history[start:])
	return snapshot
}
