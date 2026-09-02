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
	createFn         func(context.Context, string, string, string, auth.Role) (*user.User, error)
	getByEmailFn     func(context.Context, string) (*user.User, error)
	getByIDFn        func(context.Context, string) (*user.User, error)
	updateFn         func(context.Context, string, *string, *string) (*user.User, error)
	updatePasswordFn func(context.Context, string, string) error
	addFavoriteFn    func(column, userID, targetID string) error
	removeFavoriteFn func(column, userID, targetID string) error
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
func (m *mockRepo) Update(ctx context.Context, id string, username, email *string) (*user.User, error) {
	return m.updateFn(ctx, id, username, email)
}
func (m *mockRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	if m.updatePasswordFn == nil {
		return nil
	}
	return m.updatePasswordFn(ctx, id, passwordHash)
}
func (m *mockRepo) call(fn func(column, userID, targetID string) error, column, userID, targetID string) error {
	if fn == nil {
		return nil
	}
	return fn(column, userID, targetID)
}
func (m *mockRepo) AddFavoriteTrack(_ context.Context, userID, trackID string) error {
	return m.call(m.addFavoriteFn, "track", userID, trackID)
}
func (m *mockRepo) RemoveFavoriteTrack(_ context.Context, userID, trackID string) error {
	return m.call(m.removeFavoriteFn, "track", userID, trackID)
}
func (m *mockRepo) AddFavoriteStream(_ context.Context, userID, streamID string) error {
	return m.call(m.addFavoriteFn, "stream", userID, streamID)
}
func (m *mockRepo) RemoveFavoriteStream(_ context.Context, userID, streamID string) error {
	return m.call(m.removeFavoriteFn, "stream", userID, streamID)
}
func (m *mockRepo) AddFavoritePlaylist(_ context.Context, userID, playlistID string) error {
	return m.call(m.addFavoriteFn, "playlist", userID, playlistID)
}
func (m *mockRepo) RemoveFavoritePlaylist(_ context.Context, userID, playlistID string) error {
	return m.call(m.removeFavoriteFn, "playlist", userID, playlistID)
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

func TestAddFavoriteTrack_DelegatesToRepo(t *testing.T) {
	var gotColumn, gotUser, gotTarget string
	svc := user.NewService(&mockRepo{
		addFavoriteFn: func(column, userID, targetID string) error {
			gotColumn, gotUser, gotTarget = column, userID, targetID
			return nil
		},
	}, newTestAuthSvc())

	if err := svc.AddFavoriteTrack(context.Background(), "u1", "t1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotColumn != "track" || gotUser != "u1" || gotTarget != "t1" {
		t.Fatalf("unexpected repo call: column=%s user=%s target=%s", gotColumn, gotUser, gotTarget)
	}
}

func TestRemoveFavoriteStream_PropagatesRepoError(t *testing.T) {
	svc := user.NewService(&mockRepo{
		removeFavoriteFn: func(string, string, string) error { return user.ErrNotFound },
	}, newTestAuthSvc())

	err := svc.RemoveFavoriteStream(context.Background(), "u1", "s1")
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAddFavoritePlaylist_DelegatesToRepo(t *testing.T) {
	var gotColumn string
	svc := user.NewService(&mockRepo{
		addFavoriteFn: func(column, _, _ string) error {
			gotColumn = column
			return nil
		},
	}, newTestAuthSvc())

	if err := svc.AddFavoritePlaylist(context.Background(), "u1", "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotColumn != "playlist" {
		t.Fatalf("expected playlist column, got %s", gotColumn)
	}
}

func TestGetByID_DelegatesToRepo(t *testing.T) {
	u := fakeUser("secret")
	svc := user.NewService(&mockRepo{
		getByIDFn: func(_ context.Context, id string) (*user.User, error) {
			if id != "00000000-0000-0000-0000-000000000001" {
				t.Fatalf("repo called with %q", id)
			}
			return u, nil
		},
	}, newTestAuthSvc())

	got, err := svc.GetByID(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != "alice" {
		t.Fatalf("username = %q", got.Username)
	}
}

func TestGetByID_PropagatesNotFound(t *testing.T) {
	svc := user.NewService(&mockRepo{
		getByIDFn: func(context.Context, string) (*user.User, error) { return nil, user.ErrNotFound },
	}, newTestAuthSvc())

	if _, err := svc.GetByID(context.Background(), "nope"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

// Each favorite method must reach its own repository call, so a "favorite a
// track" can never touch the streams or playlists array.
func TestFavorites_TargetTheRightKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
		call func(*user.Service) error
	}{
		{"add track", "track", func(s *user.Service) error {
			return s.AddFavoriteTrack(context.Background(), "u-1", "t-1")
		}},
		{"remove track", "track", func(s *user.Service) error {
			return s.RemoveFavoriteTrack(context.Background(), "u-1", "t-1")
		}},
		{"add stream", "stream", func(s *user.Service) error {
			return s.AddFavoriteStream(context.Background(), "u-1", "s-1")
		}},
		{"remove stream", "stream", func(s *user.Service) error {
			return s.RemoveFavoriteStream(context.Background(), "u-1", "s-1")
		}},
		{"add playlist", "playlist", func(s *user.Service) error {
			return s.AddFavoritePlaylist(context.Background(), "u-1", "pl-1")
		}},
		{"remove playlist", "playlist", func(s *user.Service) error {
			return s.RemoveFavoritePlaylist(context.Background(), "u-1", "pl-1")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotKind, gotUser string
			record := func(kind, userID, _ string) error {
				gotKind, gotUser = kind, userID
				return nil
			}
			svc := user.NewService(&mockRepo{addFavoriteFn: record, removeFavoriteFn: record}, newTestAuthSvc())

			if err := tt.call(svc); err != nil {
				t.Fatalf("call: %v", err)
			}
			if gotKind != tt.kind {
				t.Errorf("kind = %q, want %q", gotKind, tt.kind)
			}
			if gotUser != "u-1" {
				t.Errorf("user = %q, want u-1", gotUser)
			}
		})
	}
}

func TestFavorites_PropagateRepoError(t *testing.T) {
	sentinel := errors.New("db down")
	fail := func(string, string, string) error { return sentinel }
	svc := user.NewService(&mockRepo{addFavoriteFn: fail, removeFavoriteFn: fail}, newTestAuthSvc())

	calls := []func() error{
		func() error { return svc.AddFavoriteTrack(context.Background(), "u-1", "t-1") },
		func() error { return svc.RemoveFavoriteTrack(context.Background(), "u-1", "t-1") },
		func() error { return svc.AddFavoriteStream(context.Background(), "u-1", "s-1") },
		func() error { return svc.RemoveFavoriteStream(context.Background(), "u-1", "s-1") },
		func() error { return svc.AddFavoritePlaylist(context.Background(), "u-1", "pl-1") },
		func() error { return svc.RemoveFavoritePlaylist(context.Background(), "u-1", "pl-1") },
	}
	for i, call := range calls {
		if err := call(); !errors.Is(err, sentinel) {
			t.Errorf("call %d: err = %v, want %v", i, err, sentinel)
		}
	}
}

// An invalid UUID from the database can't be turned into a JWT subject.
func TestRegister_InvalidUserIDIsAnError(t *testing.T) {
	svc := user.NewService(&mockRepo{
		createFn: func(context.Context, string, string, string, auth.Role) (*user.User, error) {
			return &user.User{Role: auth.RoleUser}, nil // zero (invalid) pgtype.UUID
		},
	}, newTestAuthSvc())

	if _, err := svc.Register(context.Background(), "alice", "alice@example.com", "secret"); err == nil {
		t.Fatal("expected an error for a user with no valid ID")
	}
}

func TestLogin_InvalidUserIDIsAnError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	svc := user.NewService(&mockRepo{
		getByEmailFn: func(context.Context, string) (*user.User, error) {
			return &user.User{PasswordHash: string(hash), Role: auth.RoleUser}, nil
		},
	}, newTestAuthSvc())

	if _, err := svc.Login(context.Background(), "alice@example.com", "secret"); err == nil {
		t.Fatal("expected an error for a user with no valid ID")
	}
}

// A repository failure other than "not found" must not be flattened into
// ErrInvalidCredentials — that would hide an outage as a login failure.
func TestLogin_PropagatesRepoError(t *testing.T) {
	sentinel := errors.New("db down")
	svc := user.NewService(&mockRepo{
		getByEmailFn: func(context.Context, string) (*user.User, error) { return nil, sentinel },
	}, newTestAuthSvc())

	if _, err := svc.Login(context.Background(), "alice@example.com", "secret"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
