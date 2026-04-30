package apikey

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/handlers"
)

func Routes(api *API) []models.Route {
	createHandler := &handlers.CreateApiKeyHandler{Service: api.service}
	verifyHandler := &handlers.VerifyApiKeyHandler{Service: api.service}
	getAllHandler := &handlers.GetAllApiKeysHandler{Service: api.service}
	getHandler := &handlers.GetApiKeyHandler{Service: api.service}
	updateHandler := &handlers.UpdateApiKeyHandler{Service: api.service}
	deleteHandler := &handlers.DeleteApiKeyHandler{Service: api.service}

	return []models.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api-keys",
			Handler: createHandler.Handle(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api-keys/verify",
			Handler: verifyHandler.Handle(),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api-keys",
			Handler: getAllHandler.Handle(),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api-keys/{id}",
			Handler: getHandler.Handle(),
		},
		{
			Method:  http.MethodPatch,
			Path:    "/api-keys/{id}",
			Handler: updateHandler.Handle(),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api-keys/{id}",
			Handler: deleteHandler.Handle(),
		},
	}
}
