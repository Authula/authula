package apikey

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/handlers"
	"github.com/Authula/authula/plugins/api-key/usecases"
)

func Routes(useCases *usecases.UseCases) []models.Route {
	createApiKeyHandler := &handlers.CreateApiKeyHandler{UseCases: useCases}
	getAllApiKeysHandler := &handlers.GetAllApiKeysHandler{UseCases: useCases}
	getApiKeyHandler := &handlers.GetApiKeyHandler{UseCases: useCases}
	updateApiKeyHandler := &handlers.UpdateApiKeyHandler{UseCases: useCases}
	deleteApiKeyHandler := &handlers.DeleteApiKeyHandler{UseCases: useCases}
	verifyApiKeyHandler := &handlers.VerifyApiKeyHandler{UseCases: useCases}

	return []models.Route{
		{
			Method: http.MethodPost,
			Path:   "/api-keys",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: createApiKeyHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/api-keys",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllApiKeysHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/api-keys/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getApiKeyHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/api-keys/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: updateApiKeyHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/api-keys/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: deleteApiKeyHandler.Handle(),
		},
		{
			Method: http.MethodPost,
			Path:   "/api-keys/verify",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: verifyApiKeyHandler.Handle(),
		},
	}
}
