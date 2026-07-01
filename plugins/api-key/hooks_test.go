package apikey

import (
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
	rootservices "github.com/Authula/authula/services"
)

func TestApiKeyPluginHooks_MatchConfiguredHeaderOnly(t *testing.T) {
	t.Parallel()

	plugin := &ApiKeyPlugin{config: types.ApiKeyPluginConfig{Header: "X-Test-API-Key"}}
	plugin.config.ApplyDefaults()
	plugin.pluginCtx = &models.PluginContext{}

	hooks := plugin.buildHooks()
	require.Len(t, hooks, 1)

	tests := []struct {
		name    string
		headers map[string]string
		match   bool
	}{
		{
			name:    "missing configured header does not match",
			headers: map[string]string{},
			match:   false,
		},
		{
			name:    "configured header matches when present",
			headers: map[string]string{"X-Test-API-Key": "abc"},
			match:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
			for key, value := range tc.headers {
				reqCtx.Request.Header.Set(key, value)
			}

			assert.Equal(t, tc.match, hooks[0].Matcher(reqCtx))
		})
	}
}

func TestApiKeyPluginHook_ValidateApiKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		apiKey            *types.ApiKey
		verifyResult      *types.VerifyApiKeyResult
		rateLimitRule     *rootservices.RateLimitKeyRule
		rateLimitSetup    func(*internaltests.MockRateLimitService)
		userSetup         func(*internaltests.MockUserService)
		orgSetup          func(*internaltests.MockOrganizationService)
		expectedHandled   bool
		expectedStatus    int
		expectedActor     *models.Actor
		expectedRemaining string
	}{
		{
			name:            "invalid key returns unauthorized",
			verifyResult:    &types.VerifyApiKeyResult{Valid: false},
			expectedHandled: true,
			expectedStatus:  http.StatusUnauthorized,
		},
		{
			name:         "valid personal key sets user actor",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1", KeyHash: "api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser, Permissions: []string{"read"}}},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
			},
			expectedActor: &models.Actor{
				Type:   models.ActorUser,
				ID:     "user-1",
				Scopes: []string{"read"},
			},
		},
		{
			name:         "valid org key sets machine actor",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1", KeyHash: "api-key-1", OwnerID: "org-1", OwnerType: types.OwnerTypeOrganization, Permissions: []string{"org:api-key:read"}}},
			orgSetup: func(os *internaltests.MockOrganizationService) {
				os.On("ExistsByID", mock.Anything, "org-1").Return(true, nil).Once()
			},
			expectedActor: &models.Actor{
				Type:   models.ActorMachine,
				ID:     "api-key-1",
				Scopes: []string{"org:api-key:read"},
				Claims: map[string]any{"organization_id": "org-1"},
			},
		},
		{
			name:         "personal key user not found returns unauthorized",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1", KeyHash: "api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser}},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), nil).Once()
			},
			expectedHandled: true,
			expectedStatus:  http.StatusUnauthorized,
		},
		{
			name:         "org key org not found returns unauthorized",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1", KeyHash: "api-key-1", OwnerID: "org-1", OwnerType: types.OwnerTypeOrganization}},
			orgSetup: func(os *internaltests.MockOrganizationService) {
				os.On("ExistsByID", mock.Anything, "org-1").Return(false, nil).Once()
			},
			expectedHandled: true,
			expectedStatus:  http.StatusUnauthorized,
		},
		{
			name:         "valid key without rate limiting passes through",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1", KeyHash: "api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser}},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
			},
			expectedActor: &models.Actor{
				Type: models.ActorUser,
				ID:   "user-1",
			},
		},
		{
			name: "valid key with rate limiting calls shared service",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: func() *types.ApiKey {
				apiKey := &types.ApiKey{ID: "api-key-1", KeyHash: "hashed-api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser, RateLimitEnabled: true}
				return apiKey
			}()},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
			},
			rateLimitRule: &rootservices.RateLimitKeyRule{Window: time.Minute, MaxRequests: 5},
			rateLimitSetup: func(rateLimitService *internaltests.MockRateLimitService) {
				rateLimitService.On("CheckAndIncrement", mock.Anything, "hashed-api-key-1", time.Minute, 5).Return(true, 1, time.Now().UTC().Add(time.Minute), nil).Once()
			},
			expectedActor: &models.Actor{
				Type: models.ActorUser,
				ID:   "user-1",
			},
			expectedRemaining: "4",
		},
		{
			name: "rate limit exceeded returns too many requests",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: func() *types.ApiKey {
				apiKey := &types.ApiKey{ID: "api-key-1", KeyHash: "hashed-api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser, RateLimitEnabled: true}
				return apiKey
			}()},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
			},
			rateLimitRule: &rootservices.RateLimitKeyRule{Window: time.Minute, MaxRequests: 5},
			rateLimitSetup: func(rateLimitService *internaltests.MockRateLimitService) {
				rateLimitService.On("CheckAndIncrement", mock.Anything, "hashed-api-key-1", time.Minute, 5).Return(false, 6, time.Now().UTC().Add(time.Minute), nil).Once()
			},
			expectedHandled:   true,
			expectedStatus:    http.StatusTooManyRequests,
			expectedRemaining: "0",
			expectedActor: &models.Actor{
				Type: models.ActorUser,
				ID:   "user-1",
			},
		},
		{
			name: "rate limit service unavailable fails open",
			verifyResult: &types.VerifyApiKeyResult{Valid: true, ApiKey: func() *types.ApiKey {
				apiKey := &types.ApiKey{ID: "api-key-1", KeyHash: "hashed-api-key-1", OwnerID: "user-1", OwnerType: types.OwnerTypeUser, RateLimitEnabled: true}
				return apiKey
			}()},
			userSetup: func(us *internaltests.MockUserService) {
				us.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
			},
			expectedActor: &models.Actor{
				Type: models.ActorUser,
				ID:   "user-1",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &apiKeyTests.MockApiKeyService{}
			service.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "good-key"}).Return(tc.verifyResult, nil).Once()
			if tc.verifyResult != nil && tc.verifyResult.Valid && tc.verifyResult.ApiKey != nil {
				service.On("RecordLastRequest", mock.Anything, tc.verifyResult.ApiKey.ID, mock.Anything).Return(tc.verifyResult.ApiKey, nil).Maybe()
			}

			mockLogger := &internaltests.MockLogger{}
			var rateLimitService *internaltests.MockRateLimitService
			var rateLimiterService rootservices.RateLimiterService
			if tc.rateLimitRule != nil || tc.rateLimitSetup != nil {
				rateLimitService = &internaltests.MockRateLimitService{}
				if tc.rateLimitSetup != nil {
					tc.rateLimitSetup(rateLimitService)
				}
				if tc.rateLimitRule != nil {
					rateLimitService.On("GetRule", mock.Anything, mock.Anything).Return(tc.rateLimitRule, nil).Once()
				} else {
					rateLimitService.On("GetRule", mock.Anything, mock.Anything).Return((*rootservices.RateLimitKeyRule)(nil), nil).Once()
				}
				rateLimiterService = rateLimitService
			}

			mockUserService := &internaltests.MockUserService{}
			if tc.userSetup != nil {
				tc.userSetup(mockUserService)
			}

			mockOrgService := &internaltests.MockOrganizationService{}
			if tc.orgSetup != nil {
				tc.orgSetup(mockOrgService)
			}

			plugin := &ApiKeyPlugin{
				config:              types.ApiKeyPluginConfig{Header: "X-Test-API-Key"},
				logger:              mockLogger,
				pluginCtx:           &models.PluginContext{Logger: mockLogger},
				rateLimiterService:  rateLimiterService,
				userService:         mockUserService,
				organizationService: mockOrgService,
				Api:                 NewAPI(service),
			}

			reqCtx := internaltests.NewRequestContext(t, http.MethodGet, "/protected", map[string]string{"X-Test-API-Key": "good-key"})
			err := plugin.validateApiKeyHook()(reqCtx)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedHandled, reqCtx.Handled)
			if tc.expectedStatus != 0 {
				assert.Equal(t, tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedActor != nil {
				require.NotNil(t, reqCtx.Actor)
				assert.Equal(t, tc.expectedActor.Type, reqCtx.Actor.Type)
				assert.Equal(t, tc.expectedActor.ID, reqCtx.Actor.ID)
				if tc.expectedActor.Scopes != nil {
					assert.Equal(t, tc.expectedActor.Scopes, reqCtx.Actor.Scopes)
				}
				if tc.expectedActor.Claims != nil {
					assert.Equal(t, tc.expectedActor.Claims, reqCtx.Actor.Claims)
				}
			}
			if tc.expectedRemaining != "" {
				assert.Equal(t, tc.expectedRemaining, reqCtx.ResponseWriter.Header().Get("X-RateLimit-Remaining"))
			}

			service.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
			mockOrgService.AssertExpectations(t)
			if rateLimitService != nil {
				rateLimitService.AssertExpectations(t)
			}
		})
	}
}
