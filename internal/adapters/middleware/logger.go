package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			status := ww.Status()
			args := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"bytes", ww.BytesWritten(),
				"dur_ms", time.Since(start).Milliseconds(),
				"ip", r.RemoteAddr,
				"req_id", chimw.GetReqID(r.Context()),
			}

			switch {
			case status >= 500:
				logger.Error("http request", args...)
			case status >= 400:
				logger.Warn("http request", args...)
			default:
				logger.Info("http request", args...)
			}
		})
	}
}
