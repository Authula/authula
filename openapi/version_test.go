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
	t.Parallel()

	type HealthResponse struct {
		Status string `json:"status"`
	}

	tests := []struct {
		name        string
		opts        []openapi.ServiceOption
		addOp       func(t *testing.T, svc openapi.OpenAPIService)
		wantVersion string
		wantPaths   []string
		wantSchemas []string
	}{
		{
			name:        "defaults to 3.0.3",
			wantVersion: "3.0.3",
		},
		{
			name: "explicit 3.0.3",
			opts: []openapi.ServiceOption{
				openapi.WithOpenAPIVersion("3.0.3"),
			},
			addOp: func(t *testing.T, svc openapi.OpenAPIService) {
				t.Helper()

				err := svc.AddOperation(
					http.MethodGet,
					"/api/test",
					openapi.WithResponseStatus(http.StatusOK, &struct {
						Message string `json:"message"`
					}{}),
				)
				require.NoError(t, err)
			},
			wantVersion: "3.0.3",
			wantPaths:   []string{"/api/test"},
		},
		{
			name: "3.1.0 spec version and basic operation",
			opts: []openapi.ServiceOption{
				openapi.WithOpenAPIVersion("3.1.0"),
				openapi.WithShortSchemaNames(),
			},
			addOp: func(t *testing.T, svc openapi.OpenAPIService) {
				t.Helper()

				err := svc.AddOperation(
					http.MethodGet,
					"/api/health",
					openapi.WithSummary("Health"),
					openapi.WithResponseStatus(http.StatusOK, &HealthResponse{}),
				)
				require.NoError(t, err)
			},
			wantVersion: "3.1.0",
			wantPaths:   []string{"/api/health"},
			wantSchemas: []string{"HealthResponse"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", tt.opts...)
			require.NoError(t, err)

			if tt.addOp != nil {
				tt.addOp(t, svc)
			}

			spec, err := svc.SpecJSON()

			require.NoError(t, err)

			var result map[string]any
			require.NoError(t, json.Unmarshal(spec, &result))
			assert.Equal(t, tt.wantVersion, result["openapi"])

			if len(tt.wantPaths) > 0 {
				paths, ok := result["paths"].(map[string]any)
				require.True(t, ok, "expected paths object in spec")
				for _, path := range tt.wantPaths {
					assert.Contains(t, paths, path)
				}
			}

			if len(tt.wantSchemas) > 0 {
				components, ok := result["components"].(map[string]any)
				require.True(t, ok, "expected components object in spec")
				schemas, ok := components["schemas"].(map[string]any)
				require.True(t, ok, "expected schemas object in spec")
				for _, name := range tt.wantSchemas {
					assert.Contains(t, schemas, name)
				}
			}
		})
	}
}

func TestNewOpenAPIService_RejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "rejects 3.2.0",
			version: "3.2.0",
		},
		{
			name:    "rejects Swagger 2.0",
			version: "2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", openapi.WithOpenAPIVersion(tt.version))

			assert.Nil(t, svc)
			assert.EqualError(t, err, "unsupported OpenAPI version: "+tt.version)
		})
	}
}
