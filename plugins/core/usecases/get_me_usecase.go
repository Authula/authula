package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/types"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type GetMeUseCase struct {
	Logger      models.Logger
	UserService services.UserService
}

func (uc *GetMeUseCase) GetMe(ctx context.Context, userID string) (*types.GetMeResult, error) {
	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		uc.Logger.Error("failed to get user by ID: %v", err)
		return nil, err
	}

	return &types.GetMeResult{
		User: user,
	}, nil
}
