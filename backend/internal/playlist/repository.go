package playlist

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when a playlist lookup, or an owner-scoped mutation, yields no rows.
var ErrNotFound = errors.New("playlist not found")

const dbQueryTimeout = 5 * time.Second

const selectColumns = `id, owner_id, title, description, track_ids, is_public, created_at, updated_at`

// Repository defines the persistence operations for playlists.
type Repository interface {
	Create(ctx context.Context, ownerID, title, description string, isPublic bool) (*Playlist, error)
	Get(ctx context.Context, id string) (*Playlist, error)
	ListByOwner(ctx context.Context, ownerID string) ([]Playlist, error)
	// Update applies a partial update: nil fields are left untouched. If requesterID is non-empty, enforces ownership.
	Update(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*Playlist, error)
	// Delete removes the playlist. If requesterID is non-empty, enforces ownership.
	Delete(ctx context.Context, id, requesterID string) error
	// AddTrack appends trackID to track_ids, deduping if already present. If requesterID is non-empty, enforces ownership.
	AddTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error)
	// RemoveTrack removes trackID from track_ids. If requesterID is non-empty, enforces ownership.
	RemoveTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error)
}

type pgRepo struct {
	pool PgxPool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool PgxPool) Repository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) Create(ctx context.Context, ownerID, title, description string, isPublic bool) (*Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	q := `
		INSERT INTO playlists (owner_id, title, description, is_public)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + selectColumns

	return scanPlaylist(r.pool.QueryRow(ctx, q, ownerID, title, description, isPublic))
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	q := `SELECT ` + selectColumns + ` FROM playlists WHERE id = $1`

	p, err := scanPlaylist(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *pgRepo) ListByOwner(ctx context.Context, ownerID string) ([]Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	q := `SELECT ` + selectColumns + ` FROM playlists WHERE owner_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, *p)
	}
	return playlists, rows.Err()
}

func (r *pgRepo) Update(ctx context.Context, id, requesterID string, title, description *string, isPublic *bool) (*Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	var (
		q    string
		args []any
	)
	if requesterID != "" {
		q = `
			UPDATE playlists
			SET title = COALESCE($3, title),
			    description = COALESCE($4, description),
			    is_public = COALESCE($5, is_public),
			    updated_at = now()
			WHERE id = $1 AND owner_id = $2
			RETURNING ` + selectColumns
		args = []any{id, requesterID, title, description, isPublic}
	} else {
		q = `
			UPDATE playlists
			SET title = COALESCE($2, title),
			    description = COALESCE($3, description),
			    is_public = COALESCE($4, is_public),
			    updated_at = now()
			WHERE id = $1
			RETURNING ` + selectColumns
		args = []any{id, title, description, isPublic}
	}

	p, err := scanPlaylist(r.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *pgRepo) Delete(ctx context.Context, id, requesterID string) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	var (
		q    string
		args []any
	)
	if requesterID != "" {
		q = `DELETE FROM playlists WHERE id = $1 AND owner_id = $2`
		args = []any{id, requesterID}
	} else {
		q = `DELETE FROM playlists WHERE id = $1`
		args = []any{id}
	}

	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepo) AddTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const setClause = `
		SET track_ids = CASE WHEN $2 = ANY(track_ids) THEN track_ids ELSE array_append(track_ids, $2) END,
		    updated_at = now()`

	var (
		q    string
		args []any
	)
	if requesterID != "" {
		q = `UPDATE playlists ` + setClause + ` WHERE id = $1 AND owner_id = $3 RETURNING ` + selectColumns
		args = []any{id, trackID, requesterID}
	} else {
		q = `UPDATE playlists ` + setClause + ` WHERE id = $1 RETURNING ` + selectColumns
		args = []any{id, trackID}
	}

	p, err := scanPlaylist(r.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *pgRepo) RemoveTrack(ctx context.Context, id, requesterID, trackID string) (*Playlist, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const setClause = `SET track_ids = array_remove(track_ids, $2), updated_at = now()`

	var (
		q    string
		args []any
	)
	if requesterID != "" {
		q = `UPDATE playlists ` + setClause + ` WHERE id = $1 AND owner_id = $3 RETURNING ` + selectColumns
		args = []any{id, trackID, requesterID}
	} else {
		q = `UPDATE playlists ` + setClause + ` WHERE id = $1 RETURNING ` + selectColumns
		args = []any{id, trackID}
	}

	p, err := scanPlaylist(r.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanPlaylist(row scanner) (*Playlist, error) {
	var p Playlist
	err := row.Scan(
		&p.ID,
		&p.OwnerID,
		&p.Title,
		&p.Description,
		&p.TrackIDs,
		&p.IsPublic,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// PgxPool is the subset of *pgxpool.Pool the repository needs. Declaring it
// consumer-side keeps the concrete pool out of the package and lets tests
// substitute an in-memory double.
type PgxPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
