package streaming_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/streaming"
)

var streamColumns = []string{"id", "broadcaster_id", "title", "description", "status", "started_at", "ended_at"}

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

func liveRow(mock pgxmock.PgxPoolIface) *pgxmock.Rows {
	return mock.NewRows(streamColumns).
		AddRow("s-1", "bc-1", "My show", "desc", "live", time.Now(), (*time.Time)(nil))
}

func TestRepoCreate(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO streams").
		WithArgs("bc-1", "My show", "desc").
		WillReturnRows(liveRow(mock))

	got, err := streaming.NewRepository(mock).Create(context.Background(), "bc-1", "My show", "desc")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != "s-1" || got.Status != "live" || got.EndedAt != nil {
		t.Fatalf("unexpected stream: %+v", got)
	}
}

// A partial unique index guarantees one live stream per broadcaster; the
// resulting 23505 must read as ErrAlreadyLive, not a raw driver error.
func TestRepoCreate_UniqueViolationBecomesErrAlreadyLive(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO streams").
		WithArgs("bc-1", "My show", "").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	_, err := streaming.NewRepository(mock).Create(context.Background(), "bc-1", "My show", "")
	if !errors.Is(err, streaming.ErrAlreadyLive) {
		t.Fatalf("err = %v, want %v", err, streaming.ErrAlreadyLive)
	}
}

func TestRepoCreate_PropagatesOtherErrors(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO streams").WithArgs("bc-1", "t", "").WillReturnError(sentinel)

	if _, err := streaming.NewRepository(mock).Create(context.Background(), "bc-1", "t", ""); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoGet(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM streams WHERE id = ").WithArgs("s-1").WillReturnRows(liveRow(mock))

	got, err := streaming.NewRepository(mock).Get(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BroadcasterID != "bc-1" {
		t.Fatalf("broadcaster = %q", got.BroadcasterID)
	}
}

func TestRepoGet_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM streams WHERE id = ").WithArgs("nope").WillReturnError(pgx.ErrNoRows)

	if _, err := streaming.NewRepository(mock).Get(context.Background(), "nope"); !errors.Is(err, streaming.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, streaming.ErrNotFound)
	}
}

func TestRepoListLive(t *testing.T) {
	now := time.Now()
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM streams WHERE status = 'live'").
		WillReturnRows(mock.NewRows(streamColumns).
			AddRow("s-1", "bc-1", "A", "", "live", now, (*time.Time)(nil)).
			AddRow("s-2", "bc-2", "B", "", "live", now, (*time.Time)(nil)))

	got, err := streaming.NewRepository(mock).ListLive(context.Background())
	if err != nil {
		t.Fatalf("list live: %v", err)
	}
	if len(got) != 2 || got[0].Status != "live" {
		t.Fatalf("unexpected streams: %+v", got)
	}
}

func TestRepoListLive_Errors(t *testing.T) {
	t.Run("query fails", func(t *testing.T) {
		sentinel := errors.New("db down")
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM streams").WillReturnError(sentinel)

		if _, err := streaming.NewRepository(mock).ListLive(context.Background()); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})

	t.Run("row scan fails", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM streams").
			WillReturnRows(mock.NewRows(streamColumns).
				AddRow("s-1", "bc-1", "A", "", "live", "not-a-time", (*time.Time)(nil)))

		if _, err := streaming.NewRepository(mock).ListLive(context.Background()); err == nil {
			t.Fatal("expected a scan error")
		}
	})
}

func TestRepoEnd(t *testing.T) {
	ended := time.Now()
	endedRow := func(mock pgxmock.PgxPoolIface) *pgxmock.Rows {
		return mock.NewRows(streamColumns).AddRow("s-1", "bc-1", "A", "", "ended", ended, &ended)
	}

	t.Run("owner scoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE streams(.+)broadcaster_id = ").
			WithArgs("s-1", "bc-1").
			WillReturnRows(endedRow(mock))

		got, err := streaming.NewRepository(mock).End(context.Background(), "s-1", "bc-1")
		if err != nil {
			t.Fatalf("end: %v", err)
		}
		if got.Status != "ended" || got.EndedAt == nil {
			t.Fatalf("unexpected stream: %+v", got)
		}
	})

	t.Run("admin unscoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE streams").WithArgs("s-1").WillReturnRows(endedRow(mock))

		if _, err := streaming.NewRepository(mock).End(context.Background(), "s-1", ""); err != nil {
			t.Fatalf("end: %v", err)
		}
	})

	// Ending an already-ended stream, or one you don't own, matches no row.
	t.Run("no rows", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE streams").WithArgs("s-1", "bc-2").WillReturnError(pgx.ErrNoRows)

		if _, err := streaming.NewRepository(mock).End(context.Background(), "s-1", "bc-2"); !errors.Is(err, streaming.ErrNotFound) {
			t.Fatalf("err = %v, want %v", err, streaming.ErrNotFound)
		}
	})
}

func TestRepoHasLive(t *testing.T) {
	for _, want := range []bool{true, false} {
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("bc-1").
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(want))

		got, err := streaming.NewRepository(mock).HasLive(context.Background(), "bc-1")
		if err != nil {
			t.Fatalf("has live: %v", err)
		}
		if got != want {
			t.Fatalf("has live = %v, want %v", got, want)
		}
	}
}

func TestRepoHasLive_PropagatesError(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("bc-1").WillReturnError(sentinel)

	if _, err := streaming.NewRepository(mock).HasLive(context.Background(), "bc-1"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
