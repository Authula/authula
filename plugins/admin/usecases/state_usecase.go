package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type stateUseCase struct {
	userStateRepo     repositories.UserStateRepository
	sessionStateRepo  repositories.SessionStateRepository
	impersonationRepo repositories.ImpersonationRepository
}

func NewStateUseCase(userStateRepo repositories.UserStateRepository, sessionStateRepo repositories.SessionStateRepository, impersonationRepo repositories.ImpersonationRepository) StateUseCase {
	return &stateUseCase{userStateRepo: userStateRepo, sessionStateRepo: sessionStateRepo, impersonationRepo: impersonationRepo}
}

func (u *stateUseCase) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	return u.userStateRepo.GetByUserID(ctx, userID)
}

func (u *stateUseCase) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	exists, err := u.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found")
	}

	now := time.Now().UTC()
	state := &types.AdminUserState{
		UserID:   userID,
		IsBanned: request.IsBanned,
	}
	if request.IsBanned {
		state.BannedAt = &now
		state.BannedUntil = request.BannedUntil
		state.BannedReason = request.BannedReason
		state.BannedByUserID = actorUserID
	}

	if err := u.userStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return u.userStateRepo.GetByUserID(ctx, userID)
}

func (u *stateUseCase) DeleteUserState(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	return u.userStateRepo.Delete(ctx, userID)
}

func (u *stateUseCase) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return u.userStateRepo.GetBanned(ctx)
}

func (u *stateUseCase) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	return u.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (u *stateUseCase) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}

	exists, err := u.sessionStateRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("session not found")
	}

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

	if err := u.sessionStateRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}

	return u.sessionStateRepo.GetBySessionID(ctx, sessionID)
}

func (u *stateUseCase) DeleteSessionState(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	return u.sessionStateRepo.Delete(ctx, sessionID)
}

func (u *stateUseCase) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	exists, err := u.impersonationRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found")
	}

	return u.sessionStateRepo.GetByUserID(ctx, userID)
}

func (u *stateUseCase) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return u.UpsertSessionState(ctx, sessionID, types.UpsertSessionStateRequest{
		Revoke:        true,
		RevokedReason: reason,
	}, actorUserID)
}

func (u *stateUseCase) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return u.sessionStateRepo.GetRevoked(ctx)
}

func (u *stateUseCase) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.UpsertUserState(ctx, userID, types.UpsertUserStateRequest{
		IsBanned:     true,
		BannedUntil:  request.BannedUntil,
		BannedReason: request.Reason,
	}, actorUserID)
}

func (u *stateUseCase) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.UpsertUserState(ctx, userID, types.UpsertUserStateRequest{IsBanned: false}, nil)
}
