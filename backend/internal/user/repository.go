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
