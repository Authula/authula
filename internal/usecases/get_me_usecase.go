package usecases

import (
	"context"
	"sync"

	"github.com/GoBetterAuth/go-better-auth/internal/types"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type GetMeUseCase struct {
	Logger         models.Logger
	UserService    services.UserService
	SessionService services.SessionService
}

func (uc *GetMeUseCase) GetMe(ctx context.Context, userID string) (*types.GetMeResult, error) {
	var (
		wg      sync.WaitGroup
		user    *models.User
		session *models.Session
		userErr error
		sessErr error
	)

	wg.Go(func() {
		user, userErr = uc.UserService.GetByID(ctx, userID)
		if userErr != nil {
			uc.Logger.Error("failed to get user by ID: %v", userErr)
		}
	})

	wg.Go(func() {
		session, sessErr = uc.SessionService.GetByUserID(ctx, userID)
		if sessErr != nil {
			uc.Logger.Error("failed to get session by user ID: %v", sessErr)
		}
	})

	wg.Wait()

	if userErr != nil {
		return nil, userErr
	}
	if sessErr != nil {
		return nil, sessErr
	}

	return &types.GetMeResult{
		User:    user,
		Session: session,
	}, nil
}
