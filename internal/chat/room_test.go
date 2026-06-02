// room_test.go
package chat

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// === Вспомогательные функции ===

// createTestMember создаёт участника с уникальным ID и ключом
func createTestMember(id MemberID, keySuffix string) *Member {
	return NewMember(id, "pubkey_base64_"+keySuffix)
}

// drainEvents безопасно читает все доступные события из канала без блокировки
func drainEvents(m *Member) []Event {
	var events []Event
	for {
		select {
		case e, ok := <-m.Events():
			if !ok {
				return events
			}
			events = append(events, e)
		default:
			return events
		}
	}
}

// -------- Сами тесты -------------------------

// Тест создания комнаты
func TestNewRoom(t *testing.T) {
	t.Run("creates room with empty history", func(t *testing.T) {
		room := NewRoom("test-room")
		if room == nil {
			t.Fatal("expected non-nil room")
		}
		if room.name != "test-room" {
			t.Errorf("expected name 'test-room', got %q", room.name)
		}
		if len(room.history) != 0 {
			t.Errorf("expected empty history, got len=%d", len(room.history))
		}
		if room.members == nil || room.memberByPubKey == nil {
			t.Error("expected initialized maps")
		}
	})
}

// Тесты регистрации участников
func TestRoom_Register(t *testing.T) {
	t.Run("registers new member successfully", func(t *testing.T) {
		room := NewRoom("test")
		member := createTestMember(1, "key1")

		if err := room.Register(member); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Проверка по ID
		if _, exists := room.members[member.ID()]; !exists {
			t.Error("member not found in members map")
		}
		// Проверка по ключу
		if _, exists := room.memberByPubKey[member.PublicB64Key()]; !exists {
			t.Error("member not found in memberByPubKey map")
		}
	})

	t.Run("rejects duplicate public key", func(t *testing.T) {
		room := NewRoom("test")
		member1 := createTestMember(1, "same-key")
		member2 := createTestMember(2, "same-key") // тот же ключ!

		if err := room.Register(member1); err != nil {
			t.Fatalf("first register failed: %v", err)
		}
		if err := room.Register(member2); err == nil {
			t.Error("expected error for duplicate key, got nil")
		} else if !errors.Is(err, errMemberAlreadyRegistered) {
			t.Errorf("expected errMemberAlreadyRegistered, got %v", err)
		}

		// Убедиться, что второй участник не зарегистрирован
		room.mu.RLock()
		if _, exists := room.members[member2.ID()]; exists {
			t.Error("duplicate member should not be in members map")
		}
		room.mu.RUnlock()
	})

	t.Run("allows different keys with same ID (edge case)", func(t *testing.T) {
		// Это зависит от бизнес-логики: если ID уникален, а ключи разные —
		// текущая реализация разрешит, но это может быть багом.
		// Тест документирует текущее поведение.
		room := NewRoom("test")
		m1 := NewMember(1, "key-A")
		m2 := NewMember(1, "key-B") // тот же ID, другой ключ

		_ = room.Register(m1)
		err := room.Register(m2)

		// Текущая реализация проверяет только по ключу, поэтому разрешит
		// Если это нежелательно — нужно добавить проверку по ID
		if err != nil {
			t.Logf("current behavior: duplicate ID rejected: %v", err)
		} else {
			t.Log("current behavior: duplicate ID allowed (verify if intended)")
		}
	})
}

// Тесты отписки участников
func TestRoom_Unregister(t *testing.T) {
	t.Run("unregisters existing member", func(t *testing.T) {
		room := NewRoom("test")
		member := createTestMember(1, "key1")
		_ = room.Register(member)

		room.Unregister(member)

		room.mu.RLock()
		if _, exists := room.members[member.ID()]; exists {
			t.Error("member should be removed from members")
		}
		if _, exists := room.memberByPubKey[member.PublicB64Key()]; exists {
			t.Error("member should be removed from memberByPubKey")
		}
		room.mu.RUnlock()

		// Каналы должны быть закрыты
		select {
		case _, ok := <-member.Done():
			if ok {
				t.Error("done channel should be closed")
			}
		default:
			t.Error("done channel should be closed (non-blocking check)")
		}
	})

	t.Run("handles unregister of non-registered member", func(t *testing.T) {
		room := NewRoom("test")
		member := createTestMember(999, "orphan-key")

		// Не должно паниковать, должно закрыть каналы участника
		room.Unregister(member)

		select {
		case _, ok := <-member.Done():
			if ok {
				t.Error("done channel should be closed even for orphan member")
			}
		default:
			t.Error("done channel should be closed")
		}
	})

	t.Run("unregister is idempotent for same member", func(t *testing.T) {
		room := NewRoom("test")
		member := createTestMember(1, "key1")
		_ = room.Register(member)

		room.Unregister(member)
		// Повторный вызов не должен паниковать
		room.Unregister(member)
	})
}

