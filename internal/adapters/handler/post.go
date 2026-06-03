package handler

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/pavel/anonymous-chat/internal/domain"
	"github.com/pavel/anonymous-chat/internal/service"
)

type PostHandler struct {
	sender service.MessageSender
	logger *slog.Logger
}

func NewPostHandler(sender service.MessageSender, logger *slog.Logger) *PostHandler {
	return &PostHandler{
		sender: sender,
		logger: logger,
	}
}

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	roomName := chi.URLParam(r, "room_name")

	h.logger.Info("http post handler",
		"room", roomName,
	)

	if err := domain.ValidateRoomName(roomName); err != nil {
		h.logger.Error("room name is invalid",
			"room", roomName,
			"err", err,
		)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "415 Unsupported Media Type: expected text/plain", http.StatusUnsupportedMediaType)
		return
	}

	// Извлекаем IP (учитываем прокси)
	// middleware.RealIP уже всё сделал — просто читаем
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr // fallback
	}

	// Чтение тела с лимитом (лимит в мидлваре сидит)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("body read error",
			"room", roomName,
			"err", err,
		)
		// Правильная проверка MaxBytesError
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "413 Payload Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "400 Bad Request: failed to read body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		h.logger.Error("empty body",
			"room", roomName,
		)
		http.Error(w, "400 Bad Request: empty body", http.StatusBadRequest)
		return
	}
	if !utf8.Valid(body) {
		h.logger.Error("body is invalid",
			"room", roomName,
		)
		http.Error(w, "400 Bad Request: invalid UTF-8", http.StatusBadRequest)
		return
	}

	// Парсинг протокола (lenient: пропускает множественные пробелы)
	pubKeyB64, sigB64, content, err := ParsePostBody(body)
	if err != nil {
		h.logger.Error("body parse error",
			"room", roomName,
			"err", err,
		)
		http.Error(w, "400 Bad Request: invalid format", http.StatusBadRequest)
		return
	}

	h.logger.Debug("send request",
		"room", roomName,
		"ip", ip,
	)

	req := service.SendRequest{
		RoomName:  roomName,
		IP:        ip,
		PubKeyB64: pubKeyB64,
		SigB64:    sigB64,
		Message:   content,
	}
	// Передача в сервис: там будет вся внутренняя кухня, валидации, лимиты, крипто, броадкаст
	if err := h.sender.Send(r.Context(), req); err != nil {
		h.logger.Error("sender send error",
			"room", roomName,
			"err", err,
		)
		handleSenderError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.logger.Debug("message sent",
		"room", roomName,
		"ip", ip,
	)
}

func handleSenderError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	headers := map[string]string{}

	switch {
	case errors.Is(err, domain.ErrMemberNotFound), errors.Is(err, domain.ErrInvalidSignature):
		status = http.StatusUnauthorized

	case errors.Is(err, service.ErrSessionNotFound),
		errors.Is(err, service.ErrSessionExpired),
		errors.Is(err, service.ErrIPMismatch):
		status = http.StatusForbidden // 403

	case errors.Is(err, service.ErrRoomNotFound):
		status = http.StatusNotFound // 404

	case errors.Is(err, service.ErrPostRateLimited):
		status = http.StatusTooManyRequests // 429
		headers["Retry-After"] = "60"

	case errors.Is(err, service.ErrShuttingDown):
		status = http.StatusServiceUnavailable // 503

	default:
		status = http.StatusInternalServerError // 500
	}

	for k, v := range headers {
		w.Header().Set(k, v)
	}
	http.Error(w, http.StatusText(status), status)
}
