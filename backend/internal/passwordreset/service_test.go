package passwordreset_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/observability"
	"github.com/hyppoliteprn/lyo/internal/passwordreset"
	"github.com/hyppoliteprn/lyo/internal/user"
)

type mockRepo struct {
	createFn    func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	getByHashFn func(ctx context.Context, tokenHash string) (*passwordreset.Token, error)
	markUsedFn  func(ctx context.Context, id string) error
}

func (m *mockRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	return m.createFn(ctx, userID, tokenHash, expiresAt)
}
func (m *mockRepo) GetByHash(ctx context.Context, tokenHash string) (*passwordreset.Token, error) {
	return m.getByHashFn(ctx, tokenHash)
}
func (m *mockRepo) MarkUsed(ctx context.Context, id string) error {
	return m.markUsedFn(ctx, id)
}

type mockUserRepo struct {
	getByEmailFn     func(ctx context.Context, email string) (*user.User, error)
	updatePasswordFn func(ctx context.Context, id, passwordHash string) error
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.getByEmailFn(ctx, email)
}
func (m *mockUserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return m.updatePasswordFn(ctx, id, passwordHash)
}

type mockMailer struct {
	sendFn func(ctx context.Context, to, subject, body string) error
}

func (m *mockMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.sendFn == nil {
		return nil
	}
	return m.sendFn(ctx, to, subject, body)
}

func fakeUser() *user.User {
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000001")
	return &user.User{ID: id, Email: "alice@example.com"}
}

