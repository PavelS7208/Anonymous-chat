package service

import (
	"anonymous-chat/internal/chat"
	"anonymous-chat/internal/crypto"
	"context"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync"
	"time"
)

type anonymousChat struct {
	rooms map[string]*chat.Room
	mu    sync.RWMutex
}

// NewAnonymousChat конструктор дефолтный
func NewAnonymousChat() AnonymousChat {
	return &anonymousChat{
		rooms: make(map[string]*chat.Room),
	}
}

// --------------- Реализация для интерфейса RoomProvider ------------------------------

// GetOrCreate - Защищенный (двойной защитой) метод получения/создания комнаты по имени
// ПРедполагаем что слои выше уже проверили на name == ""  и сюда уже пришло корректным
func (s *anonymousChat) GetOrCreate(name string) *chat.Room {
	s.mu.RLock()
	room, ok := s.rooms[name]
	s.mu.RUnlock()
	if ok {
		return room
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if room, ok = s.rooms[name]; ok {
		return room
	}
	room = chat.NewRoom(name)
	s.rooms[name] = room
	return room
}

// Get - Защищенный метод получения комнаты по имени, если ее нет, то false во втором параметре
func (s *anonymousChat) Get(name string) (*chat.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[name]
	return room, ok
}

// --------------- Реализация для интерфейса ChatStreamer ------------------------

func (s *anonymousChat) StreamChat(ctx context.Context, w ChunkedTransferWriter, roomName string) error {
	// проверка на формат имени комнаты
	if err := ValidateRoomName(roomName); err != nil {
		return fmt.Errorf("Validate roomName: %w", err)
	}

	room := s.GetOrCreate(roomName)
	log.Printf("room name: %s", roomName)

	// Генерим два ключа для создания Member
	privateSeed, publicKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}

	// Получаем NextID от комнаты, так как уникальность требуется в пределах комнаты
	id, ok := room.NextMemberID()
	if !ok {
		return chat.ErrMemberIdOverflow
	}
	member := chat.NewMember(id, crypto.EncodeBase64(publicKey))

	if err := room.Register(member); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	defer func() {
		member.Close()
		room.Unregister(member)
		leftEvent := chat.NewLeftEvent(time.Now().Unix(), id)
		room.AddToHistory(leftEvent)
		room.Broadcast(leftEvent, &id) // Информируем всех о выходе кроме себя
	}()

	// Создаем init сообщение (с приватным ключом) и затираем его сразу, чтобы уменьшить уезвимости
	init := NewInitMessage(privateSeed, id)
	if err := w.WriteAndFlush(init); err != nil {
		return fmt.Errorf("send init: %w", err)
	}
	clear(privateSeed)                 // вместо ручного цикла забиваем zero-value приватный ключ
	runtime.KeepAlive(&privateSeed[0]) // Прочел, что это для параноиков безопасности. 100% удалена
	init = ""                          // "Забываем" строку, чтобы GC быстрее забрал

	events := room.GetHistorySnapshot()

	joinEvent := chat.NewJoinEvent(time.Now().Unix(), id)
	room.AddToHistory(joinEvent)

	// Пишем историю комнаты чанками в поток
	for _, event := range events {
		if err := w.WriteAndFlush(event.String() + "\n"); err != nil {
			return fmt.Errorf("send history: %w", err)
		}
	}

	room.Broadcast(joinEvent, &id)

	// непосредственно сам стриминг
	// Отправляем себе, все что появляется пока не закроемся
	for {
		select {
		case event, ok := <-member.Events():
			if !ok {
				return nil
			}
			//log.Printf("Получили событие %s member id=%d", event.String(), id)
			if err := w.WriteAndFlush(event.String() + "\n"); err != nil {
				return fmt.Errorf("send event: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-member.Done():
			return nil
		}
	}
}

// --------------- Реализация для интерфейса MessagePoster -----

func (s *anonymousChat) PostMessage(ctx context.Context, roomName string, msg MessageData) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	room, exist := s.Get(roomName)
	if !exist {
		return ErrRoomNotFound
	}

	// Валидация бизнес-правил
	if err := ValidateMessage(msg.Message); err != nil {
		return fmt.Errorf("Validate message: %w", err)
	}

	// Криптографическая проверка подписи
	if err := crypto.VerifyMessageBase64(msg.PubKeyB64, msg.SigB64, []byte(msg.Message)); err != nil {
		return mapCryptoError(err)
	}

	// Поиск участника по pubKeyB64
	member, exist := room.GetMemberByPublicKey(msg.PubKeyB64)
	if !exist {
		return ErrMemberUnauthorized
	}

	// Создание и рассылка события
	event := chat.NewMsgEvent(time.Now().Unix(), member.ID(), msg.Message)
	room.Broadcast(event, nil) // nil = всем, включая отправителя

	return nil
}

// --------------- Реализация для интерфейса Closer -----

// Close закрывает все комнаты сервиса
func (s *anonymousChat) Close() error {
	// Собираем список комнат под локом (в изолированном пространстве)
	roomsToClose := s.collectRoomsAndClear()

	// Закрываем комнаты вне локов
	for _, room := range roomsToClose {
		room.Close()
	}

	log.Println("anonymousChat closed")
	return nil
}

// Убедимся, что интерфейс реализован (compile-time check)
var _ io.Closer = (*anonymousChat)(nil)

// --------------- Сервисные методы -----

// collectRoomsAndClear собирает ссылки на все комнаты и очищает мапу,
// выполняется под мьютексом, возвращает срез для безопасного закрытия вне локов
func (s *anonymousChat) collectRoomsAndClear() []*chat.Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Closing anonymousChat, rooms count: %d", len(s.rooms))

	roomsToClose := make([]*chat.Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		roomsToClose = append(roomsToClose, room)
	}

	// Очищаем мапу — новые запросы не попадут в закрытые комнаты
	clear(s.rooms)

	return roomsToClose
}

// NewInitMessage формирует приветствие
func NewInitMessage(privateSeed []byte, memberID chat.MemberID) string {
	welcome := fmt.Sprintf("%d %s\n", memberID, crypto.EncodeBase64(privateSeed))
	return welcome
}
