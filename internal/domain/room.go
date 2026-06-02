package domain

import (
	"sync"
	"sync/atomic"
)

// eventHistory - Контракт для работы с хранилищем событий в комнате
// Push - сохранить событие
// LastN - вернуть N последних событий истории
type eventHistory interface {
	Push(evt Event)
	Snapshot() []Event
	Len() int
}

// Room представляет комнату анонимного чата.
// Управляет участниками, историей сообщений и асинхронной рассылкой событий.
// Ответственность за корректную очистку на вызывающем (вызвать LeaveFunc)
type Room struct {
	name string

	mu             sync.RWMutex
	members        map[MemberID]*Member
	memberByPubKey map[string]*Member

	history       eventHistory // Хранилище событий в комнате
	memberFactory MemberFactory

	broadcast chan Event // Очередь для Run(). Блокирующая запись гарантирует отсутствие потерь

	nextMemberID atomic.Int64  //  ID участника
	seq          atomic.Uint64 //  Глобальный монотонный счётчик событий

	closed atomic.Bool // Флаг для корректного закрытия комнаты
}

// NewRoom создаёт комнату. Инициализирует структуры с параметрами из cfg.
// func NewRoom(name string) *Room  - такой нет создание через Фабрику
/*
func NewRoom(name string) *Room {}
*/

//  Геттеры нужные

func (r *Room) Name() string { return r.name }

// GetMemberByPubKey находит участника по ключу публичному
func (r *Room) GetMemberByPubKey(pubKeyB64 string) (*Member, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.memberByPubKey[pubKeyB64]
	return m, ok
}

// MemberCount возвращает текущее число участников
func (r *Room) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

// -------  Основные методы ----------

type LeaveFunc func()

// Join регистрирует нового участника.
// Возвращает участника, снапшот последних событий из истории,
// seq события (для обработки и дополнительной загрузки остатков)
// и идемпотентную функцию очистки (ответственность на вызывающем вызвать очистку ресурсов)
// error в случае проблем
func (r *Room) Join(pubKeyB64 string) (*Member, []Event, uint64, LeaveFunc, error) {

	newMemberID, err := r.allocateID()
	if err != nil {
		return nil, nil, 0, func() {}, err
	}
	newMember := r.memberFactory.NewMember(newMemberID, pubKeyB64)

	// Создаем leave-функцию
	var once sync.Once
	leave := func() {
		once.Do(func() {
			if newMember == nil { // Защита на всякий случай
				return
			}
			// Если уже зарегили в списке - удаляем. Под локом
			r.mu.Lock()
			if cur, ok := r.members[newMemberID]; ok && cur == newMember {
				delete(r.members, newMemberID)
				delete(r.memberByPubKey, pubKeyB64)
			}
			r.mu.Unlock()

			newMember.Close()
		})
	}

	// Регистрация под локом с проверкой на максимум в комнате
	r.mu.Lock()
	if _, exists := r.memberByPubKey[pubKeyB64]; exists { // Защити от атаки с дублями подключений
		r.mu.Unlock()
		return nil, nil, 0, leave, ErrMemberAlreadyExists
	}
	r.members[newMemberID] = newMember
	r.memberByPubKey[pubKeyB64] = newMember

	// Снимаем снапшот и фиксируем seq
	snapshot := r.history.Snapshot()
	lastSeq := r.seq.Load()
	r.mu.Unlock()

	return newMember, snapshot, lastSeq, leave, nil
}

// Broadcast вводит событие в систему: назначает Seq, сохраняет в историю, кладёт в очередь рассылки.
func (r *Room) Broadcast(evt Event) {
	r.mu.Lock()

	evt.Seq = r.seq.Add(1) // под лок, чтобы seq была последовательной в истории
	r.history.Push(evt)
	r.mu.Unlock()
	// Теоретически тут гонка тоже есть, но она настолько минимальны,
	// что стресс тест ее не поймал (TestBroadcast_SeqOrder).
	//  Если будет - то думать про что-то уже совсем сложное типа отдельного веркера
	r.broadcast <- evt
}

