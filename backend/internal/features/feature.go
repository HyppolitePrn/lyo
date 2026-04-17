package features

import (
	"context"
	"time"
)

// Flag represents a toggleable feature flag stored in the database.
type Flag struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Enabled     bool      `db:"enabled"`
	Description string    `db:"description"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Repository defines the storage interface for feature flags.
type Repository interface {
	All(ctx context.Context) ([]Flag, error)
	IsEnabled(ctx context.Context, name string) (bool, error)
	Toggle(ctx context.Context, name string, enabled bool) (*Flag, error)
}

// Service wraps the repository and exposes the feature flag API.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) All(ctx context.Context) ([]Flag, error) {
	return s.repo.All(ctx)
}

// IsEnabled returns false (fail-closed) if the flag is not found.
func (s *Service) IsEnabled(ctx context.Context, name string) bool {
	ok, err := s.repo.IsEnabled(ctx, name)
	if err != nil {
		return false
	}
	return ok
}

func (s *Service) Toggle(ctx context.Context, name string, enabled bool) (*Flag, error) {
	return s.repo.Toggle(ctx, name, enabled)
}
