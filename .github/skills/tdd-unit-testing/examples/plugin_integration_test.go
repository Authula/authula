package myplugin_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMyPluginRouteIntegration_CreateResource_Authenticated_Success(t *testing.T) {
	// Arrange: Create fixture and seed data
	f := newMyPluginFixture(t)
	f.SeedUser("test-creator", "creator@example.com")
	f.GrantPermissionToUser("test-creator", "resources.write")
	f.AuthenticateAs("test-creator")

	payload := map[string]any{"name": "My New Resource"}

	// Act: Perform HTTP request through the router
	w := f.JSONRequest(http.MethodPost, "/auth/my-plugin/resources", payload)

	// Assert: Check response code and data
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "My New Resource", response["name"])
}

func TestMyPluginRouteIntegration_GetResource_Unauthenticated_Returns401(t *testing.T) {
	// Arrange
	f := newMyPluginFixture(t)
	f.ApplyRBACMappingsForAllPluginRoutes("resources.read")

	// Act
	w := f.JSONRequest(http.MethodGet, "/auth/my-plugin/resources/123", nil)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMyPluginRouteIntegration_GetResource_Unauthorized_Returns403(t *testing.T) {
	// Arrange: Authenticated user lacks "resources.read"
	f := newMyPluginFixture(t)
	f.SeedUser("test-outsider", "outsider@example.com")
	f.AuthenticateAs("test-outsider")
	f.ApplyRBACMappingsForAllPluginRoutes("resources.read")

	// Act
	w := f.JSONRequest(http.MethodGet, "/auth/my-plugin/resources/123", nil)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
}
