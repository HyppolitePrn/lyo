package passwordreset_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/hyppoliteprn/lyo/internal/passwordreset"
)

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

func TestRepoCreate(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	mock := newMockPool(t)
	mock.ExpectExec("INSERT INTO password_reset_tokens").
		WithArgs("user-1", "hash", expires).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := passwordreset.NewRepository(mock).Create(context.Background(), "user-1", "hash", expires); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestRepoCreate_PropagatesError(t *testing.T) {
	sentinel := errors.New("db down")
	expires := time.Now()
	mock := newMockPool(t)
	mock.ExpectExec("INSERT INTO password_reset_tokens").
		WithArgs("user-1", "hash", expires).
		WillReturnError(sentinel)

	if err := passwordreset.NewRepository(mock).Create(context.Background(), "user-1", "hash", expires); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoGetByHash(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM password_reset_tokens WHERE token_hash = ").
		WithArgs("hash").
		WillReturnRows(mock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "used_at"}).
			AddRow("tok-1", "user-1", "hash", expires, (*time.Time)(nil)))

	got, err := passwordreset.NewRepository(mock).GetByHash(context.Background(), "hash")
	if err != nil {
		t.Fatalf("get by hash: %v", err)
	}
	if got.ID != "tok-1" || got.UserID != "user-1" || got.UsedAt != nil {
		t.Fatalf("unexpected token: %+v", got)
	}
}

func TestRepoGetByHash_NoRowsBecomesErrNotFound(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM password_reset_tokens").WithArgs("hash").WillReturnError(pgx.ErrNoRows)

	if _, err := passwordreset.NewRepository(mock).GetByHash(context.Background(), "hash"); !errors.Is(err, passwordreset.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, passwordreset.ErrNotFound)
	}
}

func TestRepoGetByHash_PropagatesOtherErrors(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectQuery("SELECT (.+) FROM password_reset_tokens").WithArgs("hash").WillReturnError(sentinel)

	if _, err := passwordreset.NewRepository(mock).GetByHash(context.Background(), "hash"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestRepoMarkUsed(t *testing.T) {
	mock := newMockPool(t)
	mock.ExpectExec("UPDATE password_reset_tokens SET used_at").
		WithArgs("tok-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := passwordreset.NewRepository(mock).MarkUsed(context.Background(), "tok-1"); err != nil {
		t.Fatalf("mark used: %v", err)
	}
}

func TestRepoMarkUsed_PropagatesError(t *testing.T) {
	sentinel := errors.New("db down")
	mock := newMockPool(t)
	mock.ExpectExec("UPDATE password_reset_tokens").WithArgs("tok-1").WillReturnError(sentinel)

	if err := passwordreset.NewRepository(mock).MarkUsed(context.Background(), "tok-1"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
