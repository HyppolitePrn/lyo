package track

import "time"

// Track is the domain model for an uploaded audio track.
type Track struct {
	ID              string
	BroadcasterID   string
	Title           string
	Artist          string
	AudioURL        string
	DurationSeconds int
	CreatedAt       time.Time
}
