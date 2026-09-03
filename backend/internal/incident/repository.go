package incident

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const dbQueryTimeout = 5 * time.Second

// PgxPool is the subset of *pgxpool.Pool this repository needs. Declaring it
// consumer-side keeps the concrete pool out of the package and lets tests
// substitute a double.
type PgxPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type pgRepo struct {
	pool PgxPool
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(pool PgxPool) Repository {
	return &pgRepo{pool: pool}
}

const incidentColumns = `id, fingerprint, rule_uid, title, summary, severity, family, status,
	dashboard_url, started_at, acknowledged_at, acknowledged_by, resolved_at, created_at, updated_at`

func (r *pgRepo) Open(ctx context.Context, in Incident) (*Incident, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	// The partial unique index covers unresolved rows only, so this upsert
	// updates a still-open incident and inserts a fresh one after a previous
	// occurrence was resolved. xmax = 0 distinguishes the two: it identifies a
	// row this statement inserted rather than updated, which is what decides
	// whether admins get notified again.
	const q = `
		INSERT INTO incidents (fingerprint, rule_uid, title, summary, severity, family, status,
		                       dashboard_url, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'firing', $7, $8)
		ON CONFLICT (fingerprint) WHERE status <> 'resolved'
		DO UPDATE SET summary = EXCLUDED.summary,
		              severity = EXCLUDED.severity,
		              title = EXCLUDED.title,
		              dashboard_url = EXCLUDED.dashboard_url,
		              updated_at = now()
		RETURNING ` + incidentColumns + `, (xmax = 0) AS inserted`

	row := r.pool.QueryRow(ctx, q, in.Fingerprint, in.RuleUID, in.Title, in.Summary,
		string(in.Severity), string(in.Family), in.DashboardURL, in.StartedAt)

	var out Incident
	var created bool
	if err := scanInto(row, &out, &created); err != nil {
		return nil, false, err
	}
	return &out, created, nil
}

func (r *pgRepo) ResolveByFingerprint(ctx context.Context, fingerprint string, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		UPDATE incidents
		SET status = 'resolved', resolved_at = $2, updated_at = now()
		WHERE fingerprint = $1 AND status <> 'resolved'`
	_, err := r.pool.Exec(ctx, q, fingerprint, at)
	return err
}

func (r *pgRepo) List(ctx context.Context, status Status, limit int) ([]Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	// An empty status means "everything", so the same query serves both the
	// open-incidents view and the history view.
	const q = `
		SELECT ` + incidentColumns + `
		FROM incidents
		WHERE ($1 = '' OR status = $1)
		ORDER BY started_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, q, string(status), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Incident
	for rows.Next() {
		var in Incident
		if err := scanInto(rows, &in, nil); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `SELECT ` + incidentColumns + ` FROM incidents WHERE id = $1`
	var in Incident
	if err := scanInto(r.pool.QueryRow(ctx, q, id), &in, nil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &in, nil
}

func (r *pgRepo) Acknowledge(ctx context.Context, id, adminID string, at time.Time) (*Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	// Acknowledging an already-resolved incident is refused by the WHERE
	// clause rather than silently rewriting history.
	const q = `
		UPDATE incidents
		SET status = 'acknowledged', acknowledged_at = $3, acknowledged_by = $2, updated_at = now()
		WHERE id = $1 AND status = 'firing'
		RETURNING ` + incidentColumns
	return r.updateOne(ctx, q, id, adminID, at)
}

func (r *pgRepo) Resolve(ctx context.Context, id string, at time.Time) (*Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		UPDATE incidents
		SET status = 'resolved', resolved_at = $2, updated_at = now()
		WHERE id = $1 AND status <> 'resolved'
		RETURNING ` + incidentColumns
	return r.updateOne(ctx, q, id, at)
}

func (r *pgRepo) updateOne(ctx context.Context, q string, args ...any) (*Incident, error) {
	var in Incident
	if err := scanInto(r.pool.QueryRow(ctx, q, args...), &in, nil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &in, nil
}

func (r *pgRepo) CountsSince(ctx context.Context, since time.Time) (Counts, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `
		SELECT
			count(*) FILTER (WHERE status = 'firing'),
			count(*) FILTER (WHERE status = 'acknowledged'),
			count(*) FILTER (WHERE status = 'firing' AND severity = 'critical'),
			count(*) FILTER (WHERE status = 'resolved' AND resolved_at >= $1)
		FROM incidents`

	var c Counts
	err := r.pool.QueryRow(ctx, q, since).
		Scan(&c.Firing, &c.Acknowledged, &c.CriticalFiring, &c.Resolved24h)
	return c, err
}

func (r *pgRepo) AdminEmails(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	const q = `SELECT email FROM users WHERE role = 'admin' ORDER BY email`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanInto reads one incident row. When created is non-nil the query is
// expected to project the extra "inserted" column.
func scanInto(row scanner, in *Incident, created *bool) error {
	dest := []any{
		&in.ID, &in.Fingerprint, &in.RuleUID, &in.Title, &in.Summary,
		&in.Severity, &in.Family, &in.Status, &in.DashboardURL, &in.StartedAt,
		&in.AcknowledgedAt, &in.AcknowledgedBy, &in.ResolvedAt, &in.CreatedAt, &in.UpdatedAt,
	}
	if created != nil {
		dest = append(dest, created)
	}
	return row.Scan(dest...)
}
