package stream

import (
	"context"
	"errors"
	"net/http"
)

// ChunkedStreamer реализует StreamWriter поверх HTTP/1.1 chunked encoding.
type ChunkedStreamer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewChunkedStreamer создает новый стример и настраивает HTTP-заголовки для streaming.
// Автоматически устанавливает:
//   - Content-Type: text/plain; charset=utf-8
//   - Cache-Control: no-cache
// Возвращает ошибку, если ResponseWriter не поддерживает flushing.

func NewChunkedStreamer(w http.ResponseWriter) (*ChunkedStreamer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported: ResponseWriter does not implement http.Flusher")
	}

	// Настраиваем заголовки для streaming (не отправляет ответ, только готовит)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	return &ChunkedStreamer{w: w, flusher: flusher}, nil
}

// Write записывает WireMessage в ответ, вызывая Marshal() для сериализации.
// Автоматически делает Flush после записи для немедленной отправки клиенту.
// Первая успешная записи неявно вызывает WriteHeader(http.StatusOK).
func (s *ChunkedStreamer) Write(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Сериализуем сообщение в байты
	// Пишем в ResponseWriter
	if _, err := s.w.Write(data); err != nil {
		return err
	}

	// Flush для chunked-encoding: отправляем данные клиенту немедленно
	s.flusher.Flush()

	return nil
}
