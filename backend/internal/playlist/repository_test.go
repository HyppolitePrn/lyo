package playlist_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/playlist"
)

var playlistColumns = []string{"id", "owner_id", "title", "description", "track_ids", "is_public", "created_at", "updated_at"}

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

func playlistRow(mock pgxmock.PgxPoolIface, trackIDs ...string) *pgxmock.Rows {
	now := time.Now()
	return mock.NewRows(playlistColumns).
		AddRow("pl-1", "owner-1", "Road trip", "for the car", trackIDs, true, now, now)
}

func TestRepoCreate(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO playlists").
		WithArgs("owner-1", "Road trip", "for the car", true).
		WillReturnRows(playlistRow(mock))

	got, err := playlist.NewRepository(mock).Create(context.Background(), "owner-1", "Road trip", "for the car", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != "pl-1" || got.OwnerID != "owner-1" || !got.IsPublic {
		t.Fatalf("unexpected playlist: %+v", got)
	}
}

func TestRepoGet(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM playlists WHERE id = ").
		WithArgs("pl-1").
		WillReturnRows(playlistRow(mock, "t-1", "t-2"))

	got, err := playlist.NewRepository(mock).Get(context.Background(), "pl-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.TrackIDs) != 2 {
		t.Fatalf("track ids = %v", got.TrackIDs)
	}
}

func TestRepoGet_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM playlists WHERE id = ").WithArgs("nope").WillReturnError(pgx.ErrNoRows)

	if _, err := playlist.NewRepository(mock).Get(context.Background(), "nope"); !errors.Is(err, playlist.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, playlist.ErrNotFound)
	}
}

func TestRepoListByOwner(t *testing.T) {
	now := time.Now()
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM playlists WHERE owner_id = ").
		WithArgs("owner-1").
		WillReturnRows(mock.NewRows(playlistColumns).
			AddRow("pl-1", "owner-1", "A", "", []string{}, false, now, now).
			AddRow("pl-2", "owner-1", "B", "", []string{"t-1"}, true, now, now))

	got, err := playlist.NewRepository(mock).ListByOwner(context.Background(), "owner-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[1].ID != "pl-2" {
		t.Fatalf("unexpected playlists: %+v", got)
	}
}

func TestRepoListByOwner_Errors(t *testing.T) {
	t.Run("query fails", func(t *testing.T) {
		sentinel := errors.New("db down")
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM playlists WHERE owner_id = ").WithArgs("owner-1").WillReturnError(sentinel)

		if _, err := playlist.NewRepository(mock).ListByOwner(context.Background(), "owner-1"); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})

	t.Run("row scan fails", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("SELECT (.+) FROM playlists WHERE owner_id = ").
			WithArgs("owner-1").
			WillReturnRows(mock.NewRows(playlistColumns).
				AddRow("pl-1", "owner-1", "A", "", []string{}, "not-a-bool", time.Now(), time.Now()))

		if _, err := playlist.NewRepository(mock).ListByOwner(context.Background(), "owner-1"); err == nil {
			t.Fatal("expected a scan error")
		}
	})
}

// An owner's update carries the ownership predicate; an admin's does not.
func TestRepoUpdate_OwnershipScoping(t *testing.T) {
	title := "New title"

	t.Run("owner scoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists(.+)WHERE id = (.+) AND owner_id = ").
			WithArgs("pl-1", "owner-1", &title, (*string)(nil), (*bool)(nil)).
			WillReturnRows(playlistRow(mock))

		if _, err := playlist.NewRepository(mock).Update(context.Background(), "pl-1", "owner-1", &title, nil, nil); err != nil {
			t.Fatalf("update: %v", err)
		}
	})

	t.Run("admin unscoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists").
			WithArgs("pl-1", &title, (*string)(nil), (*bool)(nil)).
			WillReturnRows(playlistRow(mock))

		if _, err := playlist.NewRepository(mock).Update(context.Background(), "pl-1", "", &title, nil, nil); err != nil {
			t.Fatalf("update: %v", err)
		}
	})
}

// Updating a playlist you don't own matches no row.
func TestRepoUpdate_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE playlists").
		WithArgs("pl-1", "someone-else", (*string)(nil), (*string)(nil), (*bool)(nil)).
		WillReturnError(pgx.ErrNoRows)

	_, err := playlist.NewRepository(mock).Update(context.Background(), "pl-1", "someone-else", nil, nil, nil)
	if !errors.Is(err, playlist.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, playlist.ErrNotFound)
	}
}

