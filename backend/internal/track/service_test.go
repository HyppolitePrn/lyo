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

	purged            []track.Track
	purgeErr          error
	purgedBroadcaster string
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
func (m *mockRepo) DeleteByBroadcaster(_ context.Context, broadcasterID string) ([]track.Track, error) {
	m.purgedBroadcaster = broadcasterID
	return m.purged, m.purgeErr
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

// A filename with no extension yields a key with no trailing dot.
func TestPresignUpload_FilenameWithoutExtension(t *testing.T) {
	st := &mockStorage{presignFn: func(context.Context, string) (string, error) { return "https://put", nil }}
	svc := track.NewService(&mockRepo{}, st)

	_, key, _, err := svc.PresignUpload(context.Background(), "bc-1", "nightcall")
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if strings.HasSuffix(key, ".") {
		t.Fatalf("key = %q, want no trailing dot", key)
	}
	if !strings.HasPrefix(key, "tracks/bc-1/") {
		t.Fatalf("key = %q, want it scoped to the broadcaster", key)
	}
}

func TestPresignUpload_PropagatesStorageError(t *testing.T) {
	sentinel := errors.New("s3 down")
	st := &mockStorage{presignFn: func(context.Context, string) (string, error) { return "", sentinel }}
	svc := track.NewService(&mockRepo{}, st)

	if _, _, _, err := svc.PresignUpload(context.Background(), "bc-1", "x.mp3"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

// ── PurgeByBroadcaster ────────────────────────────────────────────────────────

func TestPurgeByBroadcaster_DeletesEveryAudioObject(t *testing.T) {
	repo := &mockRepo{purged: []track.Track{
		{ID: "t-1", AudioURL: "https://bucket.s3.region.amazonaws.com/tracks/b-1/a.mp3"},
		{ID: "t-2", AudioURL: "https://bucket.s3.region.amazonaws.com/tracks/b-1/b.mp3"},
	}}
	var deleted []string
	store := &mockStorage{deleteFn: func(_ context.Context, key string) error {
		deleted = append(deleted, key)
		return nil
	}}

	if err := track.NewService(repo, store).PurgeByBroadcaster(context.Background(), "b-1"); err != nil {
		t.Fatalf("PurgeByBroadcaster: %v", err)
	}

	if repo.purgedBroadcaster != "b-1" {
		t.Fatalf("purged %q, want %q", repo.purgedBroadcaster, "b-1")
	}
	want := []string{"tracks/b-1/a.mp3", "tracks/b-1/b.mp3"}
	if len(deleted) != len(want) || deleted[0] != want[0] || deleted[1] != want[1] {
		t.Fatalf("deleted %v, want %v", deleted, want)
	}
}

// A track whose audio_url points somewhere we do not own is skipped, not
// deleted by key against our own bucket.
func TestPurgeByBroadcaster_SkipsForeignURLs(t *testing.T) {
	repo := &mockRepo{purged: []track.Track{{ID: "t-1", AudioURL: "https://elsewhere.example/x.mp3"}}}
	store := &mockStorage{deleteFn: func(context.Context, string) error {
		t.Fatal("a foreign URL must not be deleted from our bucket")
		return nil
	}}

	if err := track.NewService(repo, store).PurgeByBroadcaster(context.Background(), "b-1"); err != nil {
		t.Fatalf("PurgeByBroadcaster: %v", err)
	}
}

// One unreachable object must not strand the rest: every remaining object is
// still attempted, and the failure is still reported.
func TestPurgeByBroadcaster_ContinuesPastAFailedObject(t *testing.T) {
	repo := &mockRepo{purged: []track.Track{
		{ID: "t-1", AudioURL: "https://bucket.s3.region.amazonaws.com/a.mp3"},
		{ID: "t-2", AudioURL: "https://bucket.s3.region.amazonaws.com/b.mp3"},
		{ID: "t-3", AudioURL: "https://bucket.s3.region.amazonaws.com/c.mp3"},
	}}
	boom := errors.New("s3 down")
	var attempted []string
	store := &mockStorage{deleteFn: func(_ context.Context, key string) error {
		attempted = append(attempted, key)
		if key == "a.mp3" {
			return boom
		}
		return nil
	}}

	err := track.NewService(repo, store).PurgeByBroadcaster(context.Background(), "b-1")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
	if len(attempted) != 3 {
		t.Fatalf("attempted %v, want all three objects tried", attempted)
	}
}

func TestPurgeByBroadcaster_PropagatesRepoError(t *testing.T) {
	boom := errors.New("db down")
	repo := &mockRepo{purgeErr: boom}

	err := track.NewService(repo, &mockStorage{}).PurgeByBroadcaster(context.Background(), "b-1")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
