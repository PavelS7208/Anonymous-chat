package chat

import "fmt"

type eventType int

const (
	Join eventType = iota
	Left
	Msg
)

func (t eventType) String() string {
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

type UnixTS = int64

type Event struct {
	timeStamp UnixTS
	memberID  MemberID
	message   string
	kind      eventType
}

func newEvent(timeStamp UnixTS, memberID MemberID, kind eventType, message string) Event {
	return Event{
		timeStamp: timeStamp,
		memberID:  memberID,
		kind:      kind,
		message:   message,
	}
}

func NewJoinEvent(timeStamp UnixTS, id MemberID) Event {
	return newEvent(timeStamp, id, Join, "")
}

func NewLeftEvent(timeStamp UnixTS, id MemberID) Event {
	return newEvent(timeStamp, id, Left, "")
}

func NewMsgEvent(timeStamp UnixTS, id MemberID, msg string) Event {
	return newEvent(timeStamp, id, Msg, msg)
}

func (e *Event) String() string {
	if e.kind == Msg {
		return fmt.Sprintf("%d %d : %s", e.timeStamp, e.memberID, e.message)
	}

	return fmt.Sprintf("%d %d %s", e.timeStamp, e.memberID, e.kind.String())
}
