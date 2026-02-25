package services

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UserStateService struct {
	repo *repositories.UserStateRepository
}

func NewUserStateService(repo *repositories.UserStateRepository) *UserStateService {
	return &UserStateService{repo: repo}
}

func (s *UserStateService) GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *UserStateService) Upsert(ctx context.Context, state *types.AdminUserState) error {
	return s.repo.Upsert(ctx, state)
}

func (s *UserStateService) Delete(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}

func (s *UserStateService) GetBanned(ctx context.Context) ([]types.AdminUserState, error) {
	return s.repo.GetBanned(ctx)
}
