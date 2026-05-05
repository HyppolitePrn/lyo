package streaming

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// ListenHandler upgrades listener connections to WebSocket and streams audio
// chunks from the stream's Hub to the client.
type ListenHandler struct {
	svc    *Service
	auth   *auth.Service
	logger *slog.Logger
}

func NewListenHandler(svc *Service, authSvc *auth.Service, logger *slog.Logger) *ListenHandler {
	return &ListenHandler{svc: svc, auth: authSvc, logger: logger}
}

func (h *ListenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	ctx := r.Context()

	// Global Authenticate middleware handles Authorization header.
	// Fall back to ?token= query param for clients that cannot set WS headers.
	// If no token is present at all, proceed as anonymous — listening is public.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		token := r.URL.Query().Get("token")
		if token != "" {
			var err error
			claims, err = h.auth.Verify(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		} else {
			claims = &auth.Claims{Role: auth.RoleAnonymous}
		}
	}

	stream, err := h.svc.GetStream(ctx, streamID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "stream not found", http.StatusNotFound)
			return
		}
		h.logger.ErrorContext(ctx, "get stream for listen", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if stream.Status != "live" {
		http.Error(w, "stream is not live", http.StatusGone)
		return
	}

	hub := h.svc.Hub(streamID)
	if hub == nil {
		http.Error(w, "stream hub not available", http.StatusServiceUnavailable)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"audio-stream"},
	})
	if err != nil {
		h.logger.WarnContext(ctx, "websocket upgrade failed",
			slog.String("stream_id", streamID), slog.Any("err", err))
		return
	}
	defer func() {
		if err := conn.CloseNow(); err != nil {
			h.logger.WarnContext(ctx, "websocket close error", slog.Any("err", err))
		}
	}()

	listenerID := uuid.New().String()
	ch := hub.Subscribe(listenerID)
	defer hub.Unsubscribe(listenerID)

	// Derive a context that is cancelled when either the client disconnects
	// or EndStream is called (hub.Done() fires), whichever comes first.
	loopCtx, loopCancel := context.WithCancel(ctx)
	defer loopCancel()
	go func() {
		select {
		case <-hub.Done():
			loopCancel()
		case <-loopCtx.Done():
		}
	}()

	h.logger.Info("listener connected",
		slog.String("stream_id", streamID),
		slog.String("listener_id", listenerID),
		slog.String("user_id", claims.UserID))

	for {
		select {
		case <-loopCtx.Done():
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.Write(loopCtx, websocket.MessageBinary, chunk); err != nil {
				h.logger.Info("listener disconnected",
					slog.String("stream_id", streamID),
					slog.String("listener_id", listenerID),
					slog.Any("reason", err))
				return
			}
		}
	}
}
