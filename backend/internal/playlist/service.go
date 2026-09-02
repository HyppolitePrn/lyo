package playlist

import "context"

// Service contains playlist business logic: CRUD plus track membership.
type Service struct {
	repo Repository
}

// NewService creates a Service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListByOwner returns all playlists owned by ownerID, newest first.
func (s *Service) ListByOwner(ctx context.Context, ownerID string) ([]Playlist, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

// Create stores a new playlist.
func (s *Service) Create(ctx context.Context, ownerID, title, description string, isPublic bool) (*Playlist, error) {
	return s.repo.Create(ctx, ownerID, title, description, isPublic)
}

// Get retrieves a single playlist by ID, regardless of ownership or visibility.
// Callers are responsible for enforcing visibility rules.
func (s *Service) Get(ctx context.Context, id string) (*Playlist, error) {
	return s.repo.Get(ctx, id)
}

// Update applies a partial update. If requesterID is non-empty, ownership is enforced at the DB level.
func (s *Service) Update(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*Playlist, error) {
	return s.repo.Update(ctx, id, requesterID, title, description, isPublic)
}

// Delete removes a playlist. If requesterID is non-empty, ownership is enforced at the DB level.
func (s *Service) Delete(ctx context.Context, id, requesterID string) error {
	return s.repo.Delete(ctx, id, requesterID)
}

// AddTrack appends a track to the playlist, deduping if already present.
// If requesterID is non-empty, ownership is enforced at the DB level.
func (s *Service) AddTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error) {
	return s.repo.AddTrack(ctx, id, requesterID, trackID)
}

// RemoveTrack removes a track from the playlist.
// If requesterID is non-empty, ownership is enforced at the DB level.
func (s *Service) RemoveTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error) {
	return s.repo.RemoveTrack(ctx, id, requesterID, trackID)
}
