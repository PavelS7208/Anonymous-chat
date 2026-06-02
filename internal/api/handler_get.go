package api

import (
	"anonymous-chat/internal/service"
	"context"
	"errors"
	"log"
	"net/http"
)

// NewGetHandler возвращает хендлер для GET-запросов (подключение к стриму)
func NewGetHandler(streamer service.ChatStreamer, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomName := r.PathValue("roomName")
		if roomName == "" {
			http.Error(w, "invalid room name", http.StatusBadRequest)
			return
		}

		// Инициализация потокового writer-а (заголовки + chunked)
		sw, err := NewHttpFlushableStreamWriter(w)
		if err != nil {
			logger.Printf("stream init error [%s]: %v", roomName, err)
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Делегируем бизнес-логику сервису
		if err := streamer.StreamChat(r.Context(), sw, roomName); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.Printf("client disconnected [%s]: %v", roomName, err)
				return
			}
			logger.Printf("stream error [%s]: %v", roomName, err)
		}
	}
}