func TestRepoDelete(t *testing.T) {
	t.Run("owner scoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectExec("DELETE FROM playlists WHERE id = (.+) AND owner_id = ").
			WithArgs("pl-1", "owner-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		if err := playlist.NewRepository(mock).Delete(context.Background(), "pl-1", "owner-1"); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	t.Run("admin unscoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectExec("DELETE FROM playlists WHERE id = ").
			WithArgs("pl-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		if err := playlist.NewRepository(mock).Delete(context.Background(), "pl-1", ""); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	t.Run("no rows affected", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectExec("DELETE FROM playlists").
			WithArgs("pl-1", "owner-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		if err := playlist.NewRepository(mock).Delete(context.Background(), "pl-1", "owner-1"); !errors.Is(err, playlist.ErrNotFound) {
			t.Fatalf("err = %v, want %v", err, playlist.ErrNotFound)
		}
	})

	t.Run("exec fails", func(t *testing.T) {
		sentinel := errors.New("db down")
		mock := newMockPool(t)
		mock.ExpectExec("DELETE FROM playlists").WithArgs("pl-1", "owner-1").WillReturnError(sentinel)

		if err := playlist.NewRepository(mock).Delete(context.Background(), "pl-1", "owner-1"); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})
}

func TestRepoAddTrack(t *testing.T) {
	t.Run("owner scoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists(.+)array_append(.+)WHERE id = (.+) AND owner_id = ").
			WithArgs("pl-1", "t-9", "owner-1").
			WillReturnRows(playlistRow(mock, "t-9"))

		got, err := playlist.NewRepository(mock).AddTrack(context.Background(), "pl-1", "owner-1", "t-9")
		if err != nil {
			t.Fatalf("add track: %v", err)
		}
		if len(got.TrackIDs) != 1 || got.TrackIDs[0] != "t-9" {
			t.Fatalf("track ids = %v", got.TrackIDs)
		}
	})

	t.Run("admin unscoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists(.+)array_append").
			WithArgs("pl-1", "t-9").
			WillReturnRows(playlistRow(mock, "t-9"))

		if _, err := playlist.NewRepository(mock).AddTrack(context.Background(), "pl-1", "", "t-9"); err != nil {
			t.Fatalf("add track: %v", err)
		}
	})

	t.Run("no rows", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists").WithArgs("pl-1", "t-9", "someone-else").WillReturnError(pgx.ErrNoRows)

		_, err := playlist.NewRepository(mock).AddTrack(context.Background(), "pl-1", "someone-else", "t-9")
		if !errors.Is(err, playlist.ErrNotFound) {
			t.Fatalf("err = %v, want %v", err, playlist.ErrNotFound)
		}
	})
}

func TestRepoRemoveTrack(t *testing.T) {
	t.Run("owner scoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists(.+)array_remove(.+)WHERE id = (.+) AND owner_id = ").
			WithArgs("pl-1", "t-9", "owner-1").
			WillReturnRows(playlistRow(mock))

		got, err := playlist.NewRepository(mock).RemoveTrack(context.Background(), "pl-1", "owner-1", "t-9")
		if err != nil {
			t.Fatalf("remove track: %v", err)
		}
		if len(got.TrackIDs) != 0 {
			t.Fatalf("track ids = %v", got.TrackIDs)
		}
	})

	t.Run("admin unscoped", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists(.+)array_remove").
			WithArgs("pl-1", "t-9").
			WillReturnRows(playlistRow(mock))

		if _, err := playlist.NewRepository(mock).RemoveTrack(context.Background(), "pl-1", "", "t-9"); err != nil {
			t.Fatalf("remove track: %v", err)
		}
	})

	t.Run("no rows", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectQuery("UPDATE playlists").WithArgs("pl-1", "t-9", "someone-else").WillReturnError(pgx.ErrNoRows)

		_, err := playlist.NewRepository(mock).RemoveTrack(context.Background(), "pl-1", "someone-else", "t-9")
		if !errors.Is(err, playlist.ErrNotFound) {
			t.Fatalf("err = %v, want %v", err, playlist.ErrNotFound)
		}
	})
}
