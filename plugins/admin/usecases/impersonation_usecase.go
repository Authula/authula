package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type ImpersonationUseCase struct {
	stateService         *services.StateService
	impersonationService *services.ImpersonationService
}

func NewImpersonationUseCase(
	stateService *services.StateService,
	impersonationService *services.ImpersonationService,
) ImpersonationUseCase {
	return ImpersonationUseCase{
		stateService:         stateService,
		impersonationService: impersonationService,
	}
}

func (u ImpersonationUseCase) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return u.impersonationService.GetAllImpersonations(ctx)
}

func (u ImpersonationUseCase) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return u.impersonationService.GetImpersonationByID(ctx, impersonationID)
}

func (u ImpersonationUseCase) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return u.impersonationService.StartImpersonation(ctx, actorUserID, actorSessionID, ipAddress, userAgent, req)
}

func (u ImpersonationUseCase) StopImpersonation(ctx context.Context, impersonatedUserID string, impersonatedSessionID string, request types.StopImpersonationRequest) error {
	sessionState, err := u.stateService.GetSessionState(ctx, impersonatedSessionID)
	if err != nil {
		return err
	}

	if sessionState == nil || sessionState.ImpersonatorUserID == nil {
		return constants.ErrUnauthorized
	}

	actorUserID := *sessionState.ImpersonatorUserID
	if actorUserID == "" {
		return constants.ErrUnauthorized
	}

	return u.impersonationService.StopImpersonation(ctx, actorUserID, request)
}
