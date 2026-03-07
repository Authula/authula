package usecases

import (
	"context"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type StateUseCase struct {
	service *services.StateService
}

func NewStateUseCase(service *services.StateService) StateUseCase {
	return StateUseCase{service: service}
}

func (u StateUseCase) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.GetUserState(ctx, userID)
}

func (u StateUseCase) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.UpsertUserState(ctx, userID, request, actorUserID)
}

func (u StateUseCase) DeleteUserState(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return constants.ErrBadRequest
	}

	return u.service.DeleteUserState(ctx, userID)
}

func (u StateUseCase) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return u.service.GetBannedUserStates(ctx)
}

func (u StateUseCase) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.GetSessionState(ctx, sessionID)
}

func (u StateUseCase) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.UpsertSessionState(ctx, sessionID, request, actorUserID)
}

func (u StateUseCase) DeleteSessionState(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return constants.ErrBadRequest
	}

	return u.service.DeleteSessionState(ctx, sessionID)
}

func (u StateUseCase) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.GetUserAdminSessions(ctx, userID)
}

func (u StateUseCase) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return u.service.RevokeSession(ctx, sessionID, reason, actorUserID)
}

func (u StateUseCase) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return u.service.GetRevokedSessionStates(ctx)
}

func (u StateUseCase) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.service.BanUser(ctx, userID, request, actorUserID)
}

func (u StateUseCase) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.service.UnbanUser(ctx, userID)
}
