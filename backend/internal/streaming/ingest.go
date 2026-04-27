package streaming

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/coder/websocket"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

const maxChunkBytes = 1 << 20 // 1 MiB per audio chunk

// IngestHandler upgrades broadcaster connections to WebSocket and fans audio
// chunks out through the stream's Hub.
type IngestHandler struct {
	svc    *Service
	auth   *auth.Service
	logger *slog.Logger
}

func NewIngestHandler(svc *Service, authSvc *auth.Service, logger *slog.Logger) *IngestHandler {
	return &IngestHandler{svc: svc, auth: authSvc, logger: logger}
}

func (h *IngestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	ctx := r.Context()

	// Global Authenticate middleware handles Authorization header.
	// Fall back to ?token= query param for clients that cannot set WS headers.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var err error
		claims, err = h.auth.Verify(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if !claims.Role.AtLeast(auth.RoleBroadcaster) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	stream, err := h.svc.GetStream(ctx, streamID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "stream not found", http.StatusNotFound)
			return
		}
		h.logger.ErrorContext(ctx, "get stream for ingest", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if stream.BroadcasterID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if stream.Status != "live" {
		http.Error(w, "stream is not live", http.StatusGone)
		return
	}

	hub := h.svc.Hub(streamID)
	if hub == nil {
		// Stream record exists but Hub was already cleaned up (race with EndStream).
		http.Error(w, "stream hub not available", http.StatusServiceUnavailable)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"audio-ingest"},
	})
	if err != nil {
		// Accept writes the error response itself on failure.
		h.logger.WarnContext(ctx, "websocket upgrade failed",
			slog.String("stream_id", streamID), slog.Any("err", err))
		return
	}
	defer conn.CloseNow() //nolint:errcheck

	conn.SetReadLimit(maxChunkBytes)

	h.logger.Info("broadcaster connected",
		slog.String("stream_id", streamID),
		slog.String("broadcaster_id", claims.UserID))

	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			h.logger.Info("broadcaster disconnected",
				slog.String("stream_id", streamID),
				slog.Any("reason", err))
			return
		}
		if msgType != websocket.MessageBinary {
			continue // ignore text frames
		}
		hub.Broadcast(ctx, Chunk(data))
	}
}
