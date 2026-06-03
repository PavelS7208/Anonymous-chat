package domain

import (
	"time"
)

// UnixTS — алиас для временной метки в секундах с начала UNIX-эпохи.
// Используется для совместимости с протоколом и упрощения чтения кода.
type UnixTS = int64

// EventType определяет тип события в чате.
// Соответствует значению {verb} в текстовом протоколе.
type EventType int

const (
	// EventJoin — участник присоединился к комнате.
	// Формат в протоколе: "{timestamp} {id} joined\n"
	EventJoin EventType = iota

	// EventLeft — участник покинул комнату.
	// Формат в протоколе: "{timestamp} {id} left\n"
	EventLeft

	// EventMsg — обычное сообщение от участника.
	// Формат в протоколе: "{timestamp} {id} : {message}\n"
	EventMsg
)

// String возвращает строковое представление типа события для протокола.
// Используется при форматировании события в байты для отправки в стрим.
func (t EventType) String() string {
	switch t {
	case EventJoin:
		return "joined"
	case EventLeft:
		return "left"
	case EventMsg:
		return ":"
	default:
		return "unknown"
	}
}

// Event представляет неизменяемое событие чата.
// Создаётся только через фабричные функции (NewJoinEvent, NewLeftEvent, NewMsgEvent).
// Поля экспортированы для zero-copy чтения в форматтерах и стримерах.
// Контракт: после создания поля не должны модифицироваться.
type Event struct {
	// Seq — монотонный счётчик события.
	// Присваивается в Room.Broadcast() для защиты от гонок при отдаче снапшота истории.
	// Используется стримером для фильтрации дублей: события с Seq <= lastSnapshotSeq игнорируются.
	Seq uint64

	// Timestamp — время события в секундах с начала UNIX-эпохи (серверное время).
	// Источник истины для синхронизации клиентов.
	Timestamp UnixTS

	// SenderID — идентификатор участника, инициировавшего событие.
	// Уникален в пределах комнаты и сессии (1..2^63-1).
	SenderID MemberID

	// Kind — тип события: joined / left / :
	// Определяет формат и семантику сообщения.
	Kind EventType

	// Message — тело сообщения в байтах.
	// Для EventJoin и EventLeft — пустой срез (nil или len=0).
	// Для EventMsg — валидированный контент (1..1024 байта, UTF-8, без \n, без пробелов по краям).
	// Хранится как []byte для эффективности и корректной работы с криптографией.
	Message []byte
}

// NewJoinEvent создаёт событие присоединения участника.
// Timestamp устанавливается в текущее серверное время.
func NewJoinEvent(id MemberID) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		SenderID:  id,
		Kind:      EventJoin,
		Message:   nil,
	}
}

// NewLeftEvent создаёт событие отключения участника.
// Timestamp устанавливается в текущее серверное время.
func NewLeftEvent(id MemberID) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		SenderID:  id,
		Kind:      EventLeft,
		Message:   nil,
	}
}

// NewMsgEvent создаёт событие обычного сообщения.
// Timestamp устанавливается в текущее серверное время.
// content должен быть предварительно валидирован (см. domain/message/validators.go).
func NewMsgEvent(id MemberID, content []byte) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		SenderID:  id,
		Kind:      EventMsg,
		Message:   content,
	}
}

// В конец файла event.go добавить:

// Marshal реализует интерфейс WireMessage для прямой передачи в StreamWriter по форматам из ТЗ
func (e Event) Marshal() []byte {
	return formatEvent(e)
}

// IsSystem возвращает true, если событие системное (joined/left), а не пользовательское сообщение.
// Может использоваться для фильтрации при логировании или обработке.
func (e Event) IsSystem() bool {
	return e.Kind == EventJoin || e.Kind == EventLeft
}
