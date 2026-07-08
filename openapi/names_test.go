package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

func TestShortSchemaNames(t *testing.T) {
	t.Run("schema name is just the struct name without package prefix", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080",
			openapi.WithShortSchemaNames())
		require.NoError(t, err)

		type HealthResponse struct {
			Status string `json:"status"`
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/health",
			openapi.WithResponseStatus(http.StatusOK, &HealthResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		components := result["components"].(map[string]any)
		schemas := components["schemas"].(map[string]any)

		_, hasShortName := schemas["HealthResponse"]
		_, hasLongName := schemas["TypesHealthResponse"]
		_, hasPkgName := schemas["OpenapiTestHealthResponse"]

		assert.True(t, hasShortName, "schema should be named 'HealthResponse' not package-prefixed")
		assert.False(t, hasLongName, "should NOT have 'TypesHealthResponse'")
		assert.False(t, hasPkgName, "should NOT have 'OpenapiTestHealthResponse'")
	})

	t.Run("default (no option) still uses package-prefixed names", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type MyResponse struct {
			Message string `json:"message"`
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/test",
			openapi.WithResponseStatus(http.StatusOK, &MyResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		schemas := result["components"].(map[string]any)["schemas"].(map[string]any)
		assert.Contains(t, schemas, "OpenapiTestMyResponse",
			"default should include package prefix in schema name")
	})
}

func TestRegisterSchema(t *testing.T) {
	t.Run("registers a type under a custom schema name", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		type MyType struct {
			ID string `json:"id"`
		}

		err = svc.RegisterSchema("CustomName", &MyType{})
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)

		schemas := result["components"].(map[string]any)["schemas"].(map[string]any)
		assert.Contains(t, schemas, "CustomName")
	})
}
