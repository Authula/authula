package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

func TestNewOpenAPIService(t *testing.T) {
	t.Run("creates service with metadata", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "A test API", "http://localhost:8080")
		require.NoError(t, err)
		require.NotNil(t, svc)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		assert.Equal(t, "3.0.3", result["openapi"])

		info, ok := result["info"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Test API", info["title"])
		assert.Equal(t, "1.0.0", info["version"])
		assert.Equal(t, "A test API", info["description"])

		servers, ok := result["servers"].([]any)
		require.True(t, ok)
		require.Len(t, servers, 1)
		server := servers[0].(map[string]any)
		assert.Equal(t, "http://localhost:8080", server["url"])
	})
}

func TestOpenAPIService_AddOperation_Get(t *testing.T) {
	t.Run("adds a GET operation with response", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type healthResponse struct {
			Status string `json:"status"`
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/health",
			openapi.WithSummary("Health check"),
			openapi.WithDescription("Returns the health status"),
			openapi.WithTags("System"),
			openapi.WithResponseStatus(http.StatusOK, &healthResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		paths, ok := result["paths"].(map[string]any)
		require.True(t, ok, "spec should contain paths")

		pathItem, ok := paths["/api/v1/health"].(map[string]any)
		require.True(t, ok, "paths should contain /api/v1/health")

		getOp, ok := pathItem["get"].(map[string]any)
		require.True(t, ok, "path should contain get operation")
		assert.Equal(t, "Health check", getOp["summary"])
		assert.Equal(t, "Returns the health status", getOp["description"])
		assert.Equal(t, []any{"System"}, getOp["tags"])
	})

	t.Run("adds operation with path parameters", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type getGuideRequest struct {
			ID string `path:"id"`
		}
		type getGuideResponse struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/guides/{id}",
			openapi.WithSummary("Get guide by ID"),
			openapi.WithTags("Guides"),
			openapi.WithRequest(&getGuideRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &getGuideResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		paths := result["paths"].(map[string]any)
		pathItem := paths["/api/v1/guides/{id}"].(map[string]any)

		getOp := pathItem["get"].(map[string]any)
		assert.Contains(t, getOp, "parameters", "path params should be extracted from struct tags")

		params := getOp["parameters"].([]any)
		foundPathParam := false
		for _, p := range params {
			param := p.(map[string]any)
			if param["name"] == "id" && param["in"] == "path" {
				foundPathParam = true
				assert.Equal(t, true, param["required"])
			}
		}
		assert.True(t, foundPathParam, "path parameter 'id' should be present")
	})
}

func TestOpenAPIService_AddOperation_Post(t *testing.T) {
	t.Run("adds a POST operation with request body", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type createGuideRequest struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		type createGuideResponse struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}

		err = svc.AddOperation(
			http.MethodPost,
			"/api/v1/guides",
			openapi.WithSummary("Create a guide"),
			openapi.WithTags("Guides"),
			openapi.WithRequest(&createGuideRequest{}),
			openapi.WithResponseStatus(http.StatusCreated, &createGuideResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		paths := result["paths"].(map[string]any)
		pathItem := paths["/api/v1/guides"].(map[string]any)
		postOp := pathItem["post"].(map[string]any)

		assert.Contains(t, postOp, "requestBody", "POST should have a request body")
		components := result["components"].(map[string]any)
		schemas := components["schemas"].(map[string]any)
		assert.NotEmpty(t, schemas, "request/response types should be in components/schemas")
	})
}

func TestOpenAPIService_AddOperation_Delete(t *testing.T) {
	t.Run("adds a DELETE operation with path parameter", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type deleteRequest struct {
			ID string `path:"id"`
		}

		err = svc.AddOperation(
			http.MethodDelete,
			"/api/v1/guides/{id}",
			openapi.WithSummary("Delete a guide"),
			openapi.WithTags("Guides"),
			openapi.WithRequest(&deleteRequest{}),
			openapi.WithResponseStatus(http.StatusNoContent, &struct{}{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		paths := result["paths"].(map[string]any)
		pathItem := paths["/api/v1/guides/{id}"].(map[string]any)
		assert.Contains(t, pathItem, "delete")
	})

	t.Run("allows empty response for no-content operations", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type deleteRequest struct {
			ID string `path:"id"`
		}

		err = svc.AddOperation(
			http.MethodDelete,
			"/api/v1/items/{id}",
			openapi.WithRequest(&deleteRequest{}),
			openapi.WithResponseStatus(http.StatusNoContent, &struct{}{}),
		)
		require.NoError(t, err)
	})
}

func TestOpenAPIService_SpecYAML(t *testing.T) {
	svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
	require.NoError(t, err)

	err = svc.AddOperation(
		http.MethodGet,
		"/api/v1/health",
		openapi.WithResponseStatus(http.StatusOK, &struct {
			Status string `json:"status"`
		}{}),
	)
	require.NoError(t, err)

	yaml, err := svc.SpecYAML()
	require.NoError(t, err)
	assert.Contains(t, string(yaml), "openapi: ")
	assert.Contains(t, string(yaml), "/api/v1/health")
}

func TestOpenAPIService_Deprecated(t *testing.T) {
	svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
	require.NoError(t, err)

	err = svc.AddOperation(
		http.MethodGet,
		"/api/v1/old-endpoint",
		openapi.WithSummary("Old endpoint"),
		openapi.WithDeprecated(true),
		openapi.WithResponseStatus(http.StatusOK, &struct {
			Message string `json:"message"`
		}{}),
	)
	require.NoError(t, err)

	spec, err := svc.SpecJSON()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(spec, &result)
	require.NoError(t, err)

	paths := result["paths"].(map[string]any)
	pathItem := paths["/api/v1/old-endpoint"].(map[string]any)
	getOp := pathItem["get"].(map[string]any)
	assert.Equal(t, true, getOp["deprecated"])
}

func TestOpenAPIService_OperationID(t *testing.T) {
	svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
	require.NoError(t, err)

	err = svc.AddOperation(
		http.MethodPost,
		"/api/v1/guides",
		openapi.WithSummary("Create a guide"),
		openapi.WithOperationID("createGuide"),
		openapi.WithRequest(&struct {
			Title string `json:"title"`
		}{}),
		openapi.WithResponseStatus(http.StatusCreated, &struct {
			ID string `json:"id"`
		}{}),
	)
	require.NoError(t, err)

	spec, err := svc.SpecJSON()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(spec, &result)
	require.NoError(t, err)

	postOp := result["paths"].(map[string]any)["/api/v1/guides"].(map[string]any)["post"].(map[string]any)
	assert.Equal(t, "createGuide", postOp["operationId"])
}
