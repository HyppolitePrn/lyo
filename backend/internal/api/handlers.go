package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/features"
	"github.com/hyppoliteprn/lyo/internal/passwordreset"
	"github.com/hyppoliteprn/lyo/internal/playlist"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/track"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// UserService is the subset of user.Service consumed by the HTTP handlers.
type UserService interface {
	Register(ctx context.Context, username, email, password string) (auth.TokenPair, error)
	Login(ctx context.Context, email, password string) (auth.TokenPair, error)
	GetByID(ctx context.Context, id string) (*user.User, error)

	AddFavoriteTrack(ctx context.Context, userID, trackID string) error
	RemoveFavoriteTrack(ctx context.Context, userID, trackID string) error
	AddFavoriteStream(ctx context.Context, userID, streamID string) error
	RemoveFavoriteStream(ctx context.Context, userID, streamID string) error
	AddFavoritePlaylist(ctx context.Context, userID, playlistID string) error
	RemoveFavoritePlaylist(ctx context.Context, userID, playlistID string) error
}

// PlaylistService is the subset of playlist.Service consumed by the HTTP handlers.
type PlaylistService interface {
	ListByOwner(ctx context.Context, ownerID string) ([]playlist.Playlist, error)
	Create(ctx context.Context, ownerID, title, description string, isPublic bool) (*playlist.Playlist, error)
	Get(ctx context.Context, id string) (*playlist.Playlist, error)
	Update(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*playlist.Playlist, error)
	Delete(ctx context.Context, id, requesterID string) error
	AddTrack(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error)
	RemoveTrack(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error)
}

// StreamService is the subset of streaming.Service consumed by the HTTP handlers.
type StreamService interface {
	StartStream(ctx context.Context, broadcasterID, title, description string) (*streaming.Stream, error)
	EndStream(ctx context.Context, id, broadcasterID string) (*streaming.Stream, error)
	GetStream(ctx context.Context, id string) (*streaming.Stream, error)
	ListLiveStreams(ctx context.Context) ([]streaming.Stream, error)
}

// FeatureService is the subset of features.Service consumed by the HTTP handlers.
type FeatureService interface {
	IsEnabled(ctx context.Context, name string) bool
	All(ctx context.Context) ([]features.Flag, error)
	Toggle(ctx context.Context, name string, enabled bool) (*features.Flag, error)
}

// PasswordResetService is the subset of passwordreset.Service consumed by the HTTP handlers.
type PasswordResetService interface {
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, password string) error
}

// TrackService is the subset of track.Service consumed by the HTTP handlers.
type TrackService interface {
	ListTracks(ctx context.Context, broadcasterID string, page, limit int) ([]track.Track, error)
	CreateTrack(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*track.Track, error)
	GetTrack(ctx context.Context, id string) (*track.Track, error)
	DeleteTrack(ctx context.Context, id, requesterID string) error
	PresignUpload(ctx context.Context, broadcasterID, filename string) (uploadURL, key, audioURL string, err error)
}

type GetUserByIDUsecase interface {
	Execute(ctx context.Context, id string) (*user.User, error)
}

// UpdateUserByIDUsecase applies a partial update (username and/or email) to a user.
type UpdateUserByIDUsecase interface {
	Execute(ctx context.Context, id string, username, email *string) (*user.User, error)
}

// DeleteUserByIDUsecase permanently erases an account and everything it owns.
type DeleteUserByIDUsecase interface {
	Execute(ctx context.Context, id string) error
}

// Handlers implements StrictServerInterface. Dependencies are injected feature by feature.
type Handlers struct {
	userSvc          UserService
	authSvc          *auth.Service
	streamSvc        StreamService
	featureSvc       FeatureService
	pwResetSvc       PasswordResetService
	trackSvc         TrackService
	playlistSvc      PlaylistService
	getUserByIDUC    GetUserByIDUsecase
	updateUserByIDUC UpdateUserByIDUsecase
	deleteUserByIDUC DeleteUserByIDUsecase
	incidentSvc      IncidentService
	// metricsSvc is optional: nil means no Prometheus is configured and the
	// supervision endpoint reports incidents without live metrics.
	metricsSvc MetricsQuerier
	logger     *slog.Logger
}

