package user

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

// User is the domain model for a registered account.
type User struct {
	ID                  pgtype.UUID
	Username            string
	Email               string
	PasswordHash        string
	Role                auth.Role
	FavoriteTrackIDs    []pgtype.UUID
	FavoriteStreamIDs   []pgtype.UUID
	FavoritePlaylistIDs []pgtype.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
