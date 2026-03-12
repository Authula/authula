package services

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type StateService struct {
	userStateRepo     repositories.UserStateRepository
	sessionStateRepo  repositories.SessionStateRepository
	impersonationRepo repositories.ImpersonationRepository
}

func NewStateService(userStateRepo repositories.UserStateRepository, sessionStateRepo repositories.SessionStateRepository, impersonationRepo repositories.ImpersonationRepository) *StateService {
	return &StateService{userStateRepo: userStateRepo, sessionStateRepo: sessionStateRepo, impersonationRepo: impersonationRepo}
}

func (s *StateService) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) CreateUserState(ctx context.Context, userID string, request types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, constants.ErrConflict
	}

	state := buildUserStateFromCreate(userID, request, actorUserID)
	if err := s.userStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpdateUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, constants.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)
	if err := s.userStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)

	if err := s.userStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) DeleteUserState(ctx context.Context, userID string) error {
	return s.userStateRepo.Delete(ctx, userID)
}

func (s *StateService) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return s.userStateRepo.GetBanned(ctx)
}

func (s *StateService) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) CreateSessionState(ctx context.Context, sessionID string, request types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, constants.ErrConflict
	}

	state := buildSessionStateFromCreate(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpdateSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, constants.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)

	if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) DeleteSessionState(ctx context.Context, sessionID string) error {
	return s.sessionStateRepo.Delete(ctx, sessionID)
}

func (s *StateService) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, constants.ErrNotFound
	}

	return s.sessionStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return s.UpsertSessionState(ctx, sessionID, types.UpsertSessionStateRequest{
		Revoke:        true,
		RevokedReason: reason,
	}, actorUserID)
}

func (s *StateService) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return s.sessionStateRepo.GetRevoked(ctx)
}

func (s *StateService) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return s.UpsertUserState(ctx, userID, types.UpsertUserStateRequest{
		Banned:       true,
		BannedUntil:  request.BannedUntil,
		BannedReason: request.Reason,
	}, actorUserID)
}

func (s *StateService) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return s.UpsertUserState(ctx, userID, types.UpsertUserStateRequest{Banned: false}, nil)
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
