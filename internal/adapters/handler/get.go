package handler

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pavel/anonymous-chat/internal/domain"
	"github.com/pavel/anonymous-chat/internal/service"
)

// GetHandler обрабатывает GET /{room_name} — присоединение к комнате.
type GetHandler struct {
	manager service.RoomManager
	logger  *slog.Logger
}

// NewGetHandler создает новый хендлер.
func NewGetHandler(manager service.RoomManager, logger *slog.Logger) *GetHandler {
	return &GetHandler{
		manager: manager,
		logger:  logger,
	}
}

// ServeHTTP реализует http. Handler
// разделено на 2 части: инициализация и стриминг
// причина - после первого write chunked 4xx/5xx ошибки не передашь, только логи
// сначала все проверяем - отдаем ошибки если нашли и только потом стримим
func (h *GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomName := chi.URLParam(r, "room_name")
	h.logger.Info("http get handler",
		"room", roomName,
	)
	if err := domain.ValidateRoomName(roomName); err != nil {
		h.logger.Error("room name is invalid",
			"room", roomName,
			"err", err,
		)
		http.Error(w, "400", http.StatusBadRequest)
		return
	}
	// Извлекаем IP (учитываем прокси)
	// middleware.RealIP уже всё сделал — просто читаем
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr // fallback
	}

	h.logger.Debug("join request",
		"room", roomName,
		"ip", ip,
	)

	req := service.JoinRequest{
		RoomName: roomName,
		IP:       ip,
	}

	// Внутренняя кухня контролей и реализации присоединения в сервисе
	// возвращаем что на создавали
	session, err := h.manager.Join(r.Context(), req)
	if err != nil {
		h.logger.Warn("join rejected",
			"room", roomName,
			"ip", ip,
			"err", err,
		)
		h.writeJoinError(w, err)
		return
	}

	h.logger.Debug("join accepted, opening stream",
		"room", roomName,
		"ip", ip,
	)

	// Создаем наш поток чанков
	sw, err := service.NewChatWriterFromHTTP(w)
	if err != nil {
		h.logger.Error("failed to create stream writer",
			"room", roomName,
			"ip", ip,
			"err", err,
		)
		session.Release()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("stream started",
		"room", roomName,
		"ip", ip,
	)

	// Это сама внутренняя кухня чата в сервисе (broadcast рассылки и получения)
	// передаем туда данные с прошлого этапа проверок и инициализаций
	if err := h.manager.Stream(r.Context(), session, sw); err != nil {
		if isClientDisconnect(err) {
			h.logger.Info("client disconnected",
				"room", roomName,
				"ip", ip,
				"reason", err,
			)
			return
		}
		h.logger.Warn("stream error",
			"room", roomName,
			"ip", ip,
			"err", err,
		)
		return
	}

	// err == nil: клиент вышел штатно (канал событий закрыт)
	h.logger.Debug("stream finished",
		"room", roomName,
		"ip", ip,
	)
}

// isClientDisconnect проверяет, является ли ошибка штатным отключением клиента.
// Используется для фильтрации "шумных" ошибок в стриминговых хендлерах
func isClientDisconnect(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

func (h *GetHandler) writeJoinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrJoinRateLimited),
		errors.Is(err, service.ErrTooManyConnections):
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	case errors.Is(err, service.ErrGlobalCreatedRoomReached),
		errors.Is(err, service.ErrGlobalJoinedMemberReached):
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	default:
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
