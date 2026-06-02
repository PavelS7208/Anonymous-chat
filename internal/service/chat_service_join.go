package service

import (
	"context"
	"encoding/base64"

	"github.com/pavel/anonymous-chat/internal/domain"
)

// Join - проверки и алгоритм присоединения к комнате
// Все ошибки транслируются в HTTP 4xx/5xx на уровне handler-а.
// При успехе возвращает JoinSession — владелец захваченных ресурсов который перенаправляется в метод Stream
func (s *ChatService) Join(ctx context.Context, req JoinRequest) (*JoinSession, error) {

	// Защита от спам атак на создание/присоединение к комнатам с одного IP
	if err := s.limitGuard.AllowJoin(req.IP); err != nil {
		return nil, err // -> 429
	}

	// Защита на одновременно открытые сессии с одного IP
	if err := s.connTracker.Acquire(req.IP); err != nil {
		return nil, err // -> 429
	}
	// Флаг передачи владения: пока true — defer освободит слот при любой ошибке ниже
	connAcquired := true
	defer func() {
		if connAcquired {
			s.connTracker.Release(req.IP)
		}
	}()

	// Защита от атак на создание большого кол-ва комнат
	if s.repo.Count() >= s.cfg.MaxGlobalRooms {
		return nil, ErrGlobalCreatedRoomReached // -> 503
	}

	// Генерация ключей (крипто-провайдер инжектирован в сервис)
	privateSeed, pubKey, err := s.crypto.GenerateKeyPair()
	if err != nil {
		return nil, err // -> 500
	}
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	// Запоминаем IP сессии для защиты последующего Send
	s.sessionGuard.Register(pubKeyB64, req.IP)
	// Флаг передачи владения: пока true — defer почистит регистрацию при ошибке ниже
	sessionRegistered := true
	defer func() {
		if sessionRegistered {
			s.sessionGuard.Unregister(pubKeyB64)
		}
	}()

	// Ищем существующую или создаем новую комнату
	room, err := s.repo.GetOrCreate(ctx, req.RoomName)
	if err != nil {
		return nil, err // -> 500
	}

	// Защита от атак на создание большого числа участников
	if room.MemberCount() >= s.cfg.MaxMembersPerRoom {
		return nil, ErrGlobalJoinedMemberReached // -> 503
	}

	// Регистрация участника + получение leave-функции
	member, snapshot, lastSeq, leave, err := room.Join(pubKeyB64)
	if err != nil {
		return nil, err // -> 500
	}

	// Всё успешно — передаём владение ресурсами в JoinSession.
	// Оба defer выше должны стать без операций- сбрасываем флаги.
	connAcquired = false
	sessionRegistered = false

	// Захватываем локальные переменные в замыкание явно,
	// чтобы не захватить указатель на ещё не созданный session.
	capturedMember := member
	capturedRoom := room

	session := &JoinSession{
		room:        room,
		member:      member,
		snapshot:    snapshot,
		lastSeq:     lastSeq,
		privateSeed: privateSeed,
		release: func() {
			// LeftEvent только если участник успел активироваться в Stream
			if capturedMember.IsActivated() {
				capturedRoom.Broadcast(domain.NewLeftEvent(capturedMember.ID()))
			}
			leave()
			s.sessionGuard.Unregister(pubKeyB64)
			s.connTracker.Release(req.IP)
		},
	}

	return session, nil
}

// Stream — инициализация клиента и стриминг событий по выданной сессии.
// Вызывается ПОСЛЕ открытия ChunkedStreamer — первый Write фиксирует HTTP 200.
// Гарантирует освобождение ресурсов сессии через defer session.release().
func (s *ChatService) Stream(ctx context.Context, session *JoinSession, w ChatWriter) error {
	// Единственная точка вызова release — гарантирован ровно один раз.
	// LeftEvent внутри release учитывает IsActivated — безопасно при любом исходе.
	defer session.release()

	// Тайм-аут handshake: клиент должен получить bootstrap и снапшот за это время
	hCtx, cancel := context.WithTimeout(ctx, s.cfg.HandshakeTimeout)
	defer cancel()

	// Отправка bootstrap — первый Write, HTTP 200 фиксируется здесь
	if err := w.Write(hCtx, domain.NewBootstrap(session.member.ID(), session.privateSeed)); err != nil {
		return err
	}
	session.privateSeed = nil // приватный ключ больше не нужен — обнуляем

	// Отправка снапшота истории комнаты
	if err := sendSnapshot(hCtx, session.snapshot, w); err != nil {
		return err
	}

	// Handshake завершён — активируем участника и оповещаем комнату
	session.member.SetActivated()
	session.room.Broadcast(domain.NewJoinEvent(session.member.ID()))

	// Основной цикл стриминга событий
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-session.member.Events():
			if !ok {
				return nil
			}
			// Фильтр дублей: пропускаем события уже вошедшие в снапшот
			if event.Seq <= session.lastSeq {
				continue
			}
			if err := w.Write(ctx, event); err != nil {
				s.logger.Debug("client write error", "err", err, "room", session.room.Name())
				return err
			}
		}
	}
}

