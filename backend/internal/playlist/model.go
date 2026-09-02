package playlist

import "time"

// Playlist is the domain model for a user-owned, orderable list of tracks.
type Playlist struct {
	ID          string
	OwnerID     string
	Title       string
	Description string
	TrackIDs    []string
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
