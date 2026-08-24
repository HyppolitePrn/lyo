package passwordreset

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/mailer"
)

// ErrInvalidToken covers a token that is unknown, expired, or already used.
// The three cases are deliberately not distinguished so callers can't probe token state.
var ErrInvalidToken = errors.New("invalid or expired token")

const (
	tokenTTL      = time.Hour
	tokenByteLen  = 32
	resetLinkBase = "lyo://reset-password"
	emailSubject  = "Reset your Lyo password"
)

// UserRepo is the subset of user.Repository consumed by this service.
type UserRepo interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}

// Service issues and redeems password reset tokens.
type Service struct {
	repo     Repository
	userRepo UserRepo
	mailer   mailer.Sender
	logger   *slog.Logger
}

// NewService creates a Service.
func NewService(repo Repository, userRepo UserRepo, sender mailer.Sender, logger *slog.Logger) *Service {
	return &Service{repo: repo, userRepo: userRepo, mailer: sender, logger: logger}
}

// ForgotPassword looks up the user by email and, if found, issues a reset token and
// emails the reset link. An unknown email is not an error: the caller must not be able
// to distinguish "sent" from "no such user" to avoid user enumeration.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	rawToken, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	userID, err := userIDString(u)
	if err != nil {
		return err
	}

	if err := s.repo.Create(ctx, userID, hashToken(rawToken), time.Now().Add(tokenTTL)); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	link := fmt.Sprintf("%s?token=%s", resetLinkBase, rawToken)
	body := fmt.Sprintf("Click the link below to reset your Lyo password. It expires in 1 hour.\n\n%s", link)
	if err := s.mailer.Send(ctx, u.Email, emailSubject, body); err != nil {
		s.logger.ErrorContext(ctx, "send reset email failed", "err", err)
	}
	return nil
}

// ResetPassword validates the token and, if valid, updates the user's password and
// marks the token used. Returns ErrInvalidToken for any unknown, expired, or reused token.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tok, err := s.repo.GetByHash(ctx, hashToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidToken
	}
	if err != nil {
		return err
	}
	if tok.UsedAt != nil || time.Now().After(tok.ExpiresAt) {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, tok.UserID, string(hash)); err != nil {
		return err
	}
	return s.repo.MarkUsed(ctx, tok.ID)
}

// generateToken returns a URL-safe random token suitable for embedding in a deep link.
func generateToken() (string, error) {
	b := make([]byte, tokenByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken hashes a raw token for storage/lookup. SHA-256 (not bcrypt) is appropriate
// here: the token is already 256 bits of random entropy, so a fast hash keeps DB lookups
// cheap while a stolen dump still can't be turned back into a usable token.
func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func userIDString(u *user.User) (string, error) {
	v, err := u.ID.Value()
	if err != nil {
		return "", fmt.Errorf("get user id value: %w", err)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("unexpected uuid type %T", v)
	}
	return s, nil
}
