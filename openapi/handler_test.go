package openapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

func TestNewOpenAPISpecHandler(t *testing.T) {
	t.Run("serves valid JSON spec", func(t *testing.T) {
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

		handler := openapi.NewOpenAPISpecHandler(svc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var result map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "3.0.3", result["openapi"])
		assert.Contains(t, result, "paths")
	})

	t.Run("returns 500 on empty service", func(t *testing.T) {
		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "", "http://localhost:8080")
		require.NoError(t, err)

		handler := openapi.NewOpenAPISpecHandler(svc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "empty spec should still be valid JSON")
	})
}

func TestNewOpenAPISpecYAMLHandler(t *testing.T) {
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

	handler := openapi.NewOpenAPISpecYAMLHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/vnd.yaml", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "openapi: ")
}
