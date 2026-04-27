package streaming

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a stream lookup yields no rows.
var ErrNotFound = errors.New("stream not found")

// ErrAlreadyLive is returned when a broadcaster already has an active stream.
var ErrAlreadyLive = errors.New("broadcaster already has a live stream")

const dbQueryTimeout = 5 * time.Second

// Stream represents a live broadcast session.
type Stream struct {
	ID            string
	BroadcasterID string
	Title         string
	Description   string
	Status        string // "live" | "ended"
	StartedAt     time.Time
	EndedAt       *time.Time
}

// StreamRepository defines the persistence operations for streams.
type StreamRepository interface {
	Create(ctx context.Context, broadcasterID, title, description string) (*Stream, error)
	Get(ctx context.Context, id string) (*Stream, error)
	ListLive(ctx context.Context) ([]Stream, error)
	// End soft-ends the stream. If broadcasterID is non-empty, enforces ownership.
	End(ctx context.Context, id, broadcasterID string) (*Stream, error)
	HasLive(ctx context.Context, broadcasterID string) (bool, error)
}

type pgStreamRepo struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed StreamRepository.
func NewRepository(pool *pgxpool.Pool) StreamRepository {
	return &pgStreamRepo{pool: pool}
}

func (r *pgStreamRepo) Create(ctx context.Context, broadcasterID, title, description string) (*Stream, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		INSERT INTO streams (broadcaster_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id, broadcaster_id, title, description, status, started_at, ended_at`
	return scanStream(r.pool.QueryRow(ctx, q, broadcasterID, title, description))
}

func (r *pgStreamRepo) Get(ctx context.Context, id string) (*Stream, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, broadcaster_id, title, description, status, started_at, ended_at
		FROM streams WHERE id = $1`
	s, err := scanStream(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *pgStreamRepo) ListLive(ctx context.Context) ([]Stream, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT id, broadcaster_id, title, description, status, started_at, ended_at
		FROM streams WHERE status = 'live' ORDER BY started_at DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []Stream
	for rows.Next() {
		var s Stream
		var status string
		if err := rows.Scan(&s.ID, &s.BroadcasterID, &s.Title, &s.Description, &status, &s.StartedAt, &s.EndedAt); err != nil {
			return nil, err
		}
		s.Status = status
		streams = append(streams, s)
	}
	return streams, rows.Err()
}

func (r *pgStreamRepo) End(ctx context.Context, id, broadcasterID string) (*Stream, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	var (
		q    string
		args []any
	)
	if broadcasterID != "" {
		q = `
			UPDATE streams SET status = 'ended', ended_at = now()
			WHERE id = $1 AND broadcaster_id = $2 AND status = 'live'
			RETURNING id, broadcaster_id, title, description, status, started_at, ended_at`
		args = []any{id, broadcasterID}
	} else {
		q = `
			UPDATE streams SET status = 'ended', ended_at = now()
			WHERE id = $1 AND status = 'live'
			RETURNING id, broadcaster_id, title, description, status, started_at, ended_at`
		args = []any{id}
	}
	s, err := scanStream(r.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *pgStreamRepo) HasLive(ctx context.Context, broadcasterID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `SELECT EXISTS(SELECT 1 FROM streams WHERE broadcaster_id = $1 AND status = 'live')`
	var exists bool
	err := r.pool.QueryRow(ctx, q, broadcasterID).Scan(&exists)
	return exists, err
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanStream(row scanner) (*Stream, error) {
	var s Stream
	var status string
	err := row.Scan(&s.ID, &s.BroadcasterID, &s.Title, &s.Description, &status, &s.StartedAt, &s.EndedAt)
	if err != nil {
		return nil, err
	}
	s.Status = status
	return &s, nil
}
