package usecases

import (
	"context"
	"strings"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type StateUseCase struct {
	service    *services.StateService
	authorizer rootservices.Authorizer
}

func NewStateUseCase(service *services.StateService, authorizer rootservices.Authorizer) StateUseCase {
	return StateUseCase{service: service, authorizer: authorizer}
}

func (u StateUseCase) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateReadPermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.GetUserState(ctx, actor, userID)
}

func (u StateUseCase) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUpdatePermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.UpsertUserState(ctx, actor, userID, request, actorUserID)
}

func (u StateUseCase) CreateUserState(ctx context.Context, actor *models.Actor, userID string, request types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateCreatePermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.CreateUserState(ctx, actor, userID, request, actorUserID)
}

func (u StateUseCase) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUpdatePermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.UpdateUserState(ctx, actor, userID, request, actorUserID)
}

func (u StateUseCase) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateDeletePermission); err != nil {
		return err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return coreerrors.ErrBadRequest
	}

	return u.service.DeleteUserState(ctx, actor, userID)
}

func (u StateUseCase) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateListBannedPermission); err != nil {
		return nil, err
	}
	return u.service.GetBannedUserStates(ctx, actor)
}

func (u StateUseCase) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateReadPermission); err != nil {
		return nil, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.GetSessionState(ctx, actor, sessionID)
}

func (u StateUseCase) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateUpdatePermission); err != nil {
		return nil, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.UpsertSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u StateUseCase) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateCreatePermission); err != nil {
		return nil, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.CreateSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u StateUseCase) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateUpdatePermission); err != nil {
		return nil, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.UpdateSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u StateUseCase) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateDeletePermission); err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return coreerrors.ErrBadRequest
	}

	return u.service.DeleteSessionState(ctx, actor, sessionID)
}

func (u StateUseCase) GetSelfUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.GetSelfUserState(ctx, actor, userID)
}

func (u StateUseCase) GetSelfSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.GetSelfSessionState(ctx, sessionID)
}

func (u StateUseCase) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateListSessionsPermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.GetUserAdminSessions(ctx, actor, userID)
}

func (u StateUseCase) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateRevokePermission); err != nil {
		return nil, err
	}
	return u.service.RevokeSession(ctx, actor, sessionID, reason, actorUserID)
}

func (u StateUseCase) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateListRevokedPermission); err != nil {
		return nil, err
	}
	return u.service.GetRevokedSessionStates(ctx, actor)
}

func (u StateUseCase) BanUser(ctx context.Context, actor *models.Actor, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateBanPermission); err != nil {
		return nil, err
	}
	return u.service.BanUser(ctx, actor, userID, request, actorUserID)
}

func (u StateUseCase) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUnbanPermission); err != nil {
		return nil, err
	}
	return u.service.UnbanUser(ctx, actor, userID)
}
