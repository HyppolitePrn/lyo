package incident_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/incident"
)

var incidentColumns = []string{
	"id", "fingerprint", "rule_uid", "title", "summary", "severity", "family", "status",
	"dashboard_url", "started_at", "acknowledged_at", "acknowledged_by", "resolved_at",
	"created_at", "updated_at",
}

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("new mock pool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
		mock.Close()
	})
	return mock
}

func incidentRow(mock pgxmock.PgxPoolIface, extra ...any) *pgxmock.Rows {
	now := time.Now()
	cols := append([]string(nil), incidentColumns...)
	values := []any{
		"11111111-1111-1111-1111-111111111111", "fp-1", "lyo-backend-down",
		"Backend unreachable", "Prometheus cannot scrape", "critical", "technical", "firing",
		"http://grafana", now, nil, nil, nil, now, now,
	}
	if len(extra) > 0 {
		cols = append(cols, "inserted")
		values = append(values, extra...)
	}
	return mock.NewRows(cols).AddRow(values...)
}

// TestOpen_ReportsWhetherTheRowWasInserted pins the signal the notifier keys
// off: only a genuinely new incident may mail the admins.
func TestOpen_ReportsWhetherTheRowWasInserted(t *testing.T) {
	for name, inserted := range map[string]bool{"new incident": true, "re-fire": false} {
		t.Run(name, func(t *testing.T) {
			mock := newMockPool(t)
			mock.ExpectQuery("INSERT INTO incidents").
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnRows(incidentRow(mock, inserted))

			_, created, err := incident.NewRepository(mock).
				Open(t.Context(), incident.Incident{Fingerprint: "fp-1"})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if created != inserted {
				t.Errorf("created = %v, want %v", created, inserted)
			}
		})
	}
}

func TestList_EmptyStatusMeansEverything(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM incidents").
		WithArgs("", 50).
		WillReturnRows(incidentRow(mock))

	got, err := incident.NewRepository(mock).List(t.Context(), "", 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Fingerprint != "fp-1" {
		t.Fatalf("unexpected incidents: %+v", got)
	}
}

// TestAcknowledge_NoMatchingRowIsNotFound: the UPDATE is conditional on the
// incident still firing, so "already resolved" and "does not exist" arrive the
// same way and must both surface as ErrNotFound rather than a nil incident.
func TestAcknowledge_NoMatchingRowIsNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE incidents").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := incident.NewRepository(mock).
		Acknowledge(t.Context(), "11111111-1111-1111-1111-111111111111", "admin-1", time.Now())
	if err != incident.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestResolveByFingerprint_UnknownFingerprintIsNotAnError(t *testing.T) {
	// Grafana sends resolved notifications for alerts this instance never saw
	// fire — after a restart, for instance. That is normal, not a failure.
	mock := newMockPool(t)
	mock.ExpectExec("UPDATE incidents").
		WithArgs("fp-unknown", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	if err := incident.NewRepository(mock).
		ResolveByFingerprint(t.Context(), "fp-unknown", time.Now()); err != nil {
		t.Fatalf("ResolveByFingerprint: %v", err)
	}
}

func TestCountsSince(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT(.+)FROM incidents").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"firing", "acknowledged", "critical", "resolved"}).
			AddRow(2, 1, 1, 5))

	got, err := incident.NewRepository(mock).CountsSince(t.Context(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("CountsSince: %v", err)
	}
	want := incident.Counts{Firing: 2, Acknowledged: 1, CriticalFiring: 1, Resolved24h: 5}
	if got != want {
		t.Errorf("counts = %+v, want %+v", got, want)
	}
}

func TestAdminEmails_OnlyAdmins(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT email FROM users WHERE role = 'admin'").
		WillReturnRows(mock.NewRows([]string{"email"}).
			AddRow("one@lyo.test").AddRow("two@lyo.test"))

	got, err := incident.NewRepository(mock).AdminEmails(t.Context())
	if err != nil {
		t.Fatalf("AdminEmails: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("emails = %v, want 2", got)
	}
}
