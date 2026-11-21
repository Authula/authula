package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

func TestRegisterOpenAPIDocs_TokenEndpointFormBody(t *testing.T) {
	t.Parallel()

	svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "")
	require.NoError(t, err)
	require.NoError(t, RegisterOpenAPIDocs(svc))

	data, err := svc.SpecJSON()
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(data, &spec))

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)

	tokenPath, ok := paths["/oauth2/token"].(map[string]any)
	require.True(t, ok)

	operation, ok := tokenPath["post"].(map[string]any)
	require.True(t, ok)

	requestBody, ok := operation["requestBody"].(map[string]any)
	require.True(t, ok)

	content, ok := requestBody["content"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, content, "application/x-www-form-urlencoded")

	_, hasParameters := operation["parameters"]
	require.False(t, hasParameters, "form body fields must not be emitted as query parameters")

	responses, ok := operation["responses"].(map[string]any)
	require.True(t, ok)
	for _, status := range []string{"200", "400", "401", "500"} {
		require.Contains(t, responses, status)
	}

	internalError, ok := responses["500"].(map[string]any)
	require.True(t, ok)
	internalContent, ok := internalError["content"].(map[string]any)
	require.True(t, ok)
	internalJSON, ok := internalContent["application/json"].(map[string]any)
	require.True(t, ok)
	internalSchema, ok := internalJSON["schema"].(map[string]any)
	require.True(t, ok)
	internalRef, ok := internalSchema["$ref"].(string)
	require.True(t, ok)
	require.Contains(t, internalRef, "TokenErrorResponse")
}
