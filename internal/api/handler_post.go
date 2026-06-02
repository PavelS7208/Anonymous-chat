package api

import (
	"anonymous-chat/internal/service"
	"io"
	"log"
	"net/http"
	"strings"
)

// NewPostHandler возвращает хендлер для POST-запросов (отправка сообщения)
func NewPostHandler(poster service.MessagePoster, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomName := r.PathValue("roomName")
		if roomName == "" {
			http.Error(w, "room name is empty", http.StatusBadRequest)
			return
		}

		// Чтение тела с защитой от DoS
		body, err := io.ReadAll(io.LimitReader(r.Body, 4098))
		if err != nil {
			logger.Printf("read body error [%s]: %v", roomName, err)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		// Тело закрываем корректно
		defer func() {
			if closeErr := r.Body.Close(); closeErr != nil {
				logger.Printf("close body error [%s]: %v", roomName, closeErr)
			}
		}()

		// Парсинг формата: "<pubKeyB64> <sigB64> <message>"
		line := strings.TrimSpace(string(body))
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			http.Error(w, "invalid format: expected '<pubKey> <sig> <message>'", http.StatusBadRequest)
			return
		}
		msg := service.MessageData{
			PubKeyB64: parts[0],
			SigB64:    parts[1],
			Message:   parts[2],
		}

		if err := msg.Validate(); err != nil {
			http.Error(w, "invalid message data: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Делегируем всю внутреннюю кухню (валидацию, криптографию и рассылку) сервису
		ctx := r.Context()
		if err := poster.PostMessage(ctx, roomName, msg); err != nil {
			status := httpStatusForError(err)
			logger.Printf("post message error [%s]: %v", roomName, err)
			http.Error(w, err.Error(), status)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
