package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

// ErrNotFound is returned when a user lookup yields no rows.
var ErrNotFound = errors.New("user not found")

const dbQueryTimeout = 5 * time.Second

// Repository defines the persistence operations for users.
type Repository interface {
	Create(ctx context.Context, username, email, passwordHash string, role auth.Role) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	// Update applies a partial update: nil fields are left untouched.
	Update(ctx context.Context, id string, username, email *string) (*User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error

	AddFavoriteTrack(ctx context.Context, userID, trackID string) error
	RemoveFavoriteTrack(ctx context.Context, userID, trackID string) error
	AddFavoriteStream(ctx context.Context, userID, streamID string) error
	RemoveFavoriteStream(ctx context.Context, userID, streamID string) error
	AddFavoritePlaylist(ctx context.Context, userID, playlistID string) error
	RemoveFavoritePlaylist(ctx context.Context, userID, playlistID string) error
}

type pgRepo struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) Create(ctx context.Context, username, email, passwordHash string, role auth.Role) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, password_hash, role,
		          favorite_track_ids, favorite_stream_ids, favorite_playlist_ids,
		          created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, username, email, passwordHash, string(role))
	return scanUser(row)
}

func (r *pgRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, username, email, password_hash, role,
		       favorite_track_ids, favorite_stream_ids, favorite_playlist_ids,
		       created_at, updated_at
		FROM users WHERE email = $1`

	row := r.pool.QueryRow(ctx, q, email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *pgRepo) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, username, email, password_hash, role,
		       favorite_track_ids, favorite_stream_ids, favorite_playlist_ids,
		       created_at, updated_at
		FROM users WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *pgRepo) Update(ctx context.Context, id string, username, email *string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		UPDATE users
		SET username = COALESCE($2, username),
		    email = COALESCE($3, email),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, username, email, password_hash, role,
		          favorite_track_ids, favorite_stream_ids, favorite_playlist_ids,
		          created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, id, username, email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *pgRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepo) AddFavoriteTrack(ctx context.Context, userID, trackID string) error {
	return r.addFavorite(ctx, "favorite_track_ids", userID, trackID)
}

func (r *pgRepo) RemoveFavoriteTrack(ctx context.Context, userID, trackID string) error {
	return r.removeFavorite(ctx, "favorite_track_ids", userID, trackID)
}

func (r *pgRepo) AddFavoriteStream(ctx context.Context, userID, streamID string) error {
	return r.addFavorite(ctx, "favorite_stream_ids", userID, streamID)
}

func (r *pgRepo) RemoveFavoriteStream(ctx context.Context, userID, streamID string) error {
	return r.removeFavorite(ctx, "favorite_stream_ids", userID, streamID)
}

func (r *pgRepo) AddFavoritePlaylist(ctx context.Context, userID, playlistID string) error {
	return r.addFavorite(ctx, "favorite_playlist_ids", userID, playlistID)
}

func (r *pgRepo) RemoveFavoritePlaylist(ctx context.Context, userID, playlistID string) error {
	return r.removeFavorite(ctx, "favorite_playlist_ids", userID, playlistID)
}

// addFavorite appends targetID to the given favorite_*_ids column, deduping if already present.
// column is a fixed, package-controlled identifier (never user input), so building the query by
// concatenation here is safe from SQL injection.
func (r *pgRepo) addFavorite(ctx context.Context, column, userID, targetID string) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	q := `
		UPDATE users
		SET ` + column + ` = CASE WHEN $2 = ANY(` + column + `) THEN ` + column + ` ELSE array_append(` + column + `, $2) END,
		    updated_at = now()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, userID, targetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepo) removeFavorite(ctx context.Context, column, userID, targetID string) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	q := `
		UPDATE users
		SET ` + column + ` = array_remove(` + column + `, $2), updated_at = now()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, userID, targetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var role string
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&role,
		&u.FavoriteTrackIDs,
		&u.FavoriteStreamIDs,
		&u.FavoritePlaylistIDs,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Role = auth.Role(role)
	return &u, nil
}
