package usecases

import (
	"context"
	"errors"
	"sync"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

type GetMeUseCase struct {
	Logger         models.Logger
	UserService    services.UserService
	SessionService services.SessionService
}

func (uc *GetMeUseCase) GetMe(ctx context.Context, userID string, sessionID *string) (*types.GetMeResult, error) {
	var (
		wg      sync.WaitGroup
		user    *models.User
		session *models.Session
		userErr error
		sessErr error
	)

	wg.Go(func() {
		user, userErr = uc.UserService.GetByID(ctx, userID)
	})

	wg.Go(func() {
		if sessionID == nil || *sessionID == "" {
			return
		}
		session, sessErr = uc.SessionService.GetByID(ctx, *sessionID)
	})

	wg.Wait()

	if userErr != nil {
		uc.Logger.Error("failed to get user by ID", "error", userErr, "user_id", userID)
		return nil, userErr
	}
	if user == nil {
		return nil, coreerrors.ErrUnauthorized
	}

	// Stateless credentials such as API keys have no session to resolve.
	if sessionID == nil || *sessionID == "" {
		return &types.GetMeResult{User: user}, nil
	}

	if sessErr != nil {
		if errors.Is(sessErr, coreerrors.ErrSessionNotFound) {
			return nil, coreerrors.ErrUnauthorized
		}
		uc.Logger.Error("failed to get session", "error", sessErr, "session_id", *sessionID)
		return nil, sessErr
	}
	if session == nil || session.UserID != userID {
		return nil, coreerrors.ErrUnauthorized
	}

	return &types.GetMeResult{
		User:    user,
		Session: session,
	}, nil
}
