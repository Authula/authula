package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

func TestOpenAPIVersion(t *testing.T) {
	t.Run("defaults to 3.0.3", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)
		assert.Equal(t, "3.0.3", result["openapi"])
	})

	t.Run("explicit 3.0.3", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080",
			openapi.WithOpenAPIVersion("3.0.3"))
		require.NoError(t, err)

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/test",
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
		assert.Equal(t, "3.0.3", result["openapi"])
		assert.Contains(t, result["paths"].(map[string]any), "/api/v1/test")
	})

	t.Run("3.1.0 spec version and basic operation", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", "http://localhost:8080",
			openapi.WithOpenAPIVersion("3.1.0"),
			openapi.WithShortSchemaNames())
		require.NoError(t, err)

		type HealthResponse struct {
			Status string `json:"status"`
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/health",
			openapi.WithSummary("Health"),
			openapi.WithResponseStatus(http.StatusOK, &HealthResponse{}),
		)
		require.NoError(t, err)

		spec, err := svc.SpecJSON()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(spec, &result)
		require.NoError(t, err)
		assert.Equal(t, "3.1.0", result["openapi"])

		paths := result["paths"].(map[string]any)
		assert.Contains(t, paths, "/api/v1/health")

		schemas := result["components"].(map[string]any)["schemas"].(map[string]any)
		assert.Contains(t, schemas, "HealthResponse", "short names should work with 3.1")
	})
}
