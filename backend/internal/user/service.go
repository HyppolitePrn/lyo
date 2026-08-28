package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

// ErrInvalidCredentials is returned when email/password do not match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// Service handles user registration and authentication business logic.
type Service struct {
	repo    Repository
	authSvc *auth.Service
}

// NewService creates a Service with the given repository and JWT service.
func NewService(repo Repository, authSvc *auth.Service) *Service {
	return &Service{repo: repo, authSvc: authSvc}
}

// Register creates a new user account and returns a token pair.
func (s *Service) Register(ctx context.Context, username, email, password string) (auth.TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.repo.Create(ctx, username, email, string(hash), auth.RoleUser)
	if err != nil {
		return auth.TokenPair{}, err
	}

	idStr, err := getUserIDString(u)
	if err != nil {
		return auth.TokenPair{}, err
	}

	return s.authSvc.Issue(idStr, u.Role)
}

// Login verifies credentials and returns a token pair.
func (s *Service) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return auth.TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return auth.TokenPair{}, ErrInvalidCredentials
	}

	idStr, err := getUserIDString(u)
	if err != nil {
		return auth.TokenPair{}, err
	}

	return s.authSvc.Issue(idStr, u.Role)
}

// GetByID retrieves a user by their UUID string.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) AddFavoriteTrack(ctx context.Context, userID, trackID string) error {
	return s.repo.AddFavoriteTrack(ctx, userID, trackID)
}

func (s *Service) RemoveFavoriteTrack(ctx context.Context, userID, trackID string) error {
	return s.repo.RemoveFavoriteTrack(ctx, userID, trackID)
}

func (s *Service) AddFavoriteStream(ctx context.Context, userID, streamID string) error {
	return s.repo.AddFavoriteStream(ctx, userID, streamID)
}

func (s *Service) RemoveFavoriteStream(ctx context.Context, userID, streamID string) error {
	return s.repo.RemoveFavoriteStream(ctx, userID, streamID)
}

func (s *Service) AddFavoritePlaylist(ctx context.Context, userID, playlistID string) error {
	return s.repo.AddFavoritePlaylist(ctx, userID, playlistID)
}

func (s *Service) RemoveFavoritePlaylist(ctx context.Context, userID, playlistID string) error {
	return s.repo.RemoveFavoritePlaylist(ctx, userID, playlistID)
}

func getUserIDString(u *User) (string, error) {
	b, err := u.ID.Value()
	if err != nil {
		return "", fmt.Errorf("get user id value: %w", err)
	}
	s, ok := b.(string)
	if !ok {
		return "", fmt.Errorf("unexpected uuid type %T", b)
	}
	return s, nil
}
