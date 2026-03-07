package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type ImpersonationUseCase struct {
	service *services.ImpersonationService
}

func NewImpersonationUseCase(
	service *services.ImpersonationService,
) ImpersonationUseCase {
	return ImpersonationUseCase{
		service: service,
	}
}

func (u ImpersonationUseCase) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return u.service.GetAllImpersonations(ctx)
}

func (u ImpersonationUseCase) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return u.service.GetImpersonationByID(ctx, impersonationID)
}

func (u ImpersonationUseCase) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return u.service.StartImpersonation(ctx, actorUserID, actorSessionID, req)
}

func (u ImpersonationUseCase) StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error {
	return u.service.StopImpersonation(ctx, actorUserID, request)
}
