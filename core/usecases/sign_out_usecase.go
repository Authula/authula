package usecases

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

type SignOutUseCase struct {
	Logger         models.Logger
	SessionService services.SessionService
}

func (uc *SignOutUseCase) SignOut(
	ctx context.Context,
	userID string,
	currentSessionID *string,
	requestedSessionID *string,
	signOutAll *bool,
) (*types.SignOutResult, error) {
	if signOutAll != nil && *signOutAll {
		if err := uc.SessionService.DeleteAllByUserID(ctx, userID); err != nil {
			uc.Logger.Error("failed to delete all sessions for user", "error", err, "user_id", userID)
			return nil, err
		}
		return &types.SignOutResult{Message: "signed out from all sessions", ClearedCurrentSession: true}, nil
	}

	if requestedSessionID != nil && *requestedSessionID != "" {
		return uc.signOutRequestedSession(ctx, userID, currentSessionID, *requestedSessionID)
	}

	if currentSessionID != nil && *currentSessionID != "" {
		if err := uc.SessionService.Delete(ctx, *currentSessionID); err != nil {
			uc.Logger.Error("failed to delete current session", "error", err, "session_id", *currentSessionID)
			return nil, err
		}
		return &types.SignOutResult{Message: "signed out", ClearedCurrentSession: true}, nil
	}

	// The caller is authenticated but nothing identifies a session to revoke so
	// therefore guessing would risk revoking the wrong session, so refuse.
	return nil, coreerrors.ErrNoSessionToSignOut
}

func (uc *SignOutUseCase) signOutRequestedSession(
	ctx context.Context,
	userID string,
	currentSessionID *string,
	requestedSessionID string,
) (*types.SignOutResult, error) {
	session, err := uc.SessionService.GetByID(ctx, requestedSessionID)
	if err != nil {
		uc.Logger.Error("failed to get session", "error", err, "session_id", requestedSessionID)
		return nil, err
	}
	if session == nil {
		return nil, coreerrors.ErrNotFound
	}
	if session.UserID != userID {
		uc.Logger.Warn("user attempted to sign out a session they do not own", "user_id", userID, "session_id", requestedSessionID)
		return nil, coreerrors.ErrForbidden
	}

	if err := uc.SessionService.Delete(ctx, requestedSessionID); err != nil {
		uc.Logger.Error("failed to delete session", "error", err, "session_id", requestedSessionID)
		return nil, err
	}

	clearedCurrent := currentSessionID != nil && *currentSessionID == requestedSessionID

	return &types.SignOutResult{Message: "signed out", ClearedCurrentSession: clearedCurrent}, nil
}
