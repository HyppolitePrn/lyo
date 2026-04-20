package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
)

// Handlers implements StrictServerInterface. Dependencies are injected feature by feature.
type Handlers struct {
	userSvc *user.Service
	authSvc *auth.Service
	logger  *slog.Logger
}

func NewHandlers(userSvc *user.Service, authSvc *auth.Service, logger *slog.Logger) *Handlers {
	return &Handlers{userSvc: userSvc, authSvc: authSvc, logger: logger}
}

func (h *Handlers) GetHealth(_ context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h *Handlers) Register(ctx context.Context, req RegisterRequestObject) (RegisterResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pair, err := h.userSvc.Register(ctx, req.Body.Username, string(req.Body.Email), req.Body.Password)
	if err != nil {
		if isUniqueViolation(err) {
			return Register422JSONResponse{
				UnprocessableEntityJSONResponse: UnprocessableEntityJSONResponse{
					Code:    422,
					Message: "email or username already taken",
				},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "register timeout", "route", "POST /auth/register")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return Register201JSONResponse(TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}), nil
}

func (h *Handlers) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pair, err := h.userSvc.Login(ctx, string(req.Body.Email), req.Body.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			return Login401JSONResponse{
				UnauthorizedJSONResponse: UnauthorizedJSONResponse{
					Code:    401,
					Message: "invalid credentials",
				},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "login timeout", "route", "POST /auth/login")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return Login200JSONResponse(TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}), nil
}

func (h *Handlers) RefreshToken(_ context.Context, req RefreshTokenRequestObject) (RefreshTokenResponseObject, error) {
	claims, err := h.authSvc.Verify(req.Body.RefreshToken)
	if err != nil {
		return RefreshToken401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{
				Code:    401,
				Message: "invalid or expired refresh token",
			},
		}, nil
	}

	pair, err := h.authSvc.Issue(claims.UserID, claims.Role)
	if err != nil {
		return nil, err
	}

	return RefreshToken200JSONResponse(TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}), nil
}

func (h *Handlers) GetMe(_ context.Context, _ GetMeRequestObject) (GetMeResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) UpdateMe(_ context.Context, _ UpdateMeRequestObject) (UpdateMeResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) FavoriteTrack(_ context.Context, _ FavoriteTrackRequestObject) (FavoriteTrackResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) UnfavoriteTrack(_ context.Context, _ UnfavoriteTrackRequestObject) (UnfavoriteTrackResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) FavoriteStream(_ context.Context, _ FavoriteStreamRequestObject) (FavoriteStreamResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) UnfavoriteStream(_ context.Context, _ UnfavoriteStreamRequestObject) (UnfavoriteStreamResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) FavoritePlaylist(_ context.Context, _ FavoritePlaylistRequestObject) (FavoritePlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) UnfavoritePlaylist(_ context.Context, _ UnfavoritePlaylistRequestObject) (UnfavoritePlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) ListTracks(_ context.Context, _ ListTracksRequestObject) (ListTracksResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) CreateTrack(_ context.Context, _ CreateTrackRequestObject) (CreateTrackResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) GetTrack(_ context.Context, _ GetTrackRequestObject) (GetTrackResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) DeleteTrack(_ context.Context, _ DeleteTrackRequestObject) (DeleteTrackResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) ListStreams(_ context.Context, _ ListStreamsRequestObject) (ListStreamsResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) CreateStream(_ context.Context, _ CreateStreamRequestObject) (CreateStreamResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) GetStream(_ context.Context, _ GetStreamRequestObject) (GetStreamResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) DeleteStream(_ context.Context, _ DeleteStreamRequestObject) (DeleteStreamResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) ListPlaylists(_ context.Context, _ ListPlaylistsRequestObject) (ListPlaylistsResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) CreatePlaylist(_ context.Context, _ CreatePlaylistRequestObject) (CreatePlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) GetPlaylist(_ context.Context, _ GetPlaylistRequestObject) (GetPlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) UpdatePlaylist(_ context.Context, _ UpdatePlaylistRequestObject) (UpdatePlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) DeletePlaylist(_ context.Context, _ DeletePlaylistRequestObject) (DeletePlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) AddTrackToPlaylist(_ context.Context, _ AddTrackToPlaylistRequestObject) (AddTrackToPlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) RemoveTrackFromPlaylist(_ context.Context, _ RemoveTrackFromPlaylistRequestObject) (RemoveTrackFromPlaylistResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) ListFeatureFlags(_ context.Context, _ ListFeatureFlagsRequestObject) (ListFeatureFlagsResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) ToggleFeatureFlag(_ context.Context, _ ToggleFeatureFlagRequestObject) (ToggleFeatureFlagResponseObject, error) {
	return nil, errNotImplemented
}
