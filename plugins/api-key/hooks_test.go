package apikey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

type mockRateLimitService struct {
	mock.Mock
}

func (m *mockRateLimitService) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	args := m.Called(ctx, key, window, maxRequests)
	return args.Bool(0), args.Int(1), args.Get(2).(time.Time), args.Error(3)
}

func TestApiKeyPluginHooks_MatchConfiguredHeaderOnly(t *testing.T) {
	t.Parallel()

	plugin := &ApiKeyPlugin{config: types.ApiKeyPluginConfig{Header: "X-Test-API-Key"}}
	plugin.config.ApplyDefaults()
	plugin.pluginCtx = &models.PluginContext{}

	hooks := plugin.buildHooks()
	require.Len(t, hooks, 1)

	reqCtx := &models.RequestContext{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	assert.False(t, hooks[0].Matcher(reqCtx))
	reqCtx.Request.Header.Set("X-Test-API-Key", "abc")
	assert.True(t, hooks[0].Matcher(reqCtx))
}

func TestApiKeyPluginHook_InvalidKey_ReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	service := &apiKeyTests.MockApiKeyService{}
	service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "bad-key"}).Return(&types.VerifyApiKeyResult{Valid: false}, nil).Once()

	plugin := &ApiKeyPlugin{
		config:    types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
		pluginCtx: &models.PluginContext{ServiceRegistry: &internaltests.MockServiceRegistry{}, Logger: &internaltests.MockLogger{}},
		Api:       NewAPI(service),
	}

	reqCtx := newRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "bad-key"})
	err := plugin.validateApiKeyHook()(reqCtx)
	require.NoError(t, err)
	assert.True(t, reqCtx.Handled)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
	service.AssertExpectations(t)
}

func TestApiKeyPluginHook_ValidKeyWithoutRateLimiting_PassesThrough(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", ReferenceID: "user-1"}
	service := &apiKeyTests.MockApiKeyService{}
	service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "good-key"}).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil).Once()

	plugin := &ApiKeyPlugin{
		config:    types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
		pluginCtx: &models.PluginContext{ServiceRegistry: &internaltests.MockServiceRegistry{}, Logger: &internaltests.MockLogger{}},
		Api:       NewAPI(service),
	}

	reqCtx := newRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "good-key"})
	err := plugin.validateApiKeyHook()(reqCtx)
	require.NoError(t, err)
	assert.False(t, reqCtx.Handled)
	assert.Nil(t, reqCtx.ResponseBody)
	assert.Equal(t, "user-1", *reqCtx.UserID)
	service.AssertExpectations(t)
}

func TestApiKeyPluginHook_ValidKeyWithRateLimiting_CallsSharedService(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", ReferenceID: "user-1", RateLimitEnabled: true, RateLimitTimeWindow: new(0), RateLimitMaxRequests: new(0)}
	*apiKey.RateLimitTimeWindow = 60
	*apiKey.RateLimitMaxRequests = 5
	service := &apiKeyTests.MockApiKeyService{}
	service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "good-key"}).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil).Once()

	rateLimitService := &mockRateLimitService{}
	rateLimitService.On("CheckAndIncrement", mock.Anything, "api-key-1", time.Minute, 5).Return(true, 1, time.Now().UTC().Add(time.Minute), nil).Once()

	registry := &internaltests.MockServiceRegistry{}
	registry.On("Get", models.ServiceRateLimit.String()).Return(any(rateLimitService)).Once()

	plugin := &ApiKeyPlugin{
		config:    types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
		pluginCtx: &models.PluginContext{ServiceRegistry: registry, Logger: &internaltests.MockLogger{}},
		Api:       NewAPI(service),
	}

	reqCtx := newRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "good-key"})
	err := plugin.validateApiKeyHook()(reqCtx)
	require.NoError(t, err)
	assert.False(t, reqCtx.Handled)
	assert.Equal(t, "4", reqCtx.ResponseWriter.Header().Get("X-RateLimit-Remaining"))
	service.AssertExpectations(t)
	rateLimitService.AssertExpectations(t)
	registry.AssertExpectations(t)
}

func TestApiKeyPluginHook_RateLimitExceeded_ReturnsTooManyRequests(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", ReferenceID: "user-1", RateLimitEnabled: true, RateLimitTimeWindow: new(0), RateLimitMaxRequests: new(0)}
	*apiKey.RateLimitTimeWindow = 60
	*apiKey.RateLimitMaxRequests = 5
	service := &apiKeyTests.MockApiKeyService{}
	service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "good-key"}).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil).Once()

	rateLimitService := &mockRateLimitService{}
	rateLimitService.On("CheckAndIncrement", mock.Anything, "api-key-1", time.Minute, 5).Return(false, 6, time.Now().UTC().Add(time.Minute), nil).Once()

	registry := &internaltests.MockServiceRegistry{}
	registry.On("Get", models.ServiceRateLimit.String()).Return(any(rateLimitService)).Once()

	plugin := &ApiKeyPlugin{
		config:    types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
		pluginCtx: &models.PluginContext{ServiceRegistry: registry, Logger: &internaltests.MockLogger{}},
		Api:       NewAPI(service),
	}

	reqCtx := newRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "good-key"})
	err := plugin.validateApiKeyHook()(reqCtx)
	require.NoError(t, err)
	assert.True(t, reqCtx.Handled)
	assert.Equal(t, http.StatusTooManyRequests, reqCtx.ResponseStatus)
	assert.Equal(t, "0", reqCtx.ResponseWriter.Header().Get("X-RateLimit-Remaining"))
	service.AssertExpectations(t)
	rateLimitService.AssertExpectations(t)
	registry.AssertExpectations(t)
}

func TestApiKeyPluginHook_RateLimitServiceUnavailable_FailsOpen(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", ReferenceID: "user-1", RateLimitEnabled: true, RateLimitTimeWindow: new(0), RateLimitMaxRequests: new(0)}
	*apiKey.RateLimitTimeWindow = 60
	*apiKey.RateLimitMaxRequests = 5
	service := &apiKeyTests.MockApiKeyService{}
	service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "good-key"}).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil).Once()

	registry := &internaltests.MockServiceRegistry{}
	registry.On("Get", models.ServiceRateLimit.String()).Return(nil).Once()

	plugin := &ApiKeyPlugin{
		config:    types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
		pluginCtx: &models.PluginContext{ServiceRegistry: registry},
		Api:       NewAPI(service),
	}

	reqCtx := newRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "good-key"})
	err := plugin.validateApiKeyHook()(reqCtx)
	require.NoError(t, err)
	assert.False(t, reqCtx.Handled)
	service.AssertExpectations(t)
	registry.AssertExpectations(t)
}

func newRequestContext(t *testing.T, method, path string, headers map[string]string) *models.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return &models.RequestContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
		Path:           path,
		Method:         method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         map[string]any{},
	}
}
