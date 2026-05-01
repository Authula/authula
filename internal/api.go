package internal

import (
	"context"

	"github.com/Authula/authula/internal/types"
	"github.com/Authula/authula/internal/usecases"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

type CoreAPI interface {
	GetMe(ctx context.Context, userID string) (*types.GetMeResult, error)
	SignOut(ctx context.Context, userID string, sessionID *string, signOutAll *bool) (*types.SignOutResult, error)
}

type coreAPI struct {
	useCases *usecases.UseCases
}

func NewCoreAPI(logger models.Logger, userService services.UserService, sessionService services.SessionService) CoreAPI {
	useCases := BuildUseCases(logger, userService, sessionService)
	return &coreAPI{
		useCases: useCases,
	}
}

func (api *coreAPI) GetMe(ctx context.Context, userID string) (*types.GetMeResult, error) {
	return api.useCases.GetMeUseCase.GetMe(ctx, userID)
}

func (api *coreAPI) SignOut(ctx context.Context, userID string, sessionID *string, signOutAll *bool) (*types.SignOutResult, error) {
	return api.useCases.SignOutUseCase.SignOut(ctx, userID, sessionID, signOutAll)
}

func BuildUseCases(logger models.Logger, userService services.UserService, sessionService services.SessionService) *usecases.UseCases {
	return &usecases.UseCases{
		GetMeUseCase: &usecases.GetMeUseCase{
			Logger:         logger,
			UserService:    userService,
			SessionService: sessionService,
		},
		SignOutUseCase: &usecases.SignOutUseCase{
			Logger:         logger,
			SessionService: sessionService,
		},
	}
}