/*
// JoinAndStream непосредственно сам жизненный цикл участника чата:
func (s *ChatService) JoinAndStream(ctx context.Context, req JoinRequest, w ChatWriter) error {

	// Защита от спам атак на создание новых и присоединения к существующим комнатам c одного IP
	if err := s.limitGuard.AllowJoin(req.IP); err != nil {
		return err // 429 handler http слое
	}

	// Защита на одновременно открытые сессии на одном IP
	if err := s.connTracker.Acquire(req.IP); err != nil {
		return err // → 429 на уровне handler
	}
	defer s.connTracker.Release(req.IP) // всегда освободим слот при завершении сессии

	// Защита от атак на создание большого кол-ва комнат
	if s.repo.Count() >= s.cfg.MaxGlobalRooms {
		return ErrGlobalCreatedRoomReached
	}

	// Генерация ключей (крипто-провайдер инжектирован в сервис)
	privateSeed, pubKey, err := s.crypto.GenerateKeyPair()
	if err != nil {
		return err
	}
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	// Запоминаем с какого IP сессия получения ключей работает, для защиты последующей для Send
	s.sessionGuard.Register(pubKeyB64, req.IP)

	room, err := s.repo.GetOrCreate(ctx, req.RoomName)
	if err != nil {
		return err
	}

	// Защита от атак на создание большого числа пользователей
	if room.MemberCount() >= s.cfg.MaxMembersPerRoom {
		return ErrGlobalJoinedMemberReached
	}

	// Регистрация участника + получение leave-функции
	member, snapshot, lastSeq, leave, err := room.Join(pubKeyB64)
	if err != nil {
		s.sessionGuard.Unregister(pubKeyB64) // не зарегистрировались — чистим привязку
		return err
	}
	defer func() {
		// Отправляем Left ТОЛЬКО если участник отправлял Join событие
		if member.IsActivated() {
			room.Broadcast(domain.NewLeftEvent(member.ID()))
		}
		leave()
		s.sessionGuard.Unregister(pubKeyB64) // чистим при выходе привязку
	}()

	// Тайм-аут на регистрацию нового клиента, иначе не смог зацепится
	hCtx, cancel := context.WithTimeout(ctx, s.cfg.HandshakeTimeout)
	defer cancel()

	// Отправка bootstrap с тайм-аутом
	if err := w.Write(hCtx, domain.NewBootstrap(member.ID(), privateSeed)); err != nil {
		return err
	}
	privateSeed = nil // обнуляем ссылку — приватный ключ больше не нужен

	// Отправка истории (снапшот) с тайм-аутом
	if err := sendSnapshot(hCtx, snapshot, w); err != nil {
		return err
	}

	// Регистрация завершилась успешно - можно отправлять всем сообщение Join
	member.SetActivated()
	room.Broadcast(domain.NewJoinEvent(member.ID()))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-member.Events():
			if !ok {
				return nil
			}
			// Фильтр дублей: игнорируем события, уже бывшие в снапшоте
			if evt.Seq <= lastSeq {
				continue
			}
			if err := w.Write(ctx, evt); err != nil {
				s.logger.Debug("client write error", "err", err, "room", room.Name())
				return err
			}
		}
	}
}
*/
// sendSnapshot отправляет срез событий через StreamWriter
func sendSnapshot(ctx context.Context, events []domain.Event, w ChatWriter) error {
	for _, evt := range events {
		if err := w.Write(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}