// Run - фоновая горутина fan-out рассылки.
// Читает из broadcast и асинхронно доставляет события участникам.
// Использует алгоритм "последний шанс" для клиента у которого буфер заполнился (доп массив overflow)
func (r *Room) Run() {
	for event := range r.broadcast {
		var deadIDs []MemberID

		r.mu.RLock()
		for _, m := range r.members {
			if event.IsSystem() && m.id == event.SenderID {
				continue
			}

			// Дренаж overflow в строгом FIFO-порядке (peek-then-pop)
			// Гарантирует: старое событие уходит первым, новые не могут его обогнать.
			for {
				// Заглядываем в голову буфера, но НЕ удаляем событие
				ovEvent, ok := m.overflow.peek()
				if !ok {
					break // Буфер пуст
				}

				// Пытаемся доставить событие в основной канал
				select {
				case m.events <- ovEvent:
					// Успех: удаляем из буфера
					_, _ = m.overflow.pop()
					// Продолжаем цикл: пробуем доставить следующее событие в очереди
				default:
					// Неудача: основной канал всё ещё полон
					// Оставляем событие в голове буфера
					// Прерываем дренаж: если старое не прошло, новые тоже упрутся
					break
				}
			}

			// Обработка нового события (event) из широковещательной рассылки

			if m.overflow.len() > 0 {
				// Overflow не пуст
				// новое событие добавляем в хвост очереди
				// Оно будет доставлено после того, как уйдут все старые (гарантия FIFO)
				if !m.overflow.push(event) {
					// Буфер переполнен, клиент не справляется и с доп буфером - отключаем
					m.Close()
					deadIDs = append(deadIDs, m.id)
				}
			} else {
				// пробуем отправить новое событие из основного буфера (overflow - пуст)
				select {
				case m.events <- event:
					// Доставлено успешно
				default:
					// Основной канал полон - кладём новое событие в overflow
					if !m.overflow.push(event) {
						// Переполнение overflow - клиент не справляется, отключаем
						m.Close()
						deadIDs = append(deadIDs, m.id)
					}
				}
			}
		}
		r.mu.RUnlock()

		if len(deadIDs) > 0 {
			r.mu.Lock()
			activatedIDs := r.removeMembersUnlocked(deadIDs)
			r.mu.Unlock()
			for _, id := range activatedIDs {
				r.Broadcast(NewLeftEvent(id))
			}
		}
	}
}

// Close gracefully останавливает комнату и освобождает ресурсы.
// Потоко безопасен, идемпотентен (можно вызывать многократно).
func (r *Room) Close() {
	// Быстрая проверка: если уже закрыто — выходим
	if !r.closed.CompareAndSwap(false, true) {
		return
	}
	r.cleanup()
}

// ------  необходимые helper-ы -------------------------

// allocateID безопасно генерирует уникальный ID в диапазоне [Min, Max] через CAS-петлю.
func (r *Room) allocateID() (MemberID, error) {
	for {
		cur := r.nextMemberID.Load()
		if cur >= int64(maxMemberID) {
			return 0, ErrMemberIDOverflow
		}
		next := cur + 1
		if r.nextMemberID.CompareAndSwap(cur, next) {
			return MemberID(next), nil
		}
	}
}

// removeMembersUnlocked удаляет участников из мап и возвращает список ID у кого из удаленных был отправлен Join
// Контракт: вызывать ТОЛЬКО при захваченном r.mu.Lock().
// Возвращаемые ID используются вызывающей стороной для отправки Left-событий через Broadcast()
func (r *Room) removeMembersUnlocked(deadIDs []MemberID) []MemberID {
	var activatedIDs []MemberID

	for _, id := range deadIDs {
		if m, ok := r.members[id]; ok {
			delete(r.members, id)
			delete(r.memberByPubKey, m.PubKeyB64())

			if m.IsActivated() {
				activatedIDs = append(activatedIDs, id)
			}
		}
	}

	return activatedIDs
}

func (r *Room) cleanup() {
	// Собираем участников под локом
	members := r.collectMembersAndClear()

	// Закрываем канал рассылки
	close(r.broadcast)

	// Закрываем участников вне локов
	for _, m := range members {
		m.Close()
	}
}

// collectMembersAndClear собирает ссылки на участников и очищает структуры комнаты.
// Выполняется под мьютексом, возвращает срез для безопасного закрытия вне локов.
func (r *Room) collectMembersAndClear() []*Member {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Копируем ссылки на участников
	membersToClose := make([]*Member, 0, len(r.members))
	for _, m := range r.members {
		membersToClose = append(membersToClose, m)
	}

	// Очищаем структуры — новые запросы не попадут в закрытую комнату
	clear(r.members)
	clear(r.memberByPubKey)

	// Обнуляем историю — освобождаем память (особенно важно для больших []byte в Message)
	r.history = nil

	return membersToClose
}
