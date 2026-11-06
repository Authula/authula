package openapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
	oauth2openapi "github.com/Authula/authula/plugins/oauth2/openapi"
)

func specDocument(t *testing.T, svc openapi.OpenAPIService) map[string]any {
	t.Helper()

	spec, err := svc.SpecJSON()
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(spec, &doc))

	return doc
}

func operation(t *testing.T, doc map[string]any, method, path string) map[string]any {
	t.Helper()

	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok, "spec should contain paths")

	pathItem, ok := paths[path].(map[string]any)
	require.True(t, ok, "paths should contain %s", path)

	op, ok := pathItem[strings.ToLower(method)].(map[string]any)
	require.True(t, ok, "%s %s should be documented", method, path)

	return op
}

// resolveSchema follows a local $ref to the schema it points at.
func resolveSchema(t *testing.T, doc map[string]any, media map[string]any) map[string]any {
	t.Helper()

	schema, ok := media["schema"].(map[string]any)
	require.True(t, ok, "media type should declare a schema")

	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}

	const prefix = "#/components/schemas/"
	require.True(t, strings.HasPrefix(ref, prefix), "unexpected schema ref %q", ref)

	components, ok := doc["components"].(map[string]any)
	require.True(t, ok, "spec should contain components")
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "components should contain schemas")

	resolved, ok := schemas[strings.TrimPrefix(ref, prefix)].(map[string]any)
	require.True(t, ok, "schema %q should be defined", ref)

	return resolved
}

func TestRegisterOpenAPIDocs_FormPostCallback(t *testing.T) {
	t.Parallel()

	svc, err := openapi.NewOpenAPIService("Test API", "1.0.0", "")
	require.NoError(t, err)
	require.NoError(t, oauth2openapi.RegisterOpenAPIDocs(svc))

	doc := specDocument(t, svc)
	post := operation(t, doc, http.MethodPost, "/oauth2/callback/{provider}")

	requestBody, ok := post["requestBody"].(map[string]any)
	require.True(t, ok, "the form post callback must document a request body")

	content, ok := requestBody["content"].(map[string]any)
	require.True(t, ok, "the request body must declare a content type")

	assert.Contains(t, content, "application/x-www-form-urlencoded")
	assert.NotContains(t, content, "application/json", "a form post must not be documented as JSON")

	media, ok := content["application/x-www-form-urlencoded"].(map[string]any)
	require.True(t, ok)

	schema := resolveSchema(t, doc, media)

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "the form schema must expose properties")

	for _, field := range []string{"code", "state", "error", "user"} {
		assert.Contains(t, properties, field, "the form schema must expose %q", field)
	}

	rawRequired, ok := schema["required"].([]any)
	require.True(t, ok, "the form schema must declare required fields")

	required := make([]string, 0, len(rawRequired))
	for _, value := range rawRequired {
		field, ok := value.(string)
		require.True(t, ok, "required entries must be strings")
		required = append(required, field)
	}
	assert.ElementsMatch(t, []string{"code", "state"}, required)
}
