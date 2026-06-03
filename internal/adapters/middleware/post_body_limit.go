package middleware

import "net/http"

func PostBodyLimits(maxBodyBytes int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ограничение тела — для POST с сообщением
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxBodyBytes))
			next.ServeHTTP(w, r)
		})
	}
}
