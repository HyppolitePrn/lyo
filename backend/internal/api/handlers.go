package api

import (
	"context"
	"errors"
)

var errNotImplemented = errors.New("not implemented")

// Handlers implements StrictServerInterface. Dependencies are injected feature by feature.
type Handlers struct{}

func NewHandlers() *Handlers { return &Handlers{} }

func (h *Handlers) GetHealth(_ context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h *Handlers) Register(_ context.Context, _ RegisterRequestObject) (RegisterResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) Login(_ context.Context, _ LoginRequestObject) (LoginResponseObject, error) {
	return nil, errNotImplemented
}

func (h *Handlers) RefreshToken(_ context.Context, _ RefreshTokenRequestObject) (RefreshTokenResponseObject, error) {
	return nil, errNotImplemented
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