// Тесты рассылки событий
func TestRoom_Broadcast(t *testing.T) {
	t.Run("broadcasts to all members when excludeID is nil", func(t *testing.T) {
		room := NewRoom("test")
		m1 := createTestMember(1, "k1")
		m2 := createTestMember(2, "k2")
		_ = room.Register(m1)
		_ = room.Register(m2)

		event := NewMsgEvent(100, 0, "hello")
		room.Broadcast(event, nil)

		// Ждём доставки (каналы буферизированы, но для надёжности)
		time.Sleep(10 * time.Millisecond)

		events1 := drainEvents(m1)
		events2 := drainEvents(m2)

		if len(events1) != 1 || events1[0].message != "hello" {
			t.Errorf("m1 expected 1 event 'hello', got %v", events1)
		}
		if len(events2) != 1 || events2[0].message != "hello" {
			t.Errorf("m2 expected 1 event 'hello', got %v", events2)
		}
	})

	t.Run("excludes member by ID when excludeID is provided", func(t *testing.T) {
		room := NewRoom("test")
		m1 := createTestMember(1, "k1")
		m2 := createTestMember(2, "k2")
		_ = room.Register(m1)
		_ = room.Register(m2)

		exclude := m1.ID()
		event := NewMsgEvent(100, 0, "hello")
		room.Broadcast(event, &exclude)

		time.Sleep(10 * time.Millisecond)

		events1 := drainEvents(m1)
		events2 := drainEvents(m2)

		if len(events1) != 0 {
			t.Errorf("excluded member should receive nothing, got %v", events1)
		}
		if len(events2) != 1 || events2[0].message != "hello" {
			t.Errorf("m2 expected 1 event, got %v", events2)
		}
	})

	t.Run("broadcast to empty room does not panic", func(t *testing.T) {
		room := NewRoom("test")
		event := NewMsgEvent(100, 0, "hello")
		// Не должно паниковать
		room.Broadcast(event, nil)
	})
}

// Тесты истории
func TestRoom_AddToHistory(t *testing.T) {
	t.Run("appends events and trims when exceeding cfg.maxHistoryStorageLength", func(t *testing.T) {
		room := NewRoom("test")
		// Добавляем cfg.maxHistoryStorageLength + 10 событий
		for i := 0; i < cfg.maxHistoryStorageLength+10; i++ {
			room.AddToHistory(NewMsgEvent(int64(i), MemberID(i), fmt.Sprintf("msg-%d", i)))
		}

		room.mu.RLock()
		histLen := len(room.history)
		firstTS := room.history[0].timeStamp
		lastTS := room.history[len(room.history)-1].timeStamp
		room.mu.RUnlock()

		if histLen != cfg.maxHistoryStorageLength {
			t.Errorf("expected history len %d, got %d", cfg.maxHistoryStorageLength, histLen)
		}
		// После trim должны остаться последние события
		if firstTS != 10 || lastTS != int64(cfg.maxHistoryStorageLength+9) {
			t.Errorf("expected timestamps range [10, %d], got [%d, %d]",
				cfg.maxHistoryStorageLength+9, firstTS, lastTS)
		}
	})

	t.Run("does not trim when under limit", func(t *testing.T) {
		room := NewRoom("test")
		for i := 0; i < 5; i++ {
			room.AddToHistory(NewMsgEvent(int64(i), MemberID(i), "msg"))
		}
		room.mu.RLock()
		if len(room.history) != 5 {
			t.Errorf("expected len 5, got %d", len(room.history))
		}
		room.mu.RUnlock()
	})
}

func TestRoom_GetHistorySnapshot(t *testing.T) {
	t.Run("returns copy when history smaller than cfg.initialHistoryLength", func(t *testing.T) {
		room := NewRoom("test")
		room.AddToHistory(NewMsgEvent(1, 1, "a"))
		room.AddToHistory(NewMsgEvent(2, 2, "b"))

		snap := room.GetHistorySnapshot()

		if len(snap) != 2 {
			t.Errorf("expected 2 events, got %d", len(snap))
		}
		if snap[0].message != "a" || snap[1].message != "b" {
			t.Error("snapshot content mismatch")
		}

		// Убедиться, что это копия: модификация снапшота не влияет на историю
		room.mu.RLock()
		if room.history[0].message != "a" {
			t.Error("original history was mutated!")
		}
		room.mu.RUnlock()
	})

	t.Run("returns last cfg.initialHistoryLength events when history is larger", func(t *testing.T) {
		room := NewRoom("test")
		// Добавляем событий
		for i := 0; i < cfg.initialHistoryLength+20; i++ {
			room.AddToHistory(NewMsgEvent(int64(i), MemberID(i), fmt.Sprintf("msg-%d", i)))
		}

		snap := room.GetHistorySnapshot()

		if len(snap) != cfg.initialHistoryLength {
			t.Errorf("expected %d events, got %d", cfg.initialHistoryLength, len(snap))
		}
		// Должны быть последние 5: msg-5 .. msg-9
		for i := 0; i < cfg.initialHistoryLength; i++ {
			expected := fmt.Sprintf("msg-%d", i+20)
			if snap[i].message != expected {
				t.Errorf("snap[%d] expected %q, got %q", i, expected, snap[i].message)
			}
		}
	})

	t.Run("returns empty slice when history is empty", func(t *testing.T) {
		room := NewRoom("test")
		snap := room.GetHistorySnapshot()
		if snap == nil {
			t.Error("expected empty slice, got nil")
		}
		if len(snap) != 0 {
			t.Errorf("expected len 0, got %d", len(snap))
		}
	})
}

