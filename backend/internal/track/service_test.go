package track_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hyppoliteprn/lyo/internal/track"
)

// mockRepo is an in-memory Repository stub for unit tests.
type mockRepo struct {
	createFn func(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*track.Track, error)
	getFn    func(ctx context.Context, id string) (*track.Track, error)
	listFn   func(ctx context.Context, broadcasterID string, page, limit int) ([]track.Track, error)
	deleteFn func(ctx context.Context, id, requesterID string) (*track.Track, error)
}

func (m *mockRepo) Create(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*track.Track, error) {
	return m.createFn(ctx, broadcasterID, title, artist, audioURL, durationSeconds)
}
func (m *mockRepo) Get(ctx context.Context, id string) (*track.Track, error) {
	return m.getFn(ctx, id)
}
func (m *mockRepo) List(ctx context.Context, broadcasterID string, page, limit int) ([]track.Track, error) {
	return m.listFn(ctx, broadcasterID, page, limit)
}
func (m *mockRepo) Delete(ctx context.Context, id, requesterID string) (*track.Track, error) {
	return m.deleteFn(ctx, id, requesterID)
}

// mockStorage is an in-memory storage.Storage stub for unit tests.
type mockStorage struct {
	presignFn  func(ctx context.Context, key string) (string, error)
	deleteFn   func(ctx context.Context, key string) error
	deletedKey string
}

func (m *mockStorage) PresignUpload(ctx context.Context, key string) (string, error) {
	return m.presignFn(ctx, key)
}
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.deletedKey = key
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}
func (m *mockStorage) PublicURL(key string) string {
	return "https://bucket.s3.region.amazonaws.com/" + key
}
func (m *mockStorage) KeyFromURL(url string) (string, bool) {
	const prefix = "https://bucket.s3.region.amazonaws.com/"
	if !strings.HasPrefix(url, prefix) {
		return "", false
	}
	return strings.TrimPrefix(url, prefix), true
}

func TestListTracks_ClampsPageAndLimit(t *testing.T) {
	var gotPage, gotLimit int
	repo := &mockRepo{
		listFn: func(_ context.Context, _ string, page, limit int) ([]track.Track, error) {
			gotPage, gotLimit = page, limit
			return nil, nil
		},
	}
	svc := track.NewService(repo, &mockStorage{})

	if _, err := svc.ListTracks(context.Background(), "", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != 1 || gotLimit != 20 {
		t.Fatalf("expected page=1 limit=20, got page=%d limit=%d", gotPage, gotLimit)
	}

	if _, err := svc.ListTracks(context.Background(), "", -5, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != 1 || gotLimit != 20 {
		t.Fatalf("expected clamped page=1 limit=20, got page=%d limit=%d", gotPage, gotLimit)
	}
}

func TestCreateTrack_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*track.Track, error) {
			return &track.Track{
				ID: "t1", BroadcasterID: broadcasterID, Title: title,
				Artist: artist, AudioURL: audioURL, DurationSeconds: durationSeconds,
			}, nil
		},
	}
	svc := track.NewService(repo, &mockStorage{})

	got, err := svc.CreateTrack(context.Background(), "b1", "Title", "Artist", "https://x/y", 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "t1" || got.BroadcasterID != "b1" {
		t.Fatalf("unexpected track: %+v", got)
	}
}

func TestGetTrack_NotFound(t *testing.T) {
	repo := &mockRepo{
		getFn: func(_ context.Context, _ string) (*track.Track, error) {
			return nil, track.ErrNotFound
		},
	}
	svc := track.NewService(repo, &mockStorage{})

	_, err := svc.GetTrack(context.Background(), "missing")
	if !errors.Is(err, track.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteTrack_DeletesS3Object(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, id, requesterID string) (*track.Track, error) {
			return &track.Track{ID: id, BroadcasterID: requesterID, AudioURL: "https://bucket.s3.region.amazonaws.com/tracks/b1/abc.mp3"}, nil
		},
	}
	st := &mockStorage{}
	svc := track.NewService(repo, st)

	if err := svc.DeleteTrack(context.Background(), "t1", "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.deletedKey != "tracks/b1/abc.mp3" {
		t.Fatalf("expected S3 delete for key tracks/b1/abc.mp3, got %q", st.deletedKey)
	}
}

func TestDeleteTrack_PropagatesRepoErrorWithoutTouchingStorage(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) (*track.Track, error) {
			return nil, track.ErrNotFound
		},
	}
	st := &mockStorage{}
	svc := track.NewService(repo, st)

	err := svc.DeleteTrack(context.Background(), "missing", "b1")
	if !errors.Is(err, track.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if st.deletedKey != "" {
		t.Fatalf("expected no S3 delete attempt, got key %q", st.deletedKey)
	}
}

func TestDeleteTrack_SkipsStorageForNonS3URL(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, id, requesterID string) (*track.Track, error) {
			return &track.Track{ID: id, BroadcasterID: requesterID, AudioURL: "https://example.com/not-our-bucket.mp3"}, nil
		},
	}
	st := &mockStorage{}
	svc := track.NewService(repo, st)

	if err := svc.DeleteTrack(context.Background(), "t1", "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.deletedKey != "" {
		t.Fatalf("expected no S3 delete for foreign URL, got key %q", st.deletedKey)
	}
}

func TestPresignUpload_KeyIsScopedToBroadcasterAndKeepsExtension(t *testing.T) {
	var presignedKey string
	repo := &mockRepo{}
	st := &mockStorage{
		presignFn: func(_ context.Context, key string) (string, error) {
			presignedKey = key
			return "https://bucket.s3.region.amazonaws.com/" + key + "?signed", nil
		},
	}
	svc := track.NewService(repo, st)

	uploadURL, key, audioURL, err := svc.PresignUpload(context.Background(), "b1", "my song.mp3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != presignedKey {
		t.Fatalf("expected returned key %q to match presigned key %q", key, presignedKey)
	}
	if !strings.HasPrefix(key, "tracks/b1/") || !strings.HasSuffix(key, ".mp3") {
		t.Fatalf("expected key scoped to broadcaster with .mp3 extension, got %q", key)
	}
	if !strings.Contains(uploadURL, key) {
		t.Fatalf("expected upload URL to contain key %q, got %q", key, uploadURL)
	}
	if audioURL != "https://bucket.s3.region.amazonaws.com/"+key {
		t.Fatalf("unexpected audio URL: %q", audioURL)
	}
}
