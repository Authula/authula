package admin

import (
	adminhandlers "github.com/GoBetterAuth/go-better-auth/internal/admin/handlers"
	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func GetRoutes(config *models.Config, configManager models.ConfigManager, authService *auth.Service, basePath string, middleware *models.ApiMiddleware) []models.CustomRoute {
	updateConfigHandler := &adminhandlers.AdminUpdateConfigHandler{
		ConfigManager: configManager,
	}

	getConfigHandler := &adminhandlers.AdminGetConfigHandler{
		ConfigManager: configManager,
	}

	return []models.CustomRoute{
		{
			Method: "GET",
			Path:   "/admin/config",
			Middleware: []models.CustomRouteMiddleware{
				middleware.AdminAuth(),
			},
			Handler: getConfigHandler.Handler(),
		},
		{
			Method: "PATCH",
			Path:   "/admin/config",
			Middleware: []models.CustomRouteMiddleware{
				middleware.AdminAuth(),
			},
			Handler: updateConfigHandler.Handler(),
		},
	}
}
