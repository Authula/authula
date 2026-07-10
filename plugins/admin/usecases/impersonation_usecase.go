package usecases

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
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

func (u ImpersonationUseCase) StartImpersonation(ctx context.Context, actor *models.Actor, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest, impersonatorScopes []string, originalCookieValue string, originalCookieMaxAge int) (*types.StartImpersonationResult, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsStartPermission); err != nil {
		return nil, err
	}
	return u.impersonationService.StartImpersonation(ctx, actor, actorUserID, actorSessionID, ipAddress, userAgent, req, impersonatorScopes, originalCookieValue, originalCookieMaxAge)
}

func (u ImpersonationUseCase) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedUserID string, impersonatedSessionID string, originalCookieValue string, request types.StopImpersonationRequest) (*types.StopImpersonationResult, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.ImpersonationsStopPermission); err != nil {
		return nil, err
	}
	sessionState, err := u.stateService.GetSessionState(ctx, actor, impersonatedSessionID)
	if err != nil {
		return nil, err
	}

	if sessionState == nil || sessionState.ImpersonatorUserID == nil {
		return nil, coreerrors.ErrUnauthorized
	}

	actorUserID := *sessionState.ImpersonatorUserID
	if actorUserID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	if originalCookieValue != "" {
		originalSession, err := u.impersonationService.ValidateImpersonationCookie(ctx, originalCookieValue)
		if err != nil {
			return nil, err
		}

		if sessionState.ImpersonatorSessionID == nil || *sessionState.ImpersonatorSessionID != originalSession.ID {
			return nil, coreerrors.ErrUnauthorized
		}
	} else {
		if actor == nil || actor.Claims == nil {
			return nil, coreerrors.ErrUnauthorized
		}
		impersonatorID, ok := actor.Claims[adminconstants.ImpersonatorID]
		if !ok {
			return nil, coreerrors.ErrUnauthorized
		}
		impersonatorIDStr, ok := impersonatorID.(string)
		if !ok || impersonatorIDStr == "" || impersonatorIDStr != actorUserID {
			return nil, coreerrors.ErrUnauthorized
		}
	}

	if err := u.impersonationService.StopImpersonation(ctx, actor, actorUserID, request); err != nil {
		return nil, err
	}

	return &types.StopImpersonationResult{
		OriginalSessionToken: originalCookieValue,
	}, nil
}
