package services

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type SessionStateService struct {
	repo *repositories.SessionStateRepository
}

func NewSessionStateService(repo *repositories.SessionStateRepository) *SessionStateService {
	return &SessionStateService{repo: repo}
}

func (s *SessionStateService) GetBySessionID(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return s.repo.GetBySessionID(ctx, sessionID)
}

func (s *SessionStateService) Upsert(ctx context.Context, state *types.AdminSessionState) error {
	return s.repo.Upsert(ctx, state)
}

func (s *SessionStateService) Delete(ctx context.Context, sessionID string) error {
	return s.repo.Delete(ctx, sessionID)
}

func (s *SessionStateService) GetRevoked(ctx context.Context) ([]types.AdminSessionState, error) {
	return s.repo.GetRevoked(ctx)
}

func (s *SessionStateService) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	return s.repo.SessionExists(ctx, sessionID)
}

func (s *SessionStateService) GetByUserID(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	return s.repo.GetByUserID(ctx, userID)
}
