package handler_test

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/pavel/anonymous-chat/internal/adapters/handler"
)

// newTestLogger создаёт логер, который отбрасывает все сообщения
// Используется в тестах, где не нужно проверять логи
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestLoggerWithCapture создаёт логер, который записывает сообщения в buffer
// Используется, когда нужно ассертить логи
func newTestLoggerWithCapture() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug, // ловим всё, включая Debug
	}))
	return logger, &buf
}

// Настройки для chi для тестов
func newChiPostTestRouter(handler *handler.PostHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/{room_name}", handler.ServeHTTP)
	return r
}

func newChiGetTestRouter(handler *handler.GetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/{room_name}", handler.ServeHTTP)
	return r
}
