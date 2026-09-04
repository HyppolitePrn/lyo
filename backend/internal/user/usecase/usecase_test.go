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
	deleteFn  func(ctx context.Context, id string) error
}

func (s *stubRepo) GetByID(ctx context.Context, id string) (*user.User, error) {
	return s.getByIDFn(ctx, id)
}
func (s *stubRepo) Update(ctx context.Context, id string, username, email *string) (*user.User, error) {
	return s.updateFn(ctx, id, username, email)
}
func (s *stubRepo) Delete(ctx context.Context, id string) error {
	return s.deleteFn(ctx, id)
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

// ── DeleteUserByID ────────────────────────────────────────────────────────────

// stubPurger records the broadcaster whose audio it was asked to erase.
type stubPurger struct {
	err    error
	called []string
}

func (s *stubPurger) PurgeByBroadcaster(_ context.Context, broadcasterID string) error {
	s.called = append(s.called, broadcasterID)
	return s.err
}

func TestDeleteUserByID_ErasesAudioThenTheAccount(t *testing.T) {
	const id = "u-1"
	var order []string

	purger := &stubPurger{}
	repo := &stubRepo{deleteFn: func(_ context.Context, got string) error {
		order = append(order, "user:"+got)
		return nil
	}}
	// Wrap the purger so both calls land in one ordered list.
	uc := usecase.NewDeleteUserByIDUsecase(repo, purgeFunc(func(ctx context.Context, b string) error {
		order = append(order, "tracks:"+b)
		return purger.PurgeByBroadcaster(ctx, b)
	}))

	if err := uc.Execute(context.Background(), id); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Order matters: the track rows cascade away with the user, so once the
	// account is gone nothing records which S3 objects were its audio.
	want := []string{"tracks:" + id, "user:" + id}
	if len(order) != 2 || order[0] != want[0] || order[1] != want[1] {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

// A storage failure must leave the account intact: that state is retryable,
// whereas a deleted account with undeletable audio behind it is not.
func TestDeleteUserByID_KeepsTheAccountWhenAudioCannotBeErased(t *testing.T) {
	boom := errors.New("s3 unreachable")
	deleted := false
	repo := &stubRepo{deleteFn: func(context.Context, string) error {
		deleted = true
		return nil
	}}
	uc := usecase.NewDeleteUserByIDUsecase(repo, &stubPurger{err: boom})

	err := uc.Execute(context.Background(), "u-1")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
	if deleted {
		t.Fatal("the account was deleted even though its audio could not be")
	}
}

// No storage backend configured means there is nothing to purge, not a crash.
func TestDeleteUserByID_ToleratesANilPurger(t *testing.T) {
	deleted := ""
	repo := &stubRepo{deleteFn: func(_ context.Context, id string) error {
		deleted = id
		return nil
	}}
	uc := usecase.NewDeleteUserByIDUsecase(repo, nil)

	if err := uc.Execute(context.Background(), "u-1"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if deleted != "u-1" {
		t.Fatalf("deleted %q, want %q", deleted, "u-1")
	}
}

func TestDeleteUserByID_PropagatesRepoError(t *testing.T) {
	repo := &stubRepo{deleteFn: func(context.Context, string) error { return user.ErrNotFound }}
	uc := usecase.NewDeleteUserByIDUsecase(repo, &stubPurger{})

	if err := uc.Execute(context.Background(), "u-1"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, user.ErrNotFound)
	}
}

// purgeFunc adapts a function to usecase.TrackPurger.
type purgeFunc func(ctx context.Context, broadcasterID string) error

func (f purgeFunc) PurgeByBroadcaster(ctx context.Context, broadcasterID string) error {
	return f(ctx, broadcasterID)
}
