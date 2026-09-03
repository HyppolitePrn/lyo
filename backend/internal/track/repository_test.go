package track_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/track"
)

var trackColumns = []string{"id", "broadcaster_id", "title", "artist", "audio_url", "duration_seconds", "created_at"}

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

func trackRow(mock pgxmock.PgxPoolIface, id string) *pgxmock.Rows {
	return mock.NewRows(trackColumns).AddRow(id, "bc-1", "Nightcall", "Kavinsky", "https://cdn/x.mp3", 267, time.Now())
}

func TestRepoCreate(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO tracks").
		WithArgs("bc-1", "Nightcall", "Kavinsky", "https://cdn/x.mp3", 267).
		WillReturnRows(trackRow(mock, "t-1"))

	got, err := track.NewRepository(mock).Create(context.Background(), "bc-1", "Nightcall", "Kavinsky", "https://cdn/x.mp3", 267)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != "t-1" || got.Title != "Nightcall" || got.DurationSeconds != 267 {
		t.Fatalf("unexpected track: %+v", got)
	}
}

func TestRepoCreate_PropagatesError(t *testing.T) {
	sentinel := errors.New("constraint violation")
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO tracks").
		WithArgs("bc-1", "t", "", "u", 0).
		WillReturnError(sentinel)

	if _, err := track.NewRepository(mock).Create(context.Background(), "bc-1", "t", "", "u", 0); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoGet(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM tracks WHERE id = ").
		WithArgs("t-1").
		WillReturnRows(trackRow(mock, "t-1"))

	got, err := track.NewRepository(mock).Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "t-1" {
		t.Fatalf("id = %q", got.ID)
	}
}

// pgx.ErrNoRows must surface as the package's own sentinel.
func TestRepoGet_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM tracks WHERE id = ").WithArgs("nope").WillReturnError(pgx.ErrNoRows)

	if _, err := track.NewRepository(mock).Get(context.Background(), "nope"); !errors.Is(err, track.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, track.ErrNotFound)
	}
}

func TestRepoList_AllBroadcasters(t *testing.T) {
	mock := newMockPool(t)
	// No broadcaster filter: limit and offset are the only arguments.
	mock.ExpectQuery("SELECT (.+) FROM tracks ORDER BY created_at DESC LIMIT").
		WithArgs(20, 40). // page 3 of 20
		WillReturnRows(mock.NewRows(trackColumns).
			AddRow("t-1", "bc-1", "A", "", "u1", 1, time.Now()).
			AddRow("t-2", "bc-2", "B", "", "u2", 2, time.Now()))

	got, err := track.NewRepository(mock).List(context.Background(), "", 3, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0].ID != "t-1" || got[1].ID != "t-2" {
		t.Fatalf("unexpected tracks: %+v", got)
	}
}

func TestRepoList_FilteredByBroadcaster(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM tracks WHERE broadcaster_id = ").
		WithArgs("bc-1", 10, 0).
		WillReturnRows(mock.NewRows(trackColumns).AddRow("t-1", "bc-1", "A", "", "u1", 1, time.Now()))

	got, err := track.NewRepository(mock).List(context.Background(), "bc-1", 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].BroadcasterID != "bc-1" {
		t.Fatalf("unexpected tracks: %+v", got)
	}
}

func TestRepoList_QueryError(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM tracks").WithArgs(10, 0).WillReturnError(sentinel)

	if _, err := track.NewRepository(mock).List(context.Background(), "", 1, 10); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoList_ScanError(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM tracks").
		WithArgs(10, 0).
		WillReturnRows(mock.NewRows(trackColumns).
			AddRow("t-1", "bc-1", "A", "", "u1", "not-an-int", time.Now()))

	if _, err := track.NewRepository(mock).List(context.Background(), "", 1, 10); err == nil {
		t.Fatal("expected a scan error")
	}
}

// A broadcaster deleting their own track: ownership is enforced in the WHERE clause.
func TestRepoDelete_OwnerScoped(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("DELETE FROM tracks WHERE id = (.+) AND broadcaster_id = ").
		WithArgs("t-1", "bc-1").
		WillReturnRows(trackRow(mock, "t-1"))

	got, err := track.NewRepository(mock).Delete(context.Background(), "t-1", "bc-1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.ID != "t-1" {
		t.Fatalf("id = %q", got.ID)
	}
}

// An admin passes an empty requesterID and the ownership predicate is dropped.
func TestRepoDelete_AdminUnscoped(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("DELETE FROM tracks WHERE id = ").
		WithArgs("t-1").
		WillReturnRows(trackRow(mock, "t-1"))

	if _, err := track.NewRepository(mock).Delete(context.Background(), "t-1", ""); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// Deleting someone else's track matches no row, which must read as "not found".
func TestRepoDelete_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("DELETE FROM tracks").WithArgs("t-1", "bc-2").WillReturnError(pgx.ErrNoRows)

	if _, err := track.NewRepository(mock).Delete(context.Background(), "t-1", "bc-2"); !errors.Is(err, track.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, track.ErrNotFound)
	}
}
