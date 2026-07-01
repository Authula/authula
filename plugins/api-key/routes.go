package apikey

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/handlers"
)

func Routes(api *API) []models.Route {
	createApiKeyHandler := &handlers.CreateApiKeyHandler{Service: api.service}
	getAllApiKeysHandler := &handlers.GetAllApiKeysHandler{Service: api.service}
	getApiKeyHandler := &handlers.GetApiKeyHandler{Service: api.service}
	updateApiKeyHandler := &handlers.UpdateApiKeyHandler{Service: api.service}
	deleteApiKeyHandler := &handlers.DeleteApiKeyHandler{Service: api.service}
	verifyApiKeyHandler := &handlers.VerifyApiKeyHandler{Service: api.service}

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