func NewHandlers(
	userSvc UserService,
	authSvc *auth.Service,
	streamSvc StreamService,
	featureSvc FeatureService,
	pwResetSvc PasswordResetService,
	trackSvc TrackService,
	playlistSvc PlaylistService,
	getUserByIDUC GetUserByIDUsecase,
	updateUserByIDUC UpdateUserByIDUsecase,
	deleteUserByIDUC DeleteUserByIDUsecase,
	incidentSvc IncidentService,
	metricsSvc MetricsQuerier,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		userSvc:          userSvc,
		authSvc:          authSvc,
		streamSvc:        streamSvc,
		featureSvc:       featureSvc,
		pwResetSvc:       pwResetSvc,
		trackSvc:         trackSvc,
		playlistSvc:      playlistSvc,
		getUserByIDUC:    getUserByIDUC,
		updateUserByIDUC: updateUserByIDUC,
		deleteUserByIDUC: deleteUserByIDUC,
		incidentSvc:      incidentSvc,
		metricsSvc:       metricsSvc,
		logger:           logger,
	}
}

func streamToAPI(s *streaming.Stream) (Stream, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return Stream{}, fmt.Errorf("invalid stream ID %q: %w", s.ID, err)
	}
	bcID, err := uuid.Parse(s.BroadcasterID)
	if err != nil {
		return Stream{}, fmt.Errorf("invalid broadcaster ID %q: %w", s.BroadcasterID, err)
	}
	st := Stream{
		Id:            openapi_types.UUID(id),
		BroadcasterId: openapi_types.UUID(bcID),
		Title:         s.Title,
		Status:        StreamStatus(s.Status),
		StartedAt:     s.StartedAt,
	}
	if s.Description != "" {
		st.Description = &s.Description
	}
	st.EndedAt = s.EndedAt
	return st, nil
}

func trackToAPI(t *track.Track) (Track, error) {
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return Track{}, fmt.Errorf("invalid track ID %q: %w", t.ID, err)
	}
	bcID, err := uuid.Parse(t.BroadcasterID)
	if err != nil {
		return Track{}, fmt.Errorf("invalid broadcaster ID %q: %w", t.BroadcasterID, err)
	}
	tr := Track{
		Id:              openapi_types.UUID(id),
		BroadcasterId:   openapi_types.UUID(bcID),
		Title:           t.Title,
		AudioUrl:        t.AudioURL,
		DurationSeconds: t.DurationSeconds,
		CreatedAt:       t.CreatedAt,
	}
	if t.Artist != "" {
		tr.Artist = &t.Artist
	}
	return tr, nil
}

func featureFlagToAPI(f *features.Flag) (FeatureFlag, error) {
	id, err := uuid.Parse(f.ID)
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("invalid feature flag ID %q: %w", f.ID, err)
	}
	return FeatureFlag{
		Id:          openapi_types.UUID(id),
		Name:        f.Name,
		Enabled:     f.Enabled,
		Description: f.Description,
		UpdatedAt:   f.UpdatedAt,
	}, nil
}

