package internal

import (
	"github.com/Authula/authula/internal/handlers"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

func CoreRoutes(logger models.Logger, userService services.UserService, sessionService services.SessionService) []models.Route {
	useCases := BuildUseCases(logger, userService, sessionService)

	getMeHandler := &handlers.GetMeHandler{
		UseCase: useCases.GetMeUseCase,
	}

	signOutHandler := &handlers.SignOutHandler{
		UseCase: useCases.SignOutUseCase,
	}

	return []models.Route{
		{
			Method:  "GET",
			Path:    "/me",
			Handler: getMeHandler.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/sign-out",
			Handler: signOutHandler.Handler(),
		},
	}
}
