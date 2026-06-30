package services

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type StateService struct {
	userStateRepo     repositories.UserStateRepository
	sessionStateRepo  repositories.SessionStateRepository
	impersonationRepo repositories.ImpersonationRepository
	authorizer        rootservices.Authorizer
}

func NewStateService(userStateRepo repositories.UserStateRepository, sessionStateRepo repositories.SessionStateRepository, impersonationRepo repositories.ImpersonationRepository, authorizer rootservices.Authorizer) *StateService {
	return &StateService{userStateRepo: userStateRepo, sessionStateRepo: sessionStateRepo, impersonationRepo: impersonationRepo, authorizer: authorizer}
}

func (s *StateService) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateReadPermission); err != nil {
		return nil, err
	}
	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) CreateUserState(ctx context.Context, actor *models.Actor, userID string, request types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateCreatePermission); err != nil {
		return nil, err
	}
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, internalerrors.ErrConflict
	}

	state := buildUserStateFromCreate(userID, request, actorUserID)
	if err := s.userStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUpdatePermission); err != nil {
		return nil, err
	}
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, internalerrors.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)
	if err := s.userStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUpdatePermission); err != nil {
		return nil, err
	}
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)

	if err := s.userStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateDeletePermission); err != nil {
		return err
	}
	return s.userStateRepo.Delete(ctx, userID)
}

func (s *StateService) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateListBannedPermission); err != nil {
		return nil, err
	}
	return s.userStateRepo.GetBanned(ctx)
}

func (s *StateService) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateReadPermission); err != nil {
		return nil, err
	}
	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateCreatePermission); err != nil {
		return nil, err
	}
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, internalerrors.ErrConflict
	}

	state := buildSessionStateFromCreate(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateUpdatePermission); err != nil {
		return nil, err
	}
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, internalerrors.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateUpdatePermission); err != nil {
		return nil, err
	}
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)

	if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateDeletePermission); err != nil {
		return err
	}
	return s.sessionStateRepo.Delete(ctx, sessionID)
}

func (s *StateService) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateListSessionsPermission); err != nil {
		return nil, err
	}
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, internalerrors.ErrNotFound
	}

	return s.sessionStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateRevokePermission); err != nil {
		return nil, err
	}
	return s.UpsertSessionState(ctx, actor, sessionID, types.UpsertSessionStateRequest{
		Revoke:        true,
		RevokedReason: reason,
	}, actorUserID)
}

func (s *StateService) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.SessionStateListRevokedPermission); err != nil {
		return nil, err
	}
	return s.sessionStateRepo.GetRevoked(ctx)
}

func (s *StateService) BanUser(ctx context.Context, actor *models.Actor, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateBanPermission); err != nil {
		return nil, err
	}
	return s.UpsertUserState(ctx, actor, userID, types.UpsertUserStateRequest{
		Banned:       true,
		BannedUntil:  request.BannedUntil,
		BannedReason: request.Reason,
	}, actorUserID)
}

func (s *StateService) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UserStateUnbanPermission); err != nil {
		return nil, err
	}
	return s.UpsertUserState(ctx, actor, userID, types.UpsertUserStateRequest{Banned: false}, nil)
}

func buildUserState(userID string, request types.UpsertUserStateRequest, actorUserID *string) *types.AdminUserState {
	state := &types.AdminUserState{
		UserID: userID,
		Banned: request.Banned,
	}
	if request.Banned {
		now := time.Now().UTC()
		state.BannedAt = &now
		state.BannedUntil = request.BannedUntil
		state.BannedReason = request.BannedReason
		state.BannedByUserID = actorUserID
	}

	return state
}

func buildSessionState(sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) *types.AdminSessionState {
	state := &types.AdminSessionState{SessionID: sessionID}
	if request.Revoke {
		now := time.Now().UTC()
		state.RevokedAt = &now
		state.RevokedReason = request.RevokedReason
		state.RevokedByUserID = actorUserID
		state.ImpersonatorUserID = request.ImpersonatorUserID
		state.ImpersonationReason = request.ImpersonationReason
		state.ImpersonationExpiresAt = request.ImpersonationExpiresAt
	}

	return state
}

func buildUserStateFromCreate(userID string, request types.CreateUserStateRequest, actorUserID *string) *types.AdminUserState {
	state := &types.AdminUserState{
		UserID: userID,
		Banned: request.Banned,
	}
	if request.Banned {
		now := time.Now().UTC()
		state.BannedAt = &now
		state.BannedUntil = request.BannedUntil
		state.BannedReason = request.BannedReason
		state.BannedByUserID = actorUserID
	}

	return state
}

func buildSessionStateFromCreate(sessionID string, request types.CreateSessionStateRequest, actorUserID *string) *types.AdminSessionState {
	state := &types.AdminSessionState{SessionID: sessionID}
	if request.Revoke {
		now := time.Now().UTC()
		state.RevokedAt = &now
		state.RevokedReason = request.RevokedReason
		state.RevokedByUserID = actorUserID
		state.ImpersonatorUserID = request.ImpersonatorUserID
		state.ImpersonationReason = request.ImpersonationReason
		state.ImpersonationExpiresAt = request.ImpersonationExpiresAt
	}

	return state
}
