package track

import (
	"context"
	"fmt"
	"path"
	"regexp"

	"github.com/google/uuid"

	"github.com/hyppoliteprn/lyo/internal/storage"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// invalidExtChars strips anything but alphanumerics from a file extension
// before it becomes part of an S3 key.
var invalidExtChars = regexp.MustCompile(`[^a-zA-Z0-9]`)

// Service contains track business logic: CRUD plus the S3 upload/delete lifecycle.
type Service struct {
	repo    Repository
	storage storage.Storage
}

// NewService creates a Service with the given repository and storage backend.
func NewService(repo Repository, storage storage.Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

// ListTracks returns a page of tracks, optionally filtered by broadcaster.
func (s *Service) ListTracks(ctx context.Context, broadcasterID string, page, limit int) ([]Track, error) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return s.repo.List(ctx, broadcasterID, page, limit)
}

// CreateTrack stores a new track row pointing at an already-uploaded audio file.
func (s *Service) CreateTrack(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*Track, error) {
	return s.repo.Create(ctx, broadcasterID, title, artist, audioURL, durationSeconds)
}

// GetTrack retrieves a single track by ID.
func (s *Service) GetTrack(ctx context.Context, id string) (*Track, error) {
	return s.repo.Get(ctx, id)
}

// DeleteTrack removes the track row and its backing S3 object.
// If requesterID is non-empty, ownership is enforced at the DB level.
func (s *Service) DeleteTrack(ctx context.Context, id, requesterID string) error {
	t, err := s.repo.Delete(ctx, id, requesterID)
	if err != nil {
		return err
	}

	if key, ok := s.storage.KeyFromURL(t.AudioURL); ok {
		if err := s.storage.Delete(ctx, key); err != nil {
			return fmt.Errorf("delete s3 object %q: %w", key, err)
		}
	}
	return nil
}

// PresignUpload generates a fresh S3 object key scoped to the broadcaster and
// returns a presigned PUT URL for it, plus the public URL the caller should
// use as audio_url once the upload completes.
func (s *Service) PresignUpload(ctx context.Context, broadcasterID, filename string) (uploadURL, key, audioURL string, err error) {
	ext := invalidExtChars.ReplaceAllString(path.Ext(filename), "")
	key = fmt.Sprintf("tracks/%s/%s%s", broadcasterID, uuid.NewString(), dotIfNotEmpty(ext))

	uploadURL, err = s.storage.PresignUpload(ctx, key)
	if err != nil {
		return "", "", "", err
	}
	return uploadURL, key, s.storage.PublicURL(key), nil
}

func dotIfNotEmpty(ext string) string {
	if ext == "" {
		return ""
	}
	return "." + ext
}
