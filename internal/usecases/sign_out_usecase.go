package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/internal/types"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type SignOutUseCase struct {
	Logger         models.Logger
	SessionService services.SessionService
}

func (uc *SignOutUseCase) SignOut(
	ctx context.Context,
	userID string,
	sessionID *string,
	signOutAll bool,
) (*types.SignOutResult, error) {
	// If a specific session ID is provided, delete only that session
	if sessionID != nil && *sessionID != "" {
		if err := uc.SessionService.Delete(ctx, *sessionID); err != nil {
			uc.Logger.Error("failed to delete session", "error", err, "session_id", *sessionID)
			return nil, err
		}
		return &types.SignOutResult{Message: "sign-out successful"}, nil
	}

	// If signOutAll is true, delete all sessions for the user
	if signOutAll {
		if err := uc.SessionService.DeleteAllByUserID(ctx, userID); err != nil {
			uc.Logger.Error("failed to delete all sessions for user", "error", err, "user_id", userID)
			return nil, err
		}
		return &types.SignOutResult{Message: "signed out from all sessions"}, nil
	}

	// Default: delete all sessions (sign out everywhere)
	if err := uc.SessionService.DeleteAllByUserID(ctx, userID); err != nil {
		uc.Logger.Error("failed to delete all sessions for user", "error", err, "user_id", userID)
		return nil, err
	}

	return &types.SignOutResult{Message: "sign-out successful"}, nil
}