func TestForgotPassword_UnknownEmail_NoError_NoEmailSent(t *testing.T) {
	sent := false
	svc := passwordreset.NewService(
		&mockRepo{},
		&mockUserRepo{
			getByEmailFn: func(_ context.Context, _ string) (*user.User, error) { return nil, user.ErrNotFound },
		},
		&mockMailer{sendFn: func(_ context.Context, _, _, _ string) error { sent = true; return nil }},
		observability.NewLogger("error"),
	)

	if err := svc.ForgotPassword(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent {
		t.Fatal("expected no email to be sent for an unknown address")
	}
}

func TestForgotPassword_KnownEmail_StoresTokenAndSendsEmail(t *testing.T) {
	var storedHash string
	sent := false
	svc := passwordreset.NewService(
		&mockRepo{
			createFn: func(_ context.Context, _, tokenHash string, _ time.Time) error {
				storedHash = tokenHash
				return nil
			},
		},
		&mockUserRepo{
			getByEmailFn: func(_ context.Context, _ string) (*user.User, error) { return fakeUser(), nil },
		},
		&mockMailer{sendFn: func(_ context.Context, _, _, _ string) error { sent = true; return nil }},
		observability.NewLogger("error"),
	)

	if err := svc.ForgotPassword(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storedHash == "" {
		t.Fatal("expected a token hash to be stored")
	}
	if !sent {
		t.Fatal("expected an email to be sent")
	}
}

func TestResetPassword_ExpiredToken_ReturnsErrInvalidToken(t *testing.T) {
	svc := passwordreset.NewService(
		&mockRepo{
			getByHashFn: func(_ context.Context, _ string) (*passwordreset.Token, error) {
				return &passwordreset.Token{ID: "t1", UserID: "u1", ExpiresAt: time.Now().Add(-time.Minute)}, nil
			},
		},
		&mockUserRepo{},
		&mockMailer{},
		observability.NewLogger("error"),
	)

	err := svc.ResetPassword(context.Background(), "raw-token", "newpassword123")
	if err != passwordreset.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestResetPassword_AlreadyUsedToken_ReturnsErrInvalidToken(t *testing.T) {
	used := time.Now().Add(-time.Minute)
	svc := passwordreset.NewService(
		&mockRepo{
			getByHashFn: func(_ context.Context, _ string) (*passwordreset.Token, error) {
				return &passwordreset.Token{ID: "t1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour), UsedAt: &used}, nil
			},
		},
		&mockUserRepo{},
		&mockMailer{},
		observability.NewLogger("error"),
	)

	err := svc.ResetPassword(context.Background(), "raw-token", "newpassword123")
	if err != passwordreset.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestResetPassword_ValidToken_UpdatesPasswordAndMarksUsed(t *testing.T) {
	var updatedID, updatedHash string
	markedUsedID := ""
	svc := passwordreset.NewService(
		&mockRepo{
			getByHashFn: func(_ context.Context, _ string) (*passwordreset.Token, error) {
				return &passwordreset.Token{ID: "t1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
			markUsedFn: func(_ context.Context, id string) error {
				markedUsedID = id
				return nil
			},
		},
		&mockUserRepo{
			updatePasswordFn: func(_ context.Context, id, passwordHash string) error {
				updatedID = id
				updatedHash = passwordHash
				return nil
			},
		},
		&mockMailer{},
		observability.NewLogger("error"),
	)

	if err := svc.ResetPassword(context.Background(), "raw-token", "newpassword123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedID != "u1" || updatedHash == "" {
		t.Fatalf("expected password update for u1 with a non-empty hash, got id=%q hash=%q", updatedID, updatedHash)
	}
	if markedUsedID != "t1" {
		t.Fatalf("expected token t1 to be marked used, got %q", markedUsedID)
	}
}

func newLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// A repository failure other than "not found" must not be swallowed: unlike an
// unknown address, it is a real outage the caller should see.
func TestForgotPassword_PropagatesUserRepoError(t *testing.T) {
	sentinel := errors.New("db down")
	svc := passwordreset.NewService(
		&mockRepo{},
		&mockUserRepo{getByEmailFn: func(context.Context, string) (*user.User, error) { return nil, sentinel }},
		&mockMailer{}, newLogger(),
	)

	if err := svc.ForgotPassword(context.Background(), "alice@example.com"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestForgotPassword_TokenStorageFailureIsReported(t *testing.T) {
	sentinel := errors.New("db down")
	svc := passwordreset.NewService(
		&mockRepo{createFn: func(context.Context, string, string, time.Time) error { return sentinel }},
		&mockUserRepo{getByEmailFn: func(context.Context, string) (*user.User, error) { return fakeUser(), nil }},
		&mockMailer{}, newLogger(),
	)

	err := svc.ForgotPassword(context.Background(), "alice@example.com")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want it to wrap %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "store reset token") {
		t.Errorf("error not wrapped with context: %v", err)
	}
}

// A user whose ID didn't scan can't be given a token.
func TestForgotPassword_InvalidUserIDIsAnError(t *testing.T) {
	svc := passwordreset.NewService(
		&mockRepo{},
		&mockUserRepo{getByEmailFn: func(context.Context, string) (*user.User, error) {
			return &user.User{Email: "alice@example.com"}, nil // zero (invalid) pgtype.UUID
		}},
		&mockMailer{}, newLogger(),
	)

	if err := svc.ForgotPassword(context.Background(), "alice@example.com"); err == nil {
		t.Fatal("expected an error for a user with no valid ID")
	}
}

// A bounced email must not fail the request: the token is already stored, and
// reporting the failure would leak that the address exists.
func TestForgotPassword_MailerFailureIsSwallowed(t *testing.T) {
	svc := passwordreset.NewService(
		&mockRepo{createFn: func(context.Context, string, string, time.Time) error { return nil }},
		&mockUserRepo{getByEmailFn: func(context.Context, string) (*user.User, error) { return fakeUser(), nil }},
		&mockMailer{sendFn: func(context.Context, string, string, string) error { return errors.New("smtp down") }},
		newLogger(),
	)

	if err := svc.ForgotPassword(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("expected the send failure to be logged, not returned: %v", err)
	}
}

func TestResetPassword_PropagatesLookupError(t *testing.T) {
	sentinel := errors.New("db down")
	svc := passwordreset.NewService(
		&mockRepo{getByHashFn: func(context.Context, string) (*passwordreset.Token, error) { return nil, sentinel }},
		&mockUserRepo{}, &mockMailer{}, newLogger(),
	)

	if err := svc.ResetPassword(context.Background(), "raw", "newsecret123"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestResetPassword_PropagatesUpdateError(t *testing.T) {
	sentinel := errors.New("db down")
	svc := passwordreset.NewService(
		&mockRepo{getByHashFn: func(context.Context, string) (*passwordreset.Token, error) {
			return &passwordreset.Token{ID: "tok-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}},
		&mockUserRepo{updatePasswordFn: func(context.Context, string, string) error { return sentinel }},
		&mockMailer{}, newLogger(),
	)

	if err := svc.ResetPassword(context.Background(), "raw", "newsecret123"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}
