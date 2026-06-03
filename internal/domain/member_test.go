package domain

import (
	"sync"
	"testing"
)

func NewMember(id MemberID, pubkey string) *Member {
	mc := MemberConfig{}
	mc.withDefaults()

	mf := NewMemberFactory(mc)
	return mf.NewMember(id, pubkey)
}

func TestMember_LifecycleAndActivation(t *testing.T) {
	t.Parallel()

	id := MemberID(10)
	pubKey := "dGVzdF9wdWJrZXlfYmFzZTY0" // dummy base64
	m := NewMember(id, pubKey)

	if m.ID() != id {
		t.Errorf("ID() = %v, want %v", m.ID(), id)
	}
	if m.PubKeyB64() != pubKey {
		t.Errorf("PubKeyB64() = %v, want %v", m.PubKeyB64(), pubKey)
	}
	if m.IsActivated() {
		t.Error("IsActivated() should be false initially")
	}

	m.SetActivated()
	if !m.IsActivated() {
		t.Error("IsActivated() should be true after SetActivated()")
	}
}

func TestMember_Close_Idempotent(t *testing.T) {
	t.Parallel()

	m := NewMember(1, "key1")

	// 1. Первый вызов Close()
	m.Close()

	// Проверяем, что канал done закрыт
	select {
	case <-m.Done():
		// OK
	default:
		t.Fatal("Done() channel should be closed after first Close()")
	}

	// Проверяем, что канал events закрыт (пустой буфер -> сразу ok=false)
	_, ok := <-m.Events()
	if ok {
		t.Fatal("Events() channel should be closed after Close()")
	}

	// 2. Второй вызов Close() (идемпотентность)
	// Должен пройти без паники благодаря sync.Once
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Close() panicked on second call: %v", r)
		}
	}()
	m.Close()
}

func TestMember_Close_Concurrent(t *testing.T) {
	t.Parallel()

	m := NewMember(2, "key2")
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 100 горутин одновременно вызывают Close()
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			m.Close()
		}()
	}

	wg.Wait()

	// Проверяем, что каналы корректно закрыты после гонки
	select {
	case <-m.Done():
	default:
		t.Fatal("Done() channel should be closed after concurrent Close() calls")
	}

	_, ok := <-m.Events()
	if ok {
		t.Fatal("Events() channel should be closed after concurrent Close() calls")
	}
}

func TestMember_Events_Channel_Read(t *testing.T) {
	t.Parallel()

	m := NewMember(3, "key3")
	defer m.Close()

	// Эмулируем отправку события из Room.Run()
	// Пишем во внутреннее поле m.events напрямую (тест в том же пакете)
	go func() {
		m.events <- NewMsgEvent(55, []byte("async msg"))
	}()

	select {
	case evt := <-m.Events():
		if evt.SenderID != 55 || evt.Kind != EventMsg || string(evt.Message) != "async msg" {
			t.Errorf("Unexpected event received: %+v", evt)
		}
	case <-m.Done():
		t.Fatal("Events channel read failed or member closed prematurely")
	}
}
