package features

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbQueryTimeout = 5 * time.Second

type pgRepo struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) All(ctx context.Context) ([]Flag, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `SELECT id, name, enabled, description, updated_at FROM feature_flags ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.Name, &f.Enabled, &f.Description, &f.UpdatedAt); err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *pgRepo) IsEnabled(ctx context.Context, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `SELECT enabled FROM feature_flags WHERE name = $1`
	var enabled bool
	err := r.pool.QueryRow(ctx, q, name).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

func (r *pgRepo) Toggle(ctx context.Context, name string, enabled bool) (*Flag, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		UPDATE feature_flags
		SET enabled = $2, updated_at = now()
		WHERE name = $1
		RETURNING id, name, enabled, description, updated_at`
	var f Flag
	err := r.pool.QueryRow(ctx, q, name, enabled).Scan(&f.ID, &f.Name, &f.Enabled, &f.Description, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
