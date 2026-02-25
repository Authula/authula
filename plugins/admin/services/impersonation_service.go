package services

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type ImpersonationService struct {
	repo *repositories.ImpersonationRepository
}

func NewImpersonationService(repo *repositories.ImpersonationRepository) *ImpersonationService {
	return &ImpersonationService{repo: repo}
}

func (s *ImpersonationService) UserExists(ctx context.Context, userID string) (bool, error) {
	return s.repo.UserExists(ctx, userID)
}

func (s *ImpersonationService) CreateImpersonation(ctx context.Context, impersonation *types.Impersonation) error {
	return s.repo.CreateImpersonation(ctx, impersonation)
}

func (s *ImpersonationService) GetActiveImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return s.repo.GetActiveImpersonationByID(ctx, impersonationID)
}

func (s *ImpersonationService) GetLatestActiveImpersonationByActor(ctx context.Context, actorUserID string) (*types.Impersonation, error) {
	return s.repo.GetLatestActiveImpersonationByActor(ctx, actorUserID)
}

func (s *ImpersonationService) EndImpersonation(ctx context.Context, impersonationID string, endedByUserID *string) error {
	return s.repo.EndImpersonation(ctx, impersonationID, endedByUserID)
}

func (s *ImpersonationService) GetImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return s.repo.GetAllImpersonations(ctx)
}

func (s *ImpersonationService) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return s.repo.GetImpersonationByID(ctx, impersonationID)
}
