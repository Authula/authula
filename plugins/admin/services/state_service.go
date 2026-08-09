package services

import (
	"context"
	"strings"
	"time"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
)

type StateService struct {
	userStateRepo     repositories.UserStateRepository
	sessionStateRepo  repositories.SessionStateRepository
	impersonationRepo repositories.ImpersonationRepository
}

func NewStateService(userStateRepo repositories.UserStateRepository, sessionStateRepo repositories.SessionStateRepository, impersonationRepo repositories.ImpersonationRepository) *StateService {
	return &StateService{userStateRepo: userStateRepo, sessionStateRepo: sessionStateRepo, impersonationRepo: impersonationRepo}
}

func (s *StateService) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) CreateUserState(ctx context.Context, actor *models.Actor, userID string, request types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, coreerrors.ErrConflict
	}

	state := buildUserStateFromCreate(userID, request, actorUserID)
	if err := s.userStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	current, err := s.userStateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, coreerrors.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)
	if err := s.userStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	state := buildUserState(userID, request, actorUserID)

	if err := s.userStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.userStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return coreerrors.ErrBadRequest
	}

	return s.userStateRepo.Delete(ctx, userID)
}

func (s *StateService) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	return s.userStateRepo.GetBanned(ctx)
}

func (s *StateService) GetSelfUserState(ctx context.Context, actor *models.Actor) (*types.AdminUserState, error) {
	if actor == nil || strings.TrimSpace(actor.ID) == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return s.userStateRepo.GetByUserID(ctx, actor.ID)
}

func (s *StateService) GetSelfSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, coreerrors.ErrConflict
	}

	state := buildSessionStateFromCreate(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Create(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	current, err := s.sessionStateRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, coreerrors.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)
	if err := s.sessionStateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	state := buildSessionState(sessionID, request, actorUserID)

	if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return s.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (s *StateService) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return coreerrors.ErrBadRequest
	}

	return s.sessionStateRepo.Delete(ctx, sessionID)
}

func (s *StateService) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	exists, err := s.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, coreerrors.ErrNotFound
	}

	return s.sessionStateRepo.GetByUserID(ctx, userID)
}

func (s *StateService) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return s.UpsertSessionState(ctx, actor, sessionID, types.UpsertSessionStateRequest{
		Revoke:        true,
		RevokedReason: reason,
	}, actorUserID)
}

func (s *StateService) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	return s.sessionStateRepo.GetRevoked(ctx)
}

func (s *StateService) BanUser(ctx context.Context, actor *models.Actor, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return s.UpsertUserState(ctx, actor, userID, types.UpsertUserStateRequest{
		Banned:       true,
		BannedUntil:  request.BannedUntil,
		BannedReason: request.Reason,
	}, actorUserID)
}

func (s *StateService) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
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
