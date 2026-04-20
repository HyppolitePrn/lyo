package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
)

// mockRepo is an in-memory Repository stub for unit tests.
type mockRepo struct {
	createFn     func(context.Context, string, string, string, auth.Role) (*user.User, error)
	getByEmailFn func(context.Context, string) (*user.User, error)
	getByIDFn    func(context.Context, string) (*user.User, error)
}

func (m *mockRepo) Create(ctx context.Context, username, email, passwordHash string, role auth.Role) (*user.User, error) {
	return m.createFn(ctx, username, email, passwordHash, role)
}
func (m *mockRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.getByEmailFn(ctx, email)
}
func (m *mockRepo) GetByID(ctx context.Context, id string) (*user.User, error) {
	return m.getByIDFn(ctx, id)
}

func newTestAuthSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

// fakeUser returns a User whose PasswordHash matches plainPassword.
func fakeUser(plainPassword string) *user.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.MinCost)
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000001")
	return &user.User{
		ID:           id,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: string(hash),
		Role:         auth.RoleUser,
	}
}

func TestRegister_Success(t *testing.T) {
	u := fakeUser("secret")
	svc := user.NewService(&mockRepo{
		createFn: func(_ context.Context, _, _, _ string, _ auth.Role) (*user.User, error) {
			return u, nil
		},
	}, newTestAuthSvc())

	pair, err := svc.Register(context.Background(), "alice", "alice@example.com", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty token pair")
	}
}

func TestRegister_PropagatesUniqueViolation(t *testing.T) {
	dbErr := &pgconn.PgError{Code: "23505"}
	svc := user.NewService(&mockRepo{
		createFn: func(_ context.Context, _, _, _ string, _ auth.Role) (*user.User, error) {
			return nil, dbErr
		},
	}, newTestAuthSvc())

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "secret")
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "23505" {
		t.Fatalf("expected pgError 23505, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	u := fakeUser("secret")
	svc := user.NewService(&mockRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}, newTestAuthSvc())

	pair, err := svc.Login(context.Background(), "alice@example.com", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty token pair")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	u := fakeUser("secret")
	svc := user.NewService(&mockRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}, newTestAuthSvc())

	_, err := svc.Login(context.Background(), "alice@example.com", "wrongpassword")
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := user.NewService(&mockRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrNotFound
		},
	}, newTestAuthSvc())

	_, err := svc.Login(context.Background(), "nobody@example.com", "secret")
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
