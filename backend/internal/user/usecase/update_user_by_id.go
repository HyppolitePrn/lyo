package usecase

import (
	"context"

	"github.com/hyppoliteprn/lyo/internal/user"
)

// UpdateUserByIDUsecase applies a partial update (username and/or email) to a user.
type UpdateUserByIDUsecase struct {
	repo user.Repository
}

// NewUpdateUserByIDUsecase creates an UpdateUserByIDUsecase backed by the given repository.
func NewUpdateUserByIDUsecase(repo user.Repository) *UpdateUserByIDUsecase {
	return &UpdateUserByIDUsecase{repo: repo}
}

// Execute updates the user identified by id. Nil fields are left untouched.
func (uc *UpdateUserByIDUsecase) Execute(ctx context.Context, id string, username, email *string) (*user.User, error) {
	return uc.repo.Update(ctx, id, username, email)
}
