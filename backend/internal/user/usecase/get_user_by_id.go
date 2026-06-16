package usecase

import (
	"context"

	"github.com/hyppoliteprn/lyo/internal/user"
)

type GetUserByIDUsecase struct {
	repo user.Repository
}

func NewGetUserByIDUsecase(repo user.Repository) *GetUserByIDUsecase {
	return &GetUserByIDUsecase{repo: repo}
}

func (uc *GetUserByIDUsecase) Execute(ctx context.Context, id string) (*user.User, error) {
	return uc.repo.GetByID(ctx, id)
}
