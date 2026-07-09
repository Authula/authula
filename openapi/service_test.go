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
	tests := []struct {
		name        string
		title       string
		version     string
		description string
	}{
		{
			name:        "creates service with metadata",
			title:       "Test API",
			version:     "1.0.0",
			description: "A test API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := openapi.NewOpenAPIService(tt.title, tt.version, tt.description)
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
			assert.Equal(t, tt.title, info["title"])
			assert.Equal(t, tt.version, info["version"])
			assert.Equal(t, tt.description, info["description"])


		})
	}
}

func TestOpenAPIService_AddOperation(t *testing.T) {
	type healthResponse struct {
		Status string `json:"status"`
	}
	type getTodoRequest struct {
		ID string `path:"id"`
	}
	type getTodoResponse struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	type createTodoRequest struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	type createTodoResponse struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	type deleteRequest struct {
		ID string `path:"id"`
	}
	type messageResponse struct {
		Message string `json:"message"`
	}
	type titleRequest struct {
		Title string `json:"title"`
	}
	type idResponse struct {
		ID string `json:"id"`
	}

	tests := []struct {
		name   string
		method string
		path   string
		opts   []openapi.OperationOption
		check  func(t *testing.T, spec map[string]any)
	}{
		{
			name:   "adds a GET operation with response",
			method: http.MethodGet,
			path:   "/api/health",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Health check"),
				openapi.WithDescription("Returns the health status"),
				openapi.WithTags("System"),
				openapi.WithResponseStatus(http.StatusOK, &healthResponse{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths, ok := spec["paths"].(map[string]any)
				require.True(t, ok, "spec should contain paths")

				pathItem, ok := paths["/api/health"].(map[string]any)
				require.True(t, ok, "paths should contain /api/health")

				getOp, ok := pathItem["get"].(map[string]any)
				require.True(t, ok, "path should contain get operation")
				assert.Equal(t, "Health check", getOp["summary"])
				assert.Equal(t, "Returns the health status", getOp["description"])
				assert.Equal(t, []any{"System"}, getOp["tags"])
			},
		},
		{
			name:   "adds operation with path parameters",
			method: http.MethodGet,
			path:   "/api/todos/{id}",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Get todo by ID"),
				openapi.WithTags("Todos"),
				openapi.WithRequest(&getTodoRequest{}),
				openapi.WithResponseStatus(http.StatusOK, &getTodoResponse{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths := spec["paths"].(map[string]any)
				pathItem := paths["/api/todos/{id}"].(map[string]any)

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
			},
		},
		{
			name:   "adds a POST operation with request body",
			method: http.MethodPost,
			path:   "/api/todos",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Create a todo"),
				openapi.WithTags("Todos"),
				openapi.WithRequest(&createTodoRequest{}),
				openapi.WithResponseStatus(http.StatusCreated, &createTodoResponse{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths := spec["paths"].(map[string]any)
				pathItem := paths["/api/todos"].(map[string]any)
				postOp := pathItem["post"].(map[string]any)

				assert.Contains(t, postOp, "requestBody", "POST should have a request body")
				components := spec["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				assert.NotEmpty(t, schemas, "request/response types should be in components/schemas")
			},
		},
		{
			name:   "adds a DELETE operation with path parameter",
			method: http.MethodDelete,
			path:   "/api/todos/{id}",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Delete a todo"),
				openapi.WithTags("Todos"),
				openapi.WithRequest(&deleteRequest{}),
				openapi.WithResponseStatus(http.StatusNoContent, &struct{}{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths := spec["paths"].(map[string]any)
				pathItem := paths["/api/todos/{id}"].(map[string]any)
				assert.Contains(t, pathItem, "delete")
			},
		},
		{
			name:   "allows empty response for no-content operations",
			method: http.MethodDelete,
			path:   "/api/items/{id}",
			opts: []openapi.OperationOption{
				openapi.WithRequest(&deleteRequest{}),
				openapi.WithResponseStatus(http.StatusNoContent, &struct{}{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths := spec["paths"].(map[string]any)
				pathItem := paths["/api/items/{id}"].(map[string]any)
				assert.Contains(t, pathItem, "delete")
			},
		},
		{
			name:   "marks operation as deprecated",
			method: http.MethodGet,
			path:   "/api/old-endpoint",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Old endpoint"),
				openapi.WithDeprecated(true),
				openapi.WithResponseStatus(http.StatusOK, &messageResponse{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				paths := spec["paths"].(map[string]any)
				pathItem := paths["/api/old-endpoint"].(map[string]any)
				getOp := pathItem["get"].(map[string]any)
				assert.Equal(t, true, getOp["deprecated"])
			},
		},
		{
			name:   "sets operation ID",
			method: http.MethodPost,
			path:   "/api/todos",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Create a todo"),
				openapi.WithOperationID("createTodo"),
				openapi.WithRequest(&titleRequest{}),
				openapi.WithResponseStatus(http.StatusCreated, &idResponse{}),
			},
			check: func(t *testing.T, spec map[string]any) {
				postOp := spec["paths"].(map[string]any)["/api/todos"].(map[string]any)["post"].(map[string]any)
				assert.Equal(t, "createTodo", postOp["operationId"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "")
			require.NoError(t, err)

			err = svc.AddOperation(tt.method, tt.path, tt.opts...)
			require.NoError(t, err)

			spec, err := svc.SpecJSON()
			require.NoError(t, err)

			var result map[string]any
			err = json.Unmarshal(spec, &result)
			require.NoError(t, err)

			tt.check(t, result)
		})
	}
}

func TestOpenAPIService_SpecYAML(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		opts   []openapi.OperationOption
	}{
		{
			name:   "generates valid YAML spec",
			method: http.MethodGet,
			path:   "/api/health",
			opts: []openapi.OperationOption{
				openapi.WithResponseStatus(http.StatusOK, &struct {
					Status string `json:"status"`
				}{}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "")
			require.NoError(t, err)

			err = svc.AddOperation(tt.method, tt.path, tt.opts...)
			require.NoError(t, err)

			yaml, err := svc.SpecYAML()
			require.NoError(t, err)
			assert.Contains(t, string(yaml), "openapi: ")
			assert.Contains(t, string(yaml), "/api/health")
		})
	}
}