func userToAPI(u *user.User) User {
	favTracks := make([]openapi_types.UUID, len(u.FavoriteTrackIDs))
	for i, id := range u.FavoriteTrackIDs {
		favTracks[i] = openapi_types.UUID(id.Bytes)
	}
	favStreams := make([]openapi_types.UUID, len(u.FavoriteStreamIDs))
	for i, id := range u.FavoriteStreamIDs {
		favStreams[i] = openapi_types.UUID(id.Bytes)
	}
	favPlaylists := make([]openapi_types.UUID, len(u.FavoritePlaylistIDs))
	for i, id := range u.FavoritePlaylistIDs {
		favPlaylists[i] = openapi_types.UUID(id.Bytes)
	}

	return User{
		Id:                  openapi_types.UUID(u.ID.Bytes),
		Username:            u.Username,
		Email:               openapi_types.Email(u.Email),
		Role:                Role(u.Role),
		FavoriteTrackIds:    favTracks,
		FavoriteStreamIds:   favStreams,
		FavoritePlaylistIds: favPlaylists,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func playlistToAPI(p *playlist.Playlist) (Playlist, error) {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return Playlist{}, fmt.Errorf("invalid playlist ID %q: %w", p.ID, err)
	}
	ownerID, err := uuid.Parse(p.OwnerID)
	if err != nil {
		return Playlist{}, fmt.Errorf("invalid owner ID %q: %w", p.OwnerID, err)
	}
	trackIDs := make([]openapi_types.UUID, 0, len(p.TrackIDs))
	for _, tid := range p.TrackIDs {
		parsed, err := uuid.Parse(tid)
		if err != nil {
			return Playlist{}, fmt.Errorf("invalid track ID %q: %w", tid, err)
		}
		trackIDs = append(trackIDs, openapi_types.UUID(parsed))
	}

	pl := Playlist{
		Id:        openapi_types.UUID(id),
		OwnerId:   openapi_types.UUID(ownerID),
		Title:     p.Title,
		TrackIds:  trackIDs,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.Description != "" {
		pl.Description = &p.Description
	}
	return pl, nil
}

// pgUUIDsToStrings converts scanned uuid[] columns to plain string IDs for service calls.
func pgUUIDsToStrings(ids []pgtype.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = uuid.UUID(id.Bytes).String()
	}
	return out
}

// hydrateTracks looks up each ID individually via trackSvc, silently skipping any
// that no longer exist (e.g. a favorited/playlisted track that was later deleted).
func (h *Handlers) hydrateTracks(ctx context.Context, ids []string) ([]Track, error) {
	tracks := make([]Track, 0, len(ids))
	for _, id := range ids {
		t, err := h.trackSvc.GetTrack(ctx, id)
		if err != nil {
			if errors.Is(err, track.ErrNotFound) {
				continue
			}
			return nil, err
		}
		apiTrack, err := trackToAPI(t)
		if err != nil {
			return nil, fmt.Errorf("marshal track: %w", err)
		}
		tracks = append(tracks, apiTrack)
	}
	return tracks, nil
}

func (h *Handlers) hydrateStreams(ctx context.Context, ids []string) ([]Stream, error) {
	streams := make([]Stream, 0, len(ids))
	for _, id := range ids {
		s, err := h.streamSvc.GetStream(ctx, id)
		if err != nil {
			if errors.Is(err, streaming.ErrNotFound) {
				continue
			}
			return nil, err
		}
		apiStream, err := streamToAPI(s)
		if err != nil {
			return nil, fmt.Errorf("marshal stream: %w", err)
		}
		streams = append(streams, apiStream)
	}
	return streams, nil
}

func (h *Handlers) hydratePlaylists(ctx context.Context, ids []string) ([]Playlist, error) {
	playlists := make([]Playlist, 0, len(ids))
	for _, id := range ids {
		p, err := h.playlistSvc.Get(ctx, id)
		if err != nil {
			if errors.Is(err, playlist.ErrNotFound) {
				continue
			}
			return nil, err
		}
		apiPlaylist, err := playlistToAPI(p)
		if err != nil {
			return nil, fmt.Errorf("marshal playlist: %w", err)
		}
		playlists = append(playlists, apiPlaylist)
	}
	return playlists, nil
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

func (h *Handlers) RefreshToken(ctx context.Context, req RefreshTokenRequestObject) (RefreshTokenResponseObject, error) {
	claims, err := h.authSvc.Verify(req.Body.RefreshToken)
	if err != nil {
		return RefreshToken401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{
				Code:    401,
				Message: "invalid or expired refresh token",
			},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// The role is re-read from the database, never carried over from the token
	// being exchanged. Minting the new pair from claims.Role would make a role
	// self-perpetuating: an account demoted out of admin — or deleted — would
	// keep renewing admin tokens from its last refresh token, indefinitely.
	u, err := h.userSvc.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return RefreshToken401JSONResponse{
				UnauthorizedJSONResponse: UnauthorizedJSONResponse{
					Code:    401,
					Message: "invalid or expired refresh token",
				},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "refresh token timeout", "route", "POST /auth/refresh")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	pair, err := h.authSvc.Issue(claims.UserID, u.Role)
	if err != nil {
		return nil, err
	}

	return RefreshToken200JSONResponse(TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}), nil
}

func (h *Handlers) ForgotPassword(ctx context.Context, req ForgotPasswordRequestObject) (ForgotPasswordResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := h.pwResetSvc.ForgotPassword(ctx, string(req.Body.Email)); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "forgot password timeout", "route", "POST /auth/forgot-password")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		h.logger.ErrorContext(ctx, "forgot password failed", "err", err)
	}

	// Always 200: whether the email exists must not be observable.
	return ForgotPassword200JSONResponse{Message: "if that email exists, a reset link has been sent"}, nil
}

