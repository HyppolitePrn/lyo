package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// UserService is the subset of user.Service consumed by the HTTP handlers.
type UserService interface {
	Register(ctx context.Context, username, email, password string) (auth.TokenPair, error)
	Login(ctx context.Context, email, password string) (auth.TokenPair, error)
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
}

type GetUserByIDUsecase interface {
	Execute(ctx context.Context, id string) (*user.User, error)
}

// UpdateUserByIDUsecase applies a partial update (username and/or email) to a user.
type UpdateUserByIDUsecase interface {
	Execute(ctx context.Context, id string, username, email *string) (*user.User, error)
}

// Handlers implements StrictServerInterface. Dependencies are injected feature by feature.
type Handlers struct {
	userSvc          UserService
	authSvc          *auth.Service
	streamSvc        StreamService
	featureSvc       FeatureService
	getUserByIDUC    GetUserByIDUsecase
	updateUserByIDUC UpdateUserByIDUsecase
	logger           *slog.Logger
}

func NewHandlers(
	userSvc UserService,
	authSvc *auth.Service,
	streamSvc StreamService,
	featureSvc FeatureService,
	getUserByIDUC GetUserByIDUsecase,
	updateUserByIDUC UpdateUserByIDUsecase,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		userSvc:          userSvc,
		authSvc:          authSvc,
		streamSvc:        streamSvc,
		featureSvc:       featureSvc,
		getUserByIDUC:    getUserByIDUC,
		updateUserByIDUC: updateUserByIDUC,
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
