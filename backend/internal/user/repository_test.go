package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
)

var userColumns = []string{
	"id", "username", "email", "password_hash", "role",
	"favorite_track_ids", "favorite_stream_ids", "favorite_playlist_ids",
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

func userRow(mock pgxmock.PgxPoolIface, id uuid.UUID, role string) *pgxmock.Rows {
	now := time.Now()
	empty := []pgtype.UUID{}
	return mock.NewRows(userColumns).
		AddRow(pgtype.UUID{Bytes: id, Valid: true}, "alice", "alice@example.com", "$2a$hash", role,
			empty, empty, empty, now, now)
}

func TestRepoCreate(t *testing.T) {
	id := uuid.New()
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO users").
		WithArgs("alice", "alice@example.com", "$2a$hash", "user").
		WillReturnRows(userRow(mock, id, "user"))

	got, err := user.NewRepository(mock).Create(context.Background(), "alice", "alice@example.com", "$2a$hash", auth.RoleUser)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Username != "alice" || got.Role != auth.RoleUser {
		t.Fatalf("unexpected user: %+v", got)
	}
	if got.ID.Bytes != id {
		t.Fatalf("id = %v, want %v", got.ID.Bytes, id)
	}
}

func TestRepoCreate_PropagatesError(t *testing.T) {
	sentinel := errors.New("duplicate email")
	mock := newMockPool(t)
	mock.ExpectQuery("INSERT INTO users").
		WithArgs("alice", "alice@example.com", "h", "user").
		WillReturnError(sentinel)

	_, err := user.NewRepository(mock).Create(context.Background(), "alice", "alice@example.com", "h", auth.RoleUser)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoGetByEmail(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE email = ").
		WithArgs("alice@example.com").
		WillReturnRows(userRow(mock, uuid.New(), "broadcaster"))

	got, err := user.NewRepository(mock).GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.Role != auth.RoleBroadcaster {
		t.Fatalf("role = %q", got.Role)
	}
}

func TestRepoGetByEmail_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE email = ").WithArgs("nope@example.com").WillReturnError(pgx.ErrNoRows)

	if _, err := user.NewRepository(mock).GetByEmail(context.Background(), "nope@example.com"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

func TestRepoGetByID(t *testing.T) {
	id := uuid.New()
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = ").
		WithArgs(id.String()).
		WillReturnRows(userRow(mock, id, "admin"))

	got, err := user.NewRepository(mock).GetByID(context.Background(), id.String())
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Role != auth.RoleAdmin {
		t.Fatalf("role = %q", got.Role)
	}
}

func TestRepoGetByID_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = ").WithArgs("nope").WillReturnError(pgx.ErrNoRows)

	if _, err := user.NewRepository(mock).GetByID(context.Background(), "nope"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

// COALESCE means a nil field leaves the column untouched.
func TestRepoUpdate_PartialFields(t *testing.T) {
	id := uuid.New()
	name := "newname"
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE users").
		WithArgs(id.String(), &name, (*string)(nil)).
		WillReturnRows(userRow(mock, id, "user"))

	if _, err := user.NewRepository(mock).Update(context.Background(), id.String(), &name, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestRepoUpdate_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("UPDATE users").
		WithArgs("nope", (*string)(nil), (*string)(nil)).
		WillReturnError(pgx.ErrNoRows)

	if _, err := user.NewRepository(mock).Update(context.Background(), "nope", nil, nil); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

func TestRepoUpdatePassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectExec("UPDATE users SET password_hash").
			WithArgs("user-1", "$2a$new").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if err := user.NewRepository(mock).UpdatePassword(context.Background(), "user-1", "$2a$new"); err != nil {
			t.Fatalf("update password: %v", err)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		mock := newMockPool(t)
		mock.ExpectExec("UPDATE users SET password_hash").
			WithArgs("nope", "$2a$new").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		if err := user.NewRepository(mock).UpdatePassword(context.Background(), "nope", "$2a$new"); !errors.Is(err, user.ErrNotFound) {
			t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
		}
	})

	t.Run("exec fails", func(t *testing.T) {
		sentinel := errors.New("db down")
		mock := newMockPool(t)
		mock.ExpectExec("UPDATE users SET password_hash").WithArgs("user-1", "h").WillReturnError(sentinel)

		if err := user.NewRepository(mock).UpdatePassword(context.Background(), "user-1", "h"); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})
}

// Each favorite kind targets its own column; array_append is guarded so
// favoriting twice is idempotent.
func TestRepoAddFavorite_TargetsTheRightColumn(t *testing.T) {
	tests := []struct {
		name   string
		column string
		call   func(user.Repository) error
	}{
		{"track", "favorite_track_ids", func(r user.Repository) error {
			return r.AddFavoriteTrack(context.Background(), "user-1", "t-1")
		}},
		{"stream", "favorite_stream_ids", func(r user.Repository) error {
			return r.AddFavoriteStream(context.Background(), "user-1", "s-1")
		}},
		{"playlist", "favorite_playlist_ids", func(r user.Repository) error {
			return r.AddFavoritePlaylist(context.Background(), "user-1", "pl-1")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPool(t)
			mock.ExpectExec("UPDATE users(.+)array_append\\("+tt.column).
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			if err := tt.call(user.NewRepository(mock)); err != nil {
				t.Fatalf("add favorite: %v", err)
			}
		})
	}
}

func TestRepoRemoveFavorite_TargetsTheRightColumn(t *testing.T) {
	tests := []struct {
		name   string
		column string
		call   func(user.Repository) error
	}{
		{"track", "favorite_track_ids", func(r user.Repository) error {
			return r.RemoveFavoriteTrack(context.Background(), "user-1", "t-1")
		}},
		{"stream", "favorite_stream_ids", func(r user.Repository) error {
			return r.RemoveFavoriteStream(context.Background(), "user-1", "s-1")
		}},
		{"playlist", "favorite_playlist_ids", func(r user.Repository) error {
			return r.RemoveFavoritePlaylist(context.Background(), "user-1", "pl-1")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPool(t)
			mock.ExpectExec("UPDATE users(.+)array_remove\\("+tt.column).
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			if err := tt.call(user.NewRepository(mock)); err != nil {
				t.Fatalf("remove favorite: %v", err)
			}
		})
	}
}

func TestRepoFavorite_UnknownUser(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectExec("UPDATE users").WithArgs("nope", "t-1").WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	if err := user.NewRepository(mock).AddFavoriteTrack(context.Background(), "nope", "t-1"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

func TestRepoFavorite_PropagatesExecError(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectExec("UPDATE users").WithArgs("user-1", "s-1").WillReturnError(sentinel)

	if err := user.NewRepository(mock).RemoveFavoriteStream(context.Background(), "user-1", "s-1"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
