package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/api-key/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodPost,
			"/api-keys",
			openapi.WithOperationID("createApiKey"),
			openapi.WithSummary("Create API key"),
			openapi.WithDescription("Creates a new API key for the specified owner."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.CreateApiKeyRequest{}),
			openapi.WithResponseStatus(http.StatusCreated, &types.CreateApiKeyResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/api-keys",
			openapi.WithOperationID("listApiKeys"),
			openapi.WithSummary("List API keys"),
			openapi.WithDescription("Lists API keys with pagination, optionally filtered by owner."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.ListApiKeysQuery{}),
			openapi.WithResponseStatus(http.StatusOK, &types.GetAllApiKeysResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/api-keys/{id}",
			openapi.WithOperationID("getApiKey"),
			openapi.WithSummary("Get API key by ID"),
			openapi.WithDescription("Retrieves an API key by its ID."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.ApiKeyID{}),
			openapi.WithResponseStatus(http.StatusOK, &types.GetApiKeyResponse{}),
		),
		svc.AddOperation(
			http.MethodPatch,
			"/api-keys/{id}",
			openapi.WithOperationID("updateApiKey"),
			openapi.WithSummary("Update API key"),
			openapi.WithDescription("Updates an API key's attributes."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.ApiKeyID{}),
			openapi.WithRequest(&types.UpdateApiKeyRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.UpdateApiKeyResponse{}),
		),
		svc.AddOperation(
			http.MethodDelete,
			"/api-keys/{id}",
			openapi.WithOperationID("deleteApiKey"),
			openapi.WithSummary("Delete API key"),
			openapi.WithDescription("Deletes an API key."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.ApiKeyID{}),
			openapi.WithResponseStatus(http.StatusOK, &types.DeleteApiKeyResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/api-keys/verify",
			openapi.WithOperationID("verifyApiKey"),
			openapi.WithSummary("Verify API key"),
			openapi.WithDescription("Verifies an API key by its raw value and returns the associated key details."),
			openapi.WithTags("API Keys"),
			openapi.WithRequest(&types.VerifyApiKeyRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.VerifyApiKeyResponse{}),
		),
	)
}