func (h *Handlers) ResetPassword(ctx context.Context, req ResetPasswordRequestObject) (ResetPasswordResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := h.pwResetSvc.ResetPassword(ctx, req.Body.Token, req.Body.Password); err != nil {
		if errors.Is(err, passwordreset.ErrInvalidToken) {
			return ResetPassword400JSONResponse{
				BadRequestJSONResponse: BadRequestJSONResponse{Code: 400, Message: "invalid or expired token"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "reset password timeout", "route", "POST /auth/reset-password")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return ResetPassword200JSONResponse{Message: "password updated"}, nil
}

func (h *Handlers) GetUserByID(ctx context.Context, _ GetUserByIDRequestObject) (GetUserByIDResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return GetUserByID401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: "authentication required"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	u, err := h.getUserByIDUC.Execute(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return GetUserByID401JSONResponse{
				UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: "authentication required"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "get user by id timeout", "route", "GET /users/me")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return GetUserByID200JSONResponse(userToAPI(u)), nil
}

func (h *Handlers) UpdateUserByID(ctx context.Context, req UpdateUserByIDRequestObject) (UpdateUserByIDResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return UpdateUserByID401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: "authentication required"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var email *string
	if req.Body.Email != nil {
		e := string(*req.Body.Email)
		email = &e
	}

	u, err := h.updateUserByIDUC.Execute(ctx, claims.UserID, req.Body.Username, email)
	if err != nil {
		if isUniqueViolation(err) {
			return UpdateUserByID422JSONResponse{
				UnprocessableEntityJSONResponse: UnprocessableEntityJSONResponse{
					Code:    422,
					Message: "email or username already taken",
				},
			}, nil
		}
		if errors.Is(err, user.ErrNotFound) {
			return UpdateUserByID401JSONResponse{
				UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: "authentication required"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "update user by id timeout", "route", "PATCH /users/me")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return UpdateUserByID200JSONResponse(userToAPI(u)), nil
}

func (h *Handlers) DeleteUserByID(ctx context.Context, _ DeleteUserByIDRequestObject) (DeleteUserByIDResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "account_deletion") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "account deletion is disabled"}
	}

	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return DeleteUserByID401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: "authentication required"},
		}, nil
	}

	// Longer than the 5s the other /users/me routes take: this one cascades
	// across five tables and deletes however many audio objects the account
	// uploaded, so it is a complex query plus external calls, not a lookup.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// claims.UserID, never a path or body parameter: the only account a caller
	// can delete is the one their own token names.
	if err := h.deleteUserByIDUC.Execute(ctx, claims.UserID); err != nil {
		// Already gone. The caller wanted the account not to exist, and it
		// does not — answering 204 keeps a retried delete idempotent.
		if errors.Is(err, user.ErrNotFound) {
			return DeleteUserByID204Response{}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "delete user by id timeout", "route", "DELETE /users/me")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	h.logger.InfoContext(ctx, "account deleted", "route", "DELETE /users/me", "user_id", claims.UserID)
	return DeleteUserByID204Response{}, nil
}

func (h *Handlers) FavoriteTrack(ctx context.Context, req FavoriteTrackRequestObject) (FavoriteTrackResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return FavoriteTrack401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := h.trackSvc.GetTrack(ctx, req.TrackId.String()); err != nil {
		if errors.Is(err, track.ErrNotFound) {
			return FavoriteTrack404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "track not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite track timeout", "route", "POST /users/me/favorites/tracks/{trackId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	if err := h.userSvc.AddFavoriteTrack(ctx, claims.UserID, req.TrackId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite track timeout", "route", "POST /users/me/favorites/tracks/{trackId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return FavoriteTrack204Response{}, nil
}

func (h *Handlers) UnfavoriteTrack(ctx context.Context, req UnfavoriteTrackRequestObject) (UnfavoriteTrackResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return UnfavoriteTrack401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.userSvc.RemoveFavoriteTrack(ctx, claims.UserID, req.TrackId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "unfavorite track timeout", "route", "DELETE /users/me/favorites/tracks/{trackId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return UnfavoriteTrack204Response{}, nil
}

func (h *Handlers) FavoriteStream(ctx context.Context, req FavoriteStreamRequestObject) (FavoriteStreamResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return FavoriteStream401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := h.streamSvc.GetStream(ctx, req.StreamId.String()); err != nil {
		if errors.Is(err, streaming.ErrNotFound) {
			return FavoriteStream404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "stream not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite stream timeout", "route", "POST /users/me/favorites/streams/{streamId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	if err := h.userSvc.AddFavoriteStream(ctx, claims.UserID, req.StreamId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite stream timeout", "route", "POST /users/me/favorites/streams/{streamId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return FavoriteStream204Response{}, nil
}

func (h *Handlers) UnfavoriteStream(ctx context.Context, req UnfavoriteStreamRequestObject) (UnfavoriteStreamResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return UnfavoriteStream401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.userSvc.RemoveFavoriteStream(ctx, claims.UserID, req.StreamId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "unfavorite stream timeout", "route", "DELETE /users/me/favorites/streams/{streamId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return UnfavoriteStream204Response{}, nil
}

func (h *Handlers) FavoritePlaylist(ctx context.Context, req FavoritePlaylistRequestObject) (FavoritePlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return FavoritePlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := h.playlistSvc.Get(ctx, req.PlaylistId.String()); err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return FavoritePlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite playlist timeout", "route", "POST /users/me/favorites/playlists/{playlistId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	if err := h.userSvc.AddFavoritePlaylist(ctx, claims.UserID, req.PlaylistId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "favorite playlist timeout", "route", "POST /users/me/favorites/playlists/{playlistId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return FavoritePlaylist204Response{}, nil
}

func (h *Handlers) UnfavoritePlaylist(ctx context.Context, req UnfavoritePlaylistRequestObject) (UnfavoritePlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return UnfavoritePlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.userSvc.RemoveFavoritePlaylist(ctx, claims.UserID, req.PlaylistId.String()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "unfavorite playlist timeout", "route", "DELETE /users/me/favorites/playlists/{playlistId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	return UnfavoritePlaylist204Response{}, nil
}

func (h *Handlers) ListFavorites(ctx context.Context, _ ListFavoritesRequestObject) (ListFavoritesResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "favorites") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "favorites are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return ListFavorites401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	u, err := h.userSvc.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list favorites timeout", "route", "GET /users/me/favorites")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	tracks, err := h.hydrateTracks(ctx, pgUUIDsToStrings(u.FavoriteTrackIDs))
	if err != nil {
		return nil, err
	}
	streams, err := h.hydrateStreams(ctx, pgUUIDsToStrings(u.FavoriteStreamIDs))
	if err != nil {
		return nil, err
	}
	playlists, err := h.hydratePlaylists(ctx, pgUUIDsToStrings(u.FavoritePlaylistIDs))
	if err != nil {
		return nil, err
	}

	return ListFavorites200JSONResponse{Tracks: tracks, Streams: streams, Playlists: playlists}, nil
}

func (h *Handlers) ListTracks(ctx context.Context, req ListTracksRequestObject) (ListTracksResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	broadcasterID := ""
	if req.Params.BroadcasterId != nil {
		broadcasterID = req.Params.BroadcasterId.String()
	}
	page, limit := 1, 20
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	tracks, err := h.trackSvc.ListTracks(ctx, broadcasterID, page, limit)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list tracks timeout", "route", "GET /tracks")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	items := make([]Track, 0, len(tracks))
	for i := range tracks {
		item, err := trackToAPI(&tracks[i])
		if err != nil {
			return nil, fmt.Errorf("marshal track: %w", err)
		}
		items = append(items, item)
	}
	return ListTracks200JSONResponse{Items: items}, nil
}

func (h *Handlers) CreateTrack(ctx context.Context, req CreateTrackRequestObject) (CreateTrackResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleBroadcaster) {
		return CreateTrack403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: "forbidden"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	artist := ""
	if req.Body.Artist != nil {
		artist = *req.Body.Artist
	}
	duration := 0
	if req.Body.DurationSeconds != nil {
		duration = *req.Body.DurationSeconds
	}

	t, err := h.trackSvc.CreateTrack(ctx, claims.UserID, req.Body.Title, artist, req.Body.AudioUrl, duration)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "create track timeout", "route", "POST /tracks")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiTrack, err := trackToAPI(t)
	if err != nil {
		return nil, fmt.Errorf("marshal track: %w", err)
	}
	return CreateTrack201JSONResponse(apiTrack), nil
}

func (h *Handlers) GetTrack(ctx context.Context, req GetTrackRequestObject) (GetTrackResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t, err := h.trackSvc.GetTrack(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, track.ErrNotFound) {
			return GetTrack404JSONResponse{
				NotFoundJSONResponse: NotFoundJSONResponse{Code: 404, Message: "track not found"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "get track timeout", "route", "GET /tracks/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiTrack, err := trackToAPI(t)
	if err != nil {
		return nil, fmt.Errorf("marshal track: %w", err)
	}
	return GetTrack200JSONResponse(apiTrack), nil
}

func (h *Handlers) DeleteTrack(ctx context.Context, req DeleteTrackRequestObject) (DeleteTrackResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleBroadcaster) {
		return DeleteTrack403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: "forbidden"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Admins can delete any track; broadcasters only their own.
	requesterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		requesterID = ""
	}

	err := h.trackSvc.DeleteTrack(ctx, req.Id.String(), requesterID)
	if err != nil {
		if errors.Is(err, track.ErrNotFound) {
			return DeleteTrack404JSONResponse{
				NotFoundJSONResponse: NotFoundJSONResponse{Code: 404, Message: "track not found"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "delete track timeout", "route", "DELETE /tracks/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return DeleteTrack204Response{}, nil
}

func (h *Handlers) CreateTrackUploadURL(ctx context.Context, req CreateTrackUploadURLRequestObject) (CreateTrackUploadURLResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "track_uploads") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "track uploads are disabled"}
	}

	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleBroadcaster) {
		return CreateTrackUploadURL403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: "forbidden"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	uploadURL, key, audioURL, err := h.trackSvc.PresignUpload(ctx, claims.UserID, req.Body.Filename)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "presign upload timeout", "route", "POST /tracks/upload-url")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return CreateTrackUploadURL200JSONResponse{
		UploadUrl: uploadURL,
		Key:       key,
		AudioUrl:  audioURL,
	}, nil
}

func (h *Handlers) ListStreams(ctx context.Context, _ ListStreamsRequestObject) (ListStreamsResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	streams, err := h.streamSvc.ListLiveStreams(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list streams timeout", "route", "GET /streams")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	items := make([]Stream, 0, len(streams))
	for i := range streams {
		item, err := streamToAPI(&streams[i])
		if err != nil {
			return nil, fmt.Errorf("marshal stream: %w", err)
		}
		items = append(items, item)
	}
	return ListStreams200JSONResponse{Items: items}, nil
}

func (h *Handlers) CreateStream(ctx context.Context, req CreateStreamRequestObject) (CreateStreamResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "live_streaming") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "live streaming is disabled"}
	}

	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleBroadcaster) {
		return CreateStream403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: "forbidden"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	desc := ""
	if req.Body.Description != nil {
		desc = *req.Body.Description
	}

	stream, err := h.streamSvc.StartStream(ctx, claims.UserID, req.Body.Title, desc)
	if err != nil {
		if errors.Is(err, streaming.ErrAlreadyLive) {
			return nil, &HTTPError{Code: http.StatusConflict, Msg: "you already have a live stream"}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "create stream timeout", "route", "POST /streams")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiStream, err := streamToAPI(stream)
	if err != nil {
		return nil, fmt.Errorf("marshal stream: %w", err)
	}
	return CreateStream201JSONResponse(apiStream), nil
}

func (h *Handlers) GetStream(ctx context.Context, req GetStreamRequestObject) (GetStreamResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stream, err := h.streamSvc.GetStream(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, streaming.ErrNotFound) {
			return GetStream404JSONResponse{
				NotFoundJSONResponse: NotFoundJSONResponse{Code: 404, Message: "stream not found"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "get stream timeout", "route", "GET /streams/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiStream, err := streamToAPI(stream)
	if err != nil {
		return nil, fmt.Errorf("marshal stream: %w", err)
	}
	return GetStream200JSONResponse(apiStream), nil
}

func (h *Handlers) DeleteStream(ctx context.Context, req DeleteStreamRequestObject) (DeleteStreamResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleBroadcaster) {
		return DeleteStream403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: "forbidden"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Admins can end any stream; broadcasters only their own.
	broadcasterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		broadcasterID = ""
	}

	_, err := h.streamSvc.EndStream(ctx, req.Id.String(), broadcasterID)
	if err != nil {
		if errors.Is(err, streaming.ErrNotFound) {
			return DeleteStream404JSONResponse{
				NotFoundJSONResponse: NotFoundJSONResponse{Code: 404, Message: "stream not found"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "delete stream timeout", "route", "DELETE /streams/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return DeleteStream204Response{}, nil
}

func (h *Handlers) ListPlaylists(ctx context.Context, _ ListPlaylistsRequestObject) (ListPlaylistsResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return ListPlaylists401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	playlists, err := h.playlistSvc.ListByOwner(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list playlists timeout", "route", "GET /playlists")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	items := make([]Playlist, 0, len(playlists))
	for i := range playlists {
		item, err := playlistToAPI(&playlists[i])
		if err != nil {
			return nil, fmt.Errorf("marshal playlist: %w", err)
		}
		items = append(items, item)
	}
	return ListPlaylists200JSONResponse{Items: items}, nil
}

func (h *Handlers) CreatePlaylist(ctx context.Context, req CreatePlaylistRequestObject) (CreatePlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return CreatePlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	desc := ""
	if req.Body.Description != nil {
		desc = *req.Body.Description
	}
	isPublic := false
	if req.Body.IsPublic != nil {
		isPublic = *req.Body.IsPublic
	}

	p, err := h.playlistSvc.Create(ctx, claims.UserID, req.Body.Title, desc, isPublic)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "create playlist timeout", "route", "POST /playlists")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiPlaylist, err := playlistToAPI(p)
	if err != nil {
		return nil, fmt.Errorf("marshal playlist: %w", err)
	}
	return CreatePlaylist201JSONResponse(apiPlaylist), nil
}

func (h *Handlers) GetPlaylist(ctx context.Context, req GetPlaylistRequestObject) (GetPlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	p, err := h.playlistSvc.Get(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return GetPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "get playlist timeout", "route", "GET /playlists/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	// Private playlists are only visible to their owner (or an admin); anonymous
	// or non-owner requests get 404 rather than 403 so existence isn't leaked.
	if !p.IsPublic {
		claims, ok := middleware.ClaimsFromContext(ctx)
		if !ok || (p.OwnerID != claims.UserID && !claims.Role.AtLeast(auth.RoleAdmin)) {
			return GetPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
	}

	apiPlaylist, err := playlistToAPI(p)
	if err != nil {
		return nil, fmt.Errorf("marshal playlist: %w", err)
	}
	tracks, err := h.hydrateTracks(ctx, p.TrackIDs)
	if err != nil {
		return nil, err
	}
	apiPlaylist.Tracks = &tracks
	return GetPlaylist200JSONResponse(apiPlaylist), nil
}

func (h *Handlers) UpdatePlaylist(ctx context.Context, req UpdatePlaylistRequestObject) (UpdatePlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return UpdatePlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	existing, err := h.playlistSvc.Get(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return UpdatePlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "update playlist timeout", "route", "PATCH /playlists/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	if existing.OwnerID != claims.UserID && !claims.Role.AtLeast(auth.RoleAdmin) {
		return UpdatePlaylist403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	requesterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		requesterID = ""
	}

	p, err := h.playlistSvc.Update(ctx, req.Id.String(), requesterID, req.Body.Title, req.Body.Description, req.Body.IsPublic)
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return UpdatePlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "update playlist timeout", "route", "PATCH /playlists/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiPlaylist, err := playlistToAPI(p)
	if err != nil {
		return nil, fmt.Errorf("marshal playlist: %w", err)
	}
	return UpdatePlaylist200JSONResponse(apiPlaylist), nil
}

func (h *Handlers) DeletePlaylist(ctx context.Context, req DeletePlaylistRequestObject) (DeletePlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return DeletePlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	existing, err := h.playlistSvc.Get(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return DeletePlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "delete playlist timeout", "route", "DELETE /playlists/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	if existing.OwnerID != claims.UserID && !claims.Role.AtLeast(auth.RoleAdmin) {
		return DeletePlaylist403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	requesterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		requesterID = ""
	}

	if err := h.playlistSvc.Delete(ctx, req.Id.String(), requesterID); err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return DeletePlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "delete playlist timeout", "route", "DELETE /playlists/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	return DeletePlaylist204Response{}, nil
}

func (h *Handlers) AddTrackToPlaylist(ctx context.Context, req AddTrackToPlaylistRequestObject) (AddTrackToPlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return AddTrackToPlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	existing, err := h.playlistSvc.Get(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return AddTrackToPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "add track to playlist timeout", "route", "POST /playlists/{id}/tracks")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	if existing.OwnerID != claims.UserID && !claims.Role.AtLeast(auth.RoleAdmin) {
		return AddTrackToPlaylist403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	if _, err := h.trackSvc.GetTrack(ctx, req.Body.TrackId.String()); err != nil {
		if errors.Is(err, track.ErrNotFound) {
			return AddTrackToPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "track not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "add track to playlist timeout", "route", "POST /playlists/{id}/tracks")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	requesterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		requesterID = ""
	}

	p, err := h.playlistSvc.AddTrack(ctx, req.Id.String(), requesterID, req.Body.TrackId.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return AddTrackToPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "add track to playlist timeout", "route", "POST /playlists/{id}/tracks")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiPlaylist, err := playlistToAPI(p)
	if err != nil {
		return nil, fmt.Errorf("marshal playlist: %w", err)
	}
	return AddTrackToPlaylist200JSONResponse(apiPlaylist), nil
}

func (h *Handlers) RemoveTrackFromPlaylist(ctx context.Context, req RemoveTrackFromPlaylistRequestObject) (RemoveTrackFromPlaylistResponseObject, error) {
	if !h.featureSvc.IsEnabled(ctx, "playlists") {
		return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "playlists are disabled"}
	}
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return RemoveTrackFromPlaylist401JSONResponse{UnauthorizedJSONResponse{Code: 401, Message: "authentication required"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	existing, err := h.playlistSvc.Get(ctx, req.Id.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return RemoveTrackFromPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "remove track from playlist timeout", "route", "DELETE /playlists/{id}/tracks/{trackId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}
	if existing.OwnerID != claims.UserID && !claims.Role.AtLeast(auth.RoleAdmin) {
		return RemoveTrackFromPlaylist403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	requesterID := claims.UserID
	if claims.Role.AtLeast(auth.RoleAdmin) {
		requesterID = ""
	}

	p, err := h.playlistSvc.RemoveTrack(ctx, req.Id.String(), requesterID, req.TrackId.String())
	if err != nil {
		if errors.Is(err, playlist.ErrNotFound) {
			return RemoveTrackFromPlaylist404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "playlist not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "remove track from playlist timeout", "route", "DELETE /playlists/{id}/tracks/{trackId}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiPlaylist, err := playlistToAPI(p)
	if err != nil {
		return nil, fmt.Errorf("marshal playlist: %w", err)
	}
	return RemoveTrackFromPlaylist200JSONResponse(apiPlaylist), nil
}

// GetPublicFeatureFlags serves the flag states clients need to gate their own
// UI. It is deliberately unauthenticated — the app reads it before login — and
// returns names and states only, never descriptions or timestamps.
func (h *Handlers) GetPublicFeatureFlags(ctx context.Context, _ GetPublicFeatureFlagsRequestObject) (GetPublicFeatureFlagsResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	flags, err := h.featureSvc.All(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list public feature flags timeout", "route", "GET /features")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	states := make(PublicFeatureFlags, len(flags))
	for i := range flags {
		states[flags[i].Name] = flags[i].Enabled
	}
	return GetPublicFeatureFlags200JSONResponse(states), nil
}

func (h *Handlers) ListFeatureFlags(ctx context.Context, _ ListFeatureFlagsRequestObject) (ListFeatureFlagsResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleAdmin) {
		return ListFeatureFlags403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	flags, err := h.featureSvc.All(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list feature flags timeout", "route", "GET /admin/features")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	items := make([]FeatureFlag, len(flags))
	for i := range flags {
		apiFlag, err := featureFlagToAPI(&flags[i])
		if err != nil {
			return nil, fmt.Errorf("marshal feature flag: %w", err)
		}
		items[i] = apiFlag
	}
	return ListFeatureFlags200JSONResponse{Items: items}, nil
}

func (h *Handlers) ToggleFeatureFlag(ctx context.Context, req ToggleFeatureFlagRequestObject) (ToggleFeatureFlagResponseObject, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || !claims.Role.AtLeast(auth.RoleAdmin) {
		return ToggleFeatureFlag403JSONResponse{ForbiddenJSONResponse{Code: 403, Message: "forbidden"}}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	flag, err := h.featureSvc.Toggle(ctx, req.Name, req.Body.Enabled)
	if err != nil {
		if errors.Is(err, features.ErrNotFound) {
			return ToggleFeatureFlag404JSONResponse{NotFoundJSONResponse{Code: 404, Message: "feature flag not found"}}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "toggle feature flag timeout", "route", "PATCH /admin/features/{name}/toggle")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	apiFlag, err := featureFlagToAPI(flag)
	if err != nil {
		return nil, fmt.Errorf("marshal feature flag: %w", err)
	}
	return ToggleFeatureFlag200JSONResponse(apiFlag), nil
}
