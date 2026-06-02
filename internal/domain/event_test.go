package domain

import (
	"testing"
)

func TestEventType_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind EventType
		want string
	}{
		{EventJoin, "joined"},
		{EventLeft, "left"},
		{EventMsg, ":"},
		{EventType(99), "unknown"}, // проверка default case
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("EventType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewJoinEvent(t *testing.T) {
	t.Parallel()
	id := MemberID(1)
	evt := NewJoinEvent(id)

	if evt.SenderID != id {
		t.Errorf("SenderID = %v, want %v", evt.SenderID, id)
	}
	if evt.Kind != EventJoin {
		t.Errorf("Kind = %v, want %v", evt.Kind, EventJoin)
	}
	if len(evt.Message) != 0 {
		t.Errorf("Message should be empty for Join event, got %v", evt.Message)
	}
	if evt.Timestamp <= 0 {
		t.Errorf("Timestamp should be set to server time")
	}
	if !evt.IsSystem() {
		t.Error("Join event should be recognized as system event")
	}
}

func TestNewLeftEvent(t *testing.T) {
	t.Parallel()
	id := MemberID(42)
	evt := NewLeftEvent(id)

	if evt.SenderID != id {
		t.Errorf("SenderID = %v, want %v", evt.SenderID, id)
	}
	if evt.Kind != EventLeft {
		t.Errorf("Kind = %v, want %v", evt.Kind, EventLeft)
	}
	if len(evt.Message) != 0 {
		t.Errorf("Message should be empty for Left event")
	}
	if !evt.IsSystem() {
		t.Error("Left event should be recognized as system event")
	}
}

func TestNewMsgEvent(t *testing.T) {
	t.Parallel()
	id := MemberID(7)
	payload := []byte("hello world")
	evt := NewMsgEvent(id, payload)

	if evt.SenderID != id {
		t.Errorf("SenderID = %v, want %v", evt.SenderID, id)
	}
	if evt.Kind != EventMsg {
		t.Errorf("Kind = %v, want %v", evt.Kind, EventMsg)
	}
	if string(evt.Message) != "hello world" {
		t.Errorf("Message = %q, want %q", string(evt.Message), "hello world")
	}
	if evt.IsSystem() {
		t.Error("Msg event should NOT be recognized as system event")
	}
}
