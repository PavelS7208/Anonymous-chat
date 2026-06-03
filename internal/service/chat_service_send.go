package service

import (
	"context"

	"github.com/pavel/anonymous-chat/internal/domain"
)

type SendRequest struct {
	RoomName  string
	IP        string
	PubKeyB64 string
	SigB64    string
	Message   []byte
}

// Send обрабатывает входящее сообщение: валидация DTO, крипто, броадкаст.
func (s *ChatService) Send(ctx context.Context, req SendRequest) error {

	// Проверки на сетевые атаки по отправке сообщений
	if err := s.postGuardCheck(ctx, req); err != nil {
		return err
	}

	// Создание и валидация доменного сообщения
	msg, err := domain.NewMessage(req.PubKeyB64, req.SigB64, req.Message)
	if err != nil {
		return err
	}

	// Строгий поиск (комната должна уже существовать)
	room, err := s.repo.Get(ctx, req.RoomName)
	if err != nil {
		return err // вернёт ErrRoomNotFound или ErrShuttingDown
	}

	// Сами бизнес-правила отправки сообщения
	return s.doSend(ctx, room, msg)
}

func (s *ChatService) doSend(_ context.Context, room *domain.Room, msg domain.Message) error {
	// Поиск участника
	member, ok := room.GetMemberByPubKey(msg.SenderPubKeyB64())
	if !ok {
		return domain.ErrMemberNotFound
	}
	// Крипто-верификация
	if err := msg.Verify(s.crypto); err != nil {
		return err
	}
	// Рассылка сообщения всем в комнате
	room.Broadcast(msg.ToEvent(member.ID()))
	return nil
}

// postGuardCheck - pipeline проверок по защите от сетевых атак до отправки сообщения
func (s *ChatService) postGuardCheck(_ context.Context, req SendRequest) error {

	if err := s.limitGuard.AllowPost(req.IP); err != nil {
		s.logger.Warn("Post limit",
			"ip", req.IP,
			"err", err,
		)
		return err // 429 на уровне handler
	}

	// Проверка привязки IP к pubKey.
	// Разрешаем отправлять только с того IP куда отправили ключи шифрования
	if err := s.sessionGuard.Verify(req.PubKeyB64, req.IP); err != nil {
		s.logger.Warn("IP mismatch on send",
			"pub_key", req.PubKeyB64[:8], // только префикс в лог
			"ip", req.IP,
			"err", err,
		)
		return err // 403 на уровне handler
	}

	return nil
}
