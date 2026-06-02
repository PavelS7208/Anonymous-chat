package chat

import (
	"log"
	"sync"
	"sync/atomic"
)

type Room struct {
	name           string
	mu             sync.RWMutex
	members        map[MemberID]*Member
	memberByPubKey map[string]*Member // В base64 формате строка
	nextMemberID   atomic.Uint64
	history        []Event
}

func NewRoom(name string) *Room {
	log.Printf("room with name=\"%s\" created", name)
	return &Room{
		name:           name,
		members:        make(map[MemberID]*Member),
		memberByPubKey: make(map[string]*Member),
		history:        make([]Event, 0, cfg.initialHistoryCap),
	}
}

func (r *Room) Register(m *Member) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверка на дубликат по lookup-ключу и ID
	isDuplicate := func() bool {
		_, byKey := r.memberByPubKey[m.PublicB64Key()]
		_, byID := r.members[m.ID()]
		return byKey || byID
	}

	if isDuplicate() {
		return errMemberAlreadyRegistered
	}

	// Регистрация в двух индексах
	r.members[m.ID()] = m
	r.memberByPubKey[m.PublicB64Key()] = m

	log.Printf("m with memberID=%d registered for room=\"%s\"", m.ID(), r.name)
	return nil
}

// Unregister удаляет участника, рассылает событие left и закрывает каналы
func (r *Room) Unregister(m *Member) {
	r.mu.Lock()

	// Нет комнате такого участника - но раз дали, закрываем и его
	if _, exist := r.members[m.ID()]; !exist {
		r.mu.Unlock()
		m.Close()
		log.Printf("Unregister called for member with ID=%d not in room (%s)", m.ID(), r.name)
		return
	}

	delete(r.members, m.ID())
	delete(r.memberByPubKey, m.PublicB64Key())
	r.mu.Unlock()

	m.Close()
	log.Printf("m with memberID=%d unregistered in room=\"%s\"", m.ID(), r.name)
}

// Broadcast рассылает событие всем, кроме excludeID (используем указатель, чтобы nil - отправлять всем)
// Публичный метод работает под RLock
func (r *Room) Broadcast(e Event, excludeID *MemberID) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.broadcastUnlocked(e, excludeID)
}

// broadcastUnlocked - внутренний метод рассылки события членам комнаты (excludeID - кого исключаем)
// Требует захвата r.mu.RLock
func (r *Room) broadcastUnlocked(e Event, excludeID *MemberID) {
	for id, m := range r.members {
		if excludeID != nil && id == *excludeID {
			continue
		}
		select {
		case m.Events() <- e:
			log.Printf("Sent event (%s) to member ID =%d", e.String(), id)
		default:
			// Канал полон - дропаем медленного клиента
			log.Printf("Dropping slow member ID=%d", id)
			m.Close()
		}
	}
}

// AddToHistory добавляет событие в историю
//
//	Безопасный публичный метод с защитой от OOM
func (r *Room) AddToHistory(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addHistoryUnlocked(e)
}

// addHistoryUnlocked - внутренний метод добавления события в историю комнаты
// Требует захвата r.mu.Lock
// Встроен алгоритм trim для истории
func (r *Room) addHistoryUnlocked(e Event) {
	r.history = append(r.history, e)
	r.trimHistory()
}

// GetMemberByPublicKey — поиск по base64-ключу
func (r *Room) GetMemberByPublicKey(publicKey string) (*Member, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.memberByPubKey[publicKey]
	return m, ok
}

// GetHistorySnapshot возвращает копию последних InitialHistoryCount событий
func (r *Room) GetHistorySnapshot() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := len(r.history)
	if total <= cfg.initialHistoryLength {
		snapshot := make([]Event, total)
		copy(snapshot, r.history)
		return snapshot
	}
	start := total - cfg.initialHistoryLength
	snapshot := make([]Event, cfg.initialHistoryLength)
	copy(snapshot, r.history[start:])
	return snapshot
}

// trimHistory - обрезает историю событий в комнате по кол-ву MaxHistoryStorage
func (r *Room) trimHistory() {
	if len(r.history) > cfg.maxHistoryStorageLength {
		r.history = r.history[len(r.history)-cfg.maxHistoryStorageLength:]
	}
}

func (r *Room) NextMemberID() (MemberID, bool) {
	if MemberID(r.nextMemberID.Load()) == cfg.maxMemberID {
		return 0, false
	}
	return MemberID(r.nextMemberID.Add(1)), true
}

func (r *Room) Close() {
	// Собираем участников под локом
	membersToClose := r.collectMembersAndClear()

	// Закрываем участников вне локов
	for _, m := range membersToClose {
		m.Close() // sync.Once гарантирует безопасность
	}

	log.Printf("Room %q closed", r.name)
}

// collectMembersAndClear собирает ссылки на участников и очищает структуры комнаты,
// выполняется под мьютексом, возвращает срез для безопасного закрытия вне локов
func (r *Room) collectMembersAndClear() []*Member {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Printf("Closing room %q, members count: %d", r.name, len(r.members))

	membersToClose := make([]*Member, 0, len(r.members))
	for _, m := range r.members {
		membersToClose = append(membersToClose, m)
	}

	// Очищаем структуры — новые Register не попадут в закрытую комнату
	clear(r.members)
	clear(r.memberByPubKey)
	r.history = nil

	return membersToClose
}
