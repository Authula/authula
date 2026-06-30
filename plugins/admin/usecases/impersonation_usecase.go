package usecases

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
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

func (u ImpersonationUseCase) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	return u.impersonationService.GetAllImpersonations(ctx, actor)
}

func (u ImpersonationUseCase) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	return u.impersonationService.GetImpersonationByID(ctx, actor, impersonationID)
}

func (u ImpersonationUseCase) StartImpersonation(ctx context.Context, actor *models.Actor, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return u.impersonationService.StartImpersonation(ctx, actor, actorUserID, actorSessionID, ipAddress, userAgent, req)
}

func (u ImpersonationUseCase) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedUserID string, impersonatedSessionID string, request types.StopImpersonationRequest) error {
	sessionState, err := u.stateService.GetSessionState(ctx, actor, impersonatedSessionID)
	if err != nil {
		return err
	}

	if sessionState == nil || sessionState.ImpersonatorUserID == nil {
		return internalerrors.ErrUnauthorized
	}

	actorUserID := *sessionState.ImpersonatorUserID
	if actorUserID == "" {
		return internalerrors.ErrUnauthorized
	}

	return u.impersonationService.StopImpersonation(ctx, actor, actorUserID, request)
}
