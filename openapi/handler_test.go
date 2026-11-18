package openapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
	openapitests "github.com/Authula/authula/openapi/tests"
)

func TestSpecHandlers_ServeSpec(t *testing.T) {
	t.Parallel()

	newService := func(t *testing.T, addOperation bool) openapi.OpenAPIService {
		t.Helper()

		svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "")
		require.NoError(t, err)

		if !addOperation {
			return svc
		}

		err = svc.AddOperation(
			http.MethodGet,
			"/api/v1/health",
			openapi.WithResponseStatus(http.StatusOK, &struct {
				Status string `json:"status"`
			}{}),
		)
		require.NoError(t, err)

		return svc
	}

	tests := []struct {
		name            string
		newHandler      func(openapi.OpenAPIService) http.HandlerFunc
		addOperation    bool
		wantContentType string
		assertBody      func(t *testing.T, body []byte)
	}{
		{
			name:            "serves valid JSON spec",
			newHandler:      openapi.NewOpenAPISpecHandler,
			addOperation:    true,
			wantContentType: "application/json",
			assertBody: func(t *testing.T, body []byte) {
				var result map[string]any
				require.NoError(t, json.Unmarshal(body, &result))
				assert.Equal(t, "3.0.3", result["openapi"])
				assert.Contains(t, result, "paths")
			},
		},
		{
			name:            "serves valid YAML spec",
			newHandler:      openapi.NewOpenAPISpecYAMLHandler,
			addOperation:    true,
			wantContentType: "text/vnd.yaml",
			assertBody: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "openapi: ")
			},
		},
		{
			name:            "serves empty JSON spec",
			newHandler:      openapi.NewOpenAPISpecHandler,
			wantContentType: "application/json",
			assertBody: func(t *testing.T, body []byte) {
				assert.True(t, json.Valid(body), "empty spec should still be valid JSON")
			},
		},
		{
			name:            "serves empty YAML spec",
			newHandler:      openapi.NewOpenAPISpecYAMLHandler,
			wantContentType: "text/vnd.yaml",
			assertBody: func(t *testing.T, body []byte) {
				assert.NotEmpty(t, body, "empty spec should still render YAML")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newService(t, tt.addOperation)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi", nil)
			rec := httptest.NewRecorder()

			tt.newHandler(svc)(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.wantContentType, rec.Header().Get("Content-Type"))
			tt.assertBody(t, rec.Body.Bytes())
		})
	}
}

func TestSpecHandlers_ReturnInternalServerErrorWhenMarshalFails(t *testing.T) {
	t.Parallel()

	errMarshal := errors.New("marshal failed")

	tests := []struct {
		name       string
		newHandler func(openapi.OpenAPIService) http.HandlerFunc
		setup      func(*openapitests.MockOpenAPIService)
	}{
		{
			name:       "JSON handler returns 500 when SpecJSON fails",
			newHandler: openapi.NewOpenAPISpecHandler,
			setup: func(m *openapitests.MockOpenAPIService) {
				m.On("SpecJSON").Return([]byte(nil), errMarshal).Once()
			},
		},
		{
			name:       "YAML handler returns 500 when SpecYAML fails",
			newHandler: openapi.NewOpenAPISpecYAMLHandler,
			setup: func(m *openapitests.MockOpenAPIService) {
				m.On("SpecYAML").Return([]byte(nil), errMarshal).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := new(openapitests.MockOpenAPIService)
			tt.setup(svc)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi", nil)
			rec := httptest.NewRecorder()

			tt.newHandler(svc)(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Contains(t, rec.Body.String(), "Failed to marshal OpenAPI spec: marshal failed")
			assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
			svc.AssertExpectations(t)
		})
	}
}
