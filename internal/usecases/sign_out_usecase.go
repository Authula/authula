package usecases

import (
	"context"

	"github.com/Authula/authula/internal/types"
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
	sessionID *string,
	signOutAll *bool,
) (*types.SignOutResult, error) {
	// If a specific session ID is provided, delete only that session
	if sessionID != nil && *sessionID != "" {
		if err := uc.SessionService.Delete(ctx, *sessionID); err != nil {
			uc.Logger.Error("failed to delete session", "error", err, "session_id", *sessionID)
			return nil, err
		}
		return &types.SignOutResult{Message: "signed out"}, nil
	}

	// If signOutAll is true, delete all sessions for the user
	if signOutAll != nil && *signOutAll {
		if err := uc.SessionService.DeleteAllByUserID(ctx, userID); err != nil {
			uc.Logger.Error("failed to delete all sessions for user", "error", err, "user_id", userID)
			return nil, err
		}
		return &types.SignOutResult{Message: "signed out from all sessions"}, nil
	}

	// Else fallback to deleting the most recent session for the user
	if mostRecentSession, err := uc.SessionService.GetByUserID(ctx, userID); err != nil {
		uc.Logger.Error("failed to get session for user", "error", err, "user_id", userID)
		return nil, err
	} else if mostRecentSession != nil {
		if err := uc.SessionService.Delete(ctx, mostRecentSession.ID); err != nil {
			uc.Logger.Error("failed to delete session", "error", err, "session_id", mostRecentSession.ID)
			return nil, err
		}

		return &types.SignOutResult{Message: "signed out"}, nil
	}

	return &types.SignOutResult{Message: "no active session found"}, nil
}
