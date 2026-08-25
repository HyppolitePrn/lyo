package track

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a track lookup, or an owner-scoped delete, yields no rows.
var ErrNotFound = errors.New("track not found")

const dbQueryTimeout = 5 * time.Second

// Repository defines the persistence operations for tracks.
type Repository interface {
	Create(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*Track, error)
	Get(ctx context.Context, id string) (*Track, error)
	// List returns tracks ordered by newest first. If broadcasterID is non-empty, results are filtered to it.
	List(ctx context.Context, broadcasterID string, page, limit int) ([]Track, error)
	// Delete removes the track and returns it. If requesterID is non-empty, enforces ownership.
	Delete(ctx context.Context, id, requesterID string) (*Track, error)
}

type pgRepo struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) Create(ctx context.Context, broadcasterID, title, artist, audioURL string, durationSeconds int) (*Track, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		INSERT INTO tracks (broadcaster_id, title, artist, audio_url, duration_seconds)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at`

	return scanTrack(r.pool.QueryRow(ctx, q, broadcasterID, title, artist, audioURL, durationSeconds))
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Track, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at
		FROM tracks WHERE id = $1`

	t, err := scanTrack(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *pgRepo) List(ctx context.Context, broadcasterID string, page, limit int) ([]Track, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	offset := (page - 1) * limit

	var (
		q    string
		args []any
	)
	if broadcasterID != "" {
		q = `
			SELECT id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at
			FROM tracks WHERE broadcaster_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []any{broadcasterID, limit, offset}
	} else {
		q = `
			SELECT id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at
			FROM tracks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, *t)
	}
	return tracks, rows.Err()
}

func (r *pgRepo) Delete(ctx context.Context, id, requesterID string) (*Track, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	var (
		q    string
		args []any
	)
	if requesterID != "" {
		q = `
			DELETE FROM tracks WHERE id = $1 AND broadcaster_id = $2
			RETURNING id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at`
		args = []any{id, requesterID}
	} else {
		q = `
			DELETE FROM tracks WHERE id = $1
			RETURNING id, broadcaster_id, title, artist, audio_url, duration_seconds, created_at`
		args = []any{id}
	}

	t, err := scanTrack(r.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTrack(row scanner) (*Track, error) {
	var t Track
	err := row.Scan(&t.ID, &t.BroadcasterID, &t.Title, &t.Artist, &t.AudioURL, &t.DurationSeconds, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
