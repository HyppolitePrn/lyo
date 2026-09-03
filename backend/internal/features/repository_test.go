package features_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/features"
)

var flagColumns = []string{"id", "name", "enabled", "description", "updated_at"}

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

func TestRepoAll(t *testing.T) {
	now := time.Now()
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM feature_flags ORDER BY name").
		WillReturnRows(mock.NewRows(flagColumns).
			AddRow("1", "favorites", true, "Favoriting", now).
			AddRow("2", "transcoding", false, "ABR", now))

	got, err := features.NewRepository(mock).All(context.Background())
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 2 || got[0].Name != "favorites" || got[1].Enabled {
		t.Fatalf("unexpected flags: %+v", got)
	}
}

func TestRepoAll_Errors(t *testing.T) {
	t.Run("query fails", func(t *testing.T) {
		sentinel := errors.New("db down")
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM feature_flags").WillReturnError(sentinel)

		if _, err := features.NewRepository(mock).All(context.Background()); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})

	t.Run("row scan fails", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM feature_flags").
			WillReturnRows(mock.NewRows(flagColumns).AddRow("1", "favorites", "not-a-bool", "d", time.Now()))

		if _, err := features.NewRepository(mock).All(context.Background()); err == nil {
			t.Fatal("expected a scan error")
		}
	})
}

func TestRepoIsEnabled(t *testing.T) {
	for _, want := range []bool{true, false} {
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT enabled FROM feature_flags WHERE name = ").
			WithArgs("favorites").
			WillReturnRows(mock.NewRows([]string{"enabled"}).AddRow(want))

		got, err := features.NewRepository(mock).IsEnabled(context.Background(), "favorites")
		if err != nil {
			t.Fatalf("is enabled: %v", err)
		}
		if got != want {
			t.Fatalf("is enabled = %v, want %v", got, want)
		}
	}
}

// An unseeded flag is simply off — not an error the caller has to handle.
func TestRepoIsEnabled_UnknownFlagIsFalse(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT enabled FROM feature_flags").WithArgs("never_seeded").WillReturnError(pgx.ErrNoRows)

	got, err := features.NewRepository(mock).IsEnabled(context.Background(), "never_seeded")
	if err != nil {
		t.Fatalf("is enabled: %v", err)
	}
	if got {
		t.Fatal("an unknown flag must be reported as disabled")
	}
}

func TestRepoIsEnabled_PropagatesOtherErrors(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT enabled FROM feature_flags").WithArgs("favorites").WillReturnError(sentinel)

	if _, err := features.NewRepository(mock).IsEnabled(context.Background(), "favorites"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoToggle(t *testing.T) {
	now := time.Now()
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE feature_flags").
		WithArgs("transcoding", true).
		WillReturnRows(mock.NewRows(flagColumns).AddRow("2", "transcoding", true, "ABR", now))

	got, err := features.NewRepository(mock).Toggle(context.Background(), "transcoding", true)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if got.Name != "transcoding" || !got.Enabled {
		t.Fatalf("unexpected flag: %+v", got)
	}
}

// An unknown flag name surfaces as ErrNotFound so the handler can answer 404
// instead of 500.
func TestRepoToggle_UnknownFlagIsNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE feature_flags").WithArgs("nope", true).WillReturnError(pgx.ErrNoRows)

	if _, err := features.NewRepository(mock).Toggle(context.Background(), "nope", true); !errors.Is(err, features.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, features.ErrNotFound)
	}
}
