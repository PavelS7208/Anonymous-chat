package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type HttpFlushableStreamWriter interface {
	io.Writer
	Flush() error
	WriteAndFlush(string) error
}

type httpFlushableStreamWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewHttpFlushableStreamWriter(w http.ResponseWriter) (HttpFlushableStreamWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	// Заголовки для долгоживущего соединения
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Content-Length не ставим → Go автоматически включит chunked transfer encoding
	w.WriteHeader(http.StatusOK)
	return &httpFlushableStreamWriter{w: w, flusher: flusher}, nil
}

func (h *httpFlushableStreamWriter) Write(p []byte) (int, error) {
	return h.w.Write(p)
}

func (h *httpFlushableStreamWriter) Flush() error {
	h.flusher.Flush()
	return nil
}

// WriteAndFlush — для минимизации кода при записи чанков (пишем + flush сразу)
func (h *httpFlushableStreamWriter) WriteAndFlush(data string) error {
	if _, err := h.w.Write([]byte(data)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := h.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}