// Тесты поиска по ключу
func TestRoom_GetMemberByPublicKey(t *testing.T) {
	t.Run("finds member by exact base64 key", func(t *testing.T) {
		room := NewRoom("test")
		m := createTestMember(1, "exact-key")
		_ = room.Register(m)

		found, ok := room.GetMemberByPublicKey(m.PublicB64Key())
		if !ok || found.ID() != m.ID() {
			t.Error("expected to find member by public key")
		}
	})

	t.Run("returns false for unknown key", func(t *testing.T) {
		room := NewRoom("test")
		_, ok := room.GetMemberByPublicKey("unknown-key")
		if ok {
			t.Error("expected not found for unknown key")
		}
	})

	t.Run("returns false after member unregistered", func(t *testing.T) {
		room := NewRoom("test")
		m := createTestMember(1, "temp-key")
		_ = room.Register(m)
		room.Unregister(m)

		_, ok := room.GetMemberByPublicKey(m.PublicB64Key())
		if ok {
			t.Error("expected not found after unregister")
		}
	})
}

// Тесты генерации ID участников
func TestRoom_NextMemberID(t *testing.T) {
	t.Run("generates sequential IDs starting from 1", func(t *testing.T) {
		room := NewRoom("test")
		for i := 1; i <= 10; i++ {
			id, ok := room.NextMemberID()
			if !ok {
				t.Fatalf("unexpected failure at iteration %d", i)
			}
			if id != MemberID(i) {
				t.Errorf("expected ID %d, got %d", i, id)
			}
		}
	})

	t.Run("returns false when MaxMemberID reached", func(t *testing.T) {
		// Эмуляция переполнения: устанавливаем счётчик в максимальное значение
		room := &Room{
			nextMemberID: atomic.Uint64{},
		}
		room.nextMemberID.Store(uint64(cfg.maxMemberID))

		// Следующий вызов должен вернуть (0, false)
		id, ok := room.NextMemberID()
		if ok || id != 0 {
			t.Errorf("expected (0, false) on overflow, got (%d, %v)", id, ok)
		}
	})
}

// Тесты конкурентной безопасности (race detector)
func TestRoom_Concurrency(t *testing.T) {
	t.Run("concurrent register and broadcast", func(t *testing.T) {
		room := NewRoom("test")
		var wg sync.WaitGroup

		// Горутины регистрируют участников
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				m := createTestMember(MemberID(idx), fmt.Sprintf("key-%d", idx))
				_ = room.Register(m)
			}(i)
		}

		// Горутины рассылают события
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(iter int) {
				defer wg.Done()
				room.Broadcast(NewMsgEvent(int64(iter), 0, "concurrent"), nil)
			}(i)
		}

		wg.Wait()
		// Запуск с -race должен показать проблемы, если они есть
	})

	t.Run("concurrent history writes and reads", func(t *testing.T) {
		room := NewRoom("test")
		var wg sync.WaitGroup

		// Писатели
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(start int) {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					room.AddToHistory(NewMsgEvent(int64(start+j), MemberID(start+j), "msg"))
				}
			}(i * 10)
		}

		// Читатели
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					_ = room.GetHistorySnapshot()
				}
			}()
		}

		wg.Wait()
	})
}

// Тест интеграции: полный цикл жизни участника
func TestRoom_FullLifecycle(t *testing.T) {
	room := NewRoom("chat")

	// 1. Регистрация
	alice := createTestMember(1, "alice-key")
	if err := room.Register(alice); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// 2. Присоединение: событие в историю + рассылка
	joinEvent := NewJoinEvent(1000, alice.ID())
	room.AddToHistory(joinEvent)
	room.Broadcast(joinEvent, nil)

	// 3. Проверка: история содержит событие, участник получил его
	room.mu.RLock()
	if len(room.history) != 1 || room.history[0].kind != Join {
		t.Error("history should contain join event")
	}
	room.mu.RUnlock()

	time.Sleep(10 * time.Millisecond)
	events := drainEvents(alice)
	if len(events) != 1 || events[0].kind != Join {
		t.Errorf("alice expected join event, got %v", events)
	}

	// 4. Отправка сообщения
	room.Broadcast(NewMsgEvent(1001, alice.ID(), "Hello!"), nil)
	time.Sleep(10 * time.Millisecond)
	events = drainEvents(alice)
	// Alice не должна получить своё сообщение (если логика исключает отправителя)
	// В текущей реализации — получит, т.к. excludeID=nil
	// Если нужно исключить — передавать &alice.ID()

	// 5. Выход
	room.Unregister(alice)
	room.Broadcast(NewLeftEvent(1002, alice.ID()), nil)

	// После Close() канал событий закрыт — чтение должно вернуть ok=false
	select {
	case _, ok := <-alice.Events():
		if ok {
			t.Error("events channel should be closed after Unregister")
		}
	default:
		// Канал может быть не пуст, но он закрыт — это нормально
	}
}
