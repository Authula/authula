package core

import (
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/handlers"
)

func Routes(plugin *CorePlugin) []models.Route {
	useCases := BuildUseCases(plugin)

	healthHandler := &handlers.HealthHandler{}

	getMeHandler := &handlers.GetMeHandler{
		UseCase: useCases.GetMeUseCase,
	}

	signOutHandler := &handlers.SignOutHandler{
		UseCase: useCases.SignOutUseCase,
	}

	return []models.Route{
		{
			Method:  "GET",
			Path:    "/health",
			Handler: healthHandler.Handler(),
		},
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
