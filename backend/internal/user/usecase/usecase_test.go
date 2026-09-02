package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/internal/user/usecase"
)

// stubRepo implements user.Repository; only the methods under test do anything.
type stubRepo struct {
	user.Repository
	getByIDFn func(ctx context.Context, id string) (*user.User, error)
	updateFn  func(ctx context.Context, id string, username, email *string) (*user.User, error)
}

func (s *stubRepo) GetByID(ctx context.Context, id string) (*user.User, error) {
	return s.getByIDFn(ctx, id)
}
func (s *stubRepo) Update(ctx context.Context, id string, username, email *string) (*user.User, error) {
	return s.updateFn(ctx, id, username, email)
}

func newUser(id uuid.UUID, username string) *user.User {
	return &user.User{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		Username: username,
		Email:    "alice@example.com",
		Role:     auth.RoleUser,
	}
}

func TestGetUserByID_ReturnsRepoUser(t *testing.T) {
	id := uuid.New()
	var gotID string
	uc := usecase.NewGetUserByIDUsecase(&stubRepo{
		getByIDFn: func(_ context.Context, id string) (*user.User, error) {
			gotID = id
			return newUser(uuid.MustParse(id), "alice"), nil
		},
	})

	u, err := uc.Execute(context.Background(), id.String())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotID != id.String() {
		t.Errorf("repo called with %q, want %q", gotID, id)
	}
	if u.Username != "alice" {
		t.Errorf("username = %q, want alice", u.Username)
	}
}

func TestGetUserByID_PropagatesNotFound(t *testing.T) {
	uc := usecase.NewGetUserByIDUsecase(&stubRepo{
		getByIDFn: func(context.Context, string) (*user.User, error) { return nil, user.ErrNotFound },
	})

	if _, err := uc.Execute(context.Background(), uuid.NewString()); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

func TestUpdateUserByID_PassesPartialFieldsThrough(t *testing.T) {
	id := uuid.New()
	name := "newname"
	var gotUsername, gotEmail *string

	uc := usecase.NewUpdateUserByIDUsecase(&stubRepo{
		updateFn: func(_ context.Context, _ string, username, email *string) (*user.User, error) {
			gotUsername, gotEmail = username, email
			return newUser(id, "newname"), nil
		},
	})

	u, err := uc.Execute(context.Background(), id.String(), &name, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotUsername == nil || *gotUsername != name {
		t.Errorf("username = %v, want %q", gotUsername, name)
	}
	if gotEmail != nil {
		t.Errorf("email should stay nil, got %q", *gotEmail)
	}
	if u.Username != "newname" {
		t.Errorf("username = %q", u.Username)
	}
}

func TestUpdateUserByID_PropagatesRepoError(t *testing.T) {
	sentinel := errors.New("constraint violation")
	uc := usecase.NewUpdateUserByIDUsecase(&stubRepo{
		updateFn: func(context.Context, string, *string, *string) (*user.User, error) { return nil, sentinel },
	})

	if _, err := uc.Execute(context.Background(), uuid.NewString(), nil, nil); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
