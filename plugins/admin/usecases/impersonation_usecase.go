package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type ImpersonationUseCase struct {
	stateService         *services.StateService
	impersonationService *services.ImpersonationService
	authorizer           rootservices.Authorizer
}

func NewImpersonationUseCase(
	stateService *services.StateService,
	impersonationService *services.ImpersonationService,
	authorizer rootservices.Authorizer,
) ImpersonationUseCase {
	return ImpersonationUseCase{
		stateService:         stateService,
		impersonationService: impersonationService,
		authorizer:           authorizer,
	}
}

func (u ImpersonationUseCase) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsListPermission); err != nil {
		return nil, err
	}
	return u.impersonationService.GetAllImpersonations(ctx, actor)
}

func (u ImpersonationUseCase) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsReadPermission); err != nil {
		return nil, err
	}
	return u.impersonationService.GetImpersonationByID(ctx, actor, impersonationID)
}

func (u ImpersonationUseCase) StartImpersonation(ctx context.Context, actor *models.Actor, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest, impersonatorScopes []string, originalCookieValue string, originalCookieMaxAge int) (*types.StartImpersonationResult, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsStartPermission); err != nil {
		return nil, err
	}
	return u.impersonationService.StartImpersonation(ctx, actor, actorSessionID, ipAddress, userAgent, req, impersonatorScopes, originalCookieValue, originalCookieMaxAge)
}

func (u ImpersonationUseCase) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedUserID string, impersonatedSessionID string, originalCookieValue string, request types.StopImpersonationRequest) (*types.StopImpersonationResult, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsStopPermission); err != nil {
		return nil, err
	}
	return u.impersonationService.StopImpersonation(ctx, actor, impersonatedSessionID, originalCookieValue, request)
}
