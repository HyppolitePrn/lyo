package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger logs each HTTP request as a structured JSON line.
// WebSocket upgrade requests bypass the responseRecorder wrapper so that
// http.Hijacker is preserved on the raw ResponseWriter — required by coder/websocket.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				next.ServeHTTP(w, r)
				logger.Info("request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", http.StatusSwitchingProtocols),
					slog.Duration("latency", time.Since(start)),
					slog.String("remote_addr", r.RemoteAddr),
				)
				return
			}

			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("latency", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
