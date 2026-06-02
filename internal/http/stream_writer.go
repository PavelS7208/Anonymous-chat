package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// streamWriter Стример - чтобы не "шуметь" лишними сущностями в коде обработчиков без надобности
type streamWriter struct {
	w       io.Writer
	flusher http.Flusher
}

// Возвращает ошибку, если ResponseWriter не поддерживает Flusher
func newStreamWriter(w http.ResponseWriter) (*streamWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported by server")
	}
	return &streamWriter{
		w:       w,
		flusher: flusher,
	}, nil
}

// send отправляет одно событие в стрим.

func (sw *streamWriter) send(event string) error {
	if _, err := fmt.Fprintf(sw.w, "%s\n", event); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	sw.flusher.Flush()
	return nil
}
