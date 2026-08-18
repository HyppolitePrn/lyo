package passwordreset

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a token lookup yields no rows.
var ErrNotFound = errors.New("password reset token not found")

const dbQueryTimeout = 5 * time.Second

// Repository defines the persistence operations for password reset tokens.
type Repository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (*Token, error)
	MarkUsed(ctx context.Context, id string) error
}

type pgRepo struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	_, err := r.pool.Exec(ctx, q, userID, tokenHash, expiresAt)
	return err
}

func (r *pgRepo) GetByHash(ctx context.Context, tokenHash string) (*Token, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, user_id, token_hash, expires_at, used_at
		FROM password_reset_tokens WHERE token_hash = $1`

	var t Token
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgRepo) MarkUsed(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`

	_, err := r.pool.Exec(ctx, q, id)
	return err
}
