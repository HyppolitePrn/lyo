package playlist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hyppoliteprn/lyo/internal/playlist"
)

// mockRepo is an in-memory Repository stub for unit tests.
type mockRepo struct {
	createFn      func(ctx context.Context, ownerID, title, description string, isPublic bool) (*playlist.Playlist, error)
	getFn         func(ctx context.Context, id string) (*playlist.Playlist, error)
	listByOwnerFn func(ctx context.Context, ownerID string) ([]playlist.Playlist, error)
	updateFn      func(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*playlist.Playlist, error)
	deleteFn      func(ctx context.Context, id, requesterID string) error
	addTrackFn    func(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error)
	removeTrackFn func(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error)
}

func (m *mockRepo) Create(ctx context.Context, ownerID, title, description string, isPublic bool) (*playlist.Playlist, error) {
	return m.createFn(ctx, ownerID, title, description, isPublic)
}
func (m *mockRepo) Get(ctx context.Context, id string) (*playlist.Playlist, error) {
	return m.getFn(ctx, id)
}
func (m *mockRepo) ListByOwner(ctx context.Context, ownerID string) ([]playlist.Playlist, error) {
	return m.listByOwnerFn(ctx, ownerID)
}
func (m *mockRepo) Update(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*playlist.Playlist, error) {
	return m.updateFn(ctx, id, requesterID, title, description, isPublic)
}
func (m *mockRepo) Delete(ctx context.Context, id, requesterID string) error {
	return m.deleteFn(ctx, id, requesterID)
}
func (m *mockRepo) AddTrack(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error) {
	return m.addTrackFn(ctx, id, requesterID, trackID)
}
func (m *mockRepo) RemoveTrack(ctx context.Context, id, requesterID, trackID string) (*playlist.Playlist, error) {
	return m.removeTrackFn(ctx, id, requesterID, trackID)
}

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, ownerID, title, description string, isPublic bool) (*playlist.Playlist, error) {
			return &playlist.Playlist{ID: "p1", OwnerID: ownerID, Title: title, Description: description, IsPublic: isPublic}, nil
		},
	}
	svc := playlist.NewService(repo)

	got, err := svc.Create(context.Background(), "u1", "My Mix", "desc", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "p1" || got.OwnerID != "u1" || !got.IsPublic {
		t.Fatalf("unexpected playlist: %+v", got)
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &mockRepo{
		getFn: func(_ context.Context, _ string) (*playlist.Playlist, error) {
			return nil, playlist.ErrNotFound
		},
	}
	svc := playlist.NewService(repo)

	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, playlist.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAddTrack_PassesOwnershipThrough(t *testing.T) {
	var gotID, gotRequester, gotTrack string
	repo := &mockRepo{
		addTrackFn: func(_ context.Context, id, requesterID, trackID string) (*playlist.Playlist, error) {
			gotID, gotRequester, gotTrack = id, requesterID, trackID
			return &playlist.Playlist{ID: id, TrackIDs: []string{trackID}}, nil
		},
	}
	svc := playlist.NewService(repo)

	got, err := svc.AddTrack(context.Background(), "p1", "u1", "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "p1" || gotRequester != "u1" || gotTrack != "t1" {
		t.Fatalf("unexpected repo call args: id=%s requester=%s track=%s", gotID, gotRequester, gotTrack)
	}
	if len(got.TrackIDs) != 1 || got.TrackIDs[0] != "t1" {
		t.Fatalf("unexpected track_ids: %+v", got.TrackIDs)
	}
}

func TestDelete_PropagatesRepoError(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error {
			return playlist.ErrNotFound
		},
	}
	svc := playlist.NewService(repo)

	err := svc.Delete(context.Background(), "missing", "u1")
	if !errors.Is(err, playlist.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
