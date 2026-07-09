package internal

import (
	"net/http"

	"github.com/Authula/authula/internal/handlers"
	"github.com/Authula/authula/middleware"
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
			Method: "GET",
			Path:   "/me",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: getMeHandler.Handle(),
		},
		{
			Method: "POST",
			Path:   "/sign-out",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: signOutHandler.Handle(),
		},
	}
}
