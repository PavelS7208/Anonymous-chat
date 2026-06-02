package chat

import "fmt"

type EventType int

const (
	Join EventType = iota
	Left
	Msg
)

func (t EventType) String() string {
	switch t {
	case Join:
		return "joined"
	case Left:
		return "left"
	case Msg:
		return ":"
	default:
		return "unknown"
	}
}

type Event struct {
	timeStamp int64
	memberID  int64
	message   string
	kind      EventType
}

func newEvent(timeStamp int64, memberID int64, kind EventType, message string) Event {
	return Event{
		timeStamp: timeStamp,
		memberID:  memberID,
		kind:      kind,
		message:   message,
	}
}

func NewJoinEvent(timeStamp int64, memberID int64) Event {
	return newEvent(timeStamp, memberID, Join, "")
}

func NewLeftEvent(timeStamp int64, memberID int64) Event {
	return newEvent(timeStamp, memberID, Left, "")
}

func NewMsgEvent(timeStamp int64, memberID int64, msg string) Event {
	return newEvent(timeStamp, memberID, Msg, msg)
}

func (e *Event) String() string {
	if e.kind == Msg {
		return fmt.Sprintf("%d %d : %s", e.timeStamp, e.memberID, e.message)
	}

	return fmt.Sprintf("%d %d %s", e.timeStamp, e.memberID, e.kind.String())
}
