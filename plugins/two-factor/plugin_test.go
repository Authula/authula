package twofactor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/usecases"
)

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

type mockServiceRegistry struct {
	services map[string]any
}

func newMockServiceRegistry() *mockServiceRegistry {
	return &mockServiceRegistry{services: make(map[string]any)}
}

func (m *mockServiceRegistry) Register(name string, service any) {
	m.services[name] = service
}

func (m *mockServiceRegistry) Get(name string) any {
	return m.services[name]
}

type mockPasswordService struct {
	VerifyFn func(password, encoded string) bool
	HashFn   func(password string) (string, error)
}

func (m *mockPasswordService) Verify(password, encoded string) bool {
	return m.VerifyFn(password, encoded)
}
func (m *mockPasswordService) Hash(password string) (string, error) { return m.HashFn(password) }

type mockEventBus struct{}

func (m *mockEventBus) Publish(_ context.Context, _ models.Event) error { return nil }
func (m *mockEventBus) Close() error                                    { return nil }
func (m *mockEventBus) Subscribe(_ string, _ models.EventHandler) (models.SubscriptionID, error) {
	return 0, nil
}
func (m *mockEventBus) Unsubscribe(_ string, _ models.SubscriptionID) {}

// mockEnableUseCase implements usecases.EnableUseCase for handler-level tests.
type mockEnableUseCase struct {
	EnableFn func(ctx context.Context, userID, password, issuer string) (*types.EnableResult, error)
}

func (m *mockEnableUseCase) Enable(ctx context.Context, userID, password, issuer string) (*types.EnableResult, error) {
	return m.EnableFn(ctx, userID, password, issuer)
}

// mockVerifyTOTPUseCase implements usecases.VerifyTOTPUseCase for handler-level tests.
type mockVerifyTOTPUseCase struct {
	VerifyFn func(ctx context.Context, pendingToken, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error)
}

func (m *mockVerifyTOTPUseCase) Verify(ctx context.Context, pendingToken, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	return m.VerifyFn(ctx, pendingToken, code, trustDevice, ipAddress, userAgent)
}

// ---------------------------------------------------------------------------
// Helper: build a fully-initialized plugin with mock services
// ---------------------------------------------------------------------------

func buildTestPlugin(t *testing.T) (*TwoFactorPlugin, *tests.MockUserService, *tests.MockTokenService, *tests.MockVerificationService) {
	t.Helper()

	userSvc := &tests.MockUserService{}
	accountSvc := &tests.MockAccountService{}
	sessionSvc := &tests.MockSessionService{}
	verifSvc := &tests.MockVerificationService{}
	tokenSvc := &tests.MockTokenService{}
	passwordSvc := &mockPasswordService{
		VerifyFn: func(_, _ string) bool { return true },
		HashFn:   func(p string) (string, error) { return "hashed-" + p, nil },
	}

	reg := newMockServiceRegistry()
	reg.Register(models.ServiceUser.String(), userSvc)
	reg.Register(models.ServiceAccount.String(), accountSvc)
	reg.Register(models.ServiceSession.String(), sessionSvc)
	reg.Register(models.ServiceVerification.String(), verifSvc)
	reg.Register(models.ServiceToken.String(), tokenSvc)
	reg.Register(models.ServicePassword.String(), passwordSvc)

	plugin := New(types.TwoFactorPluginConfig{})

	pluginCtx := &models.PluginContext{
		DB:              nil, // not used in hook/handler tests
		Logger:          &tests.MockLogger{},
		EventBus:        &mockEventBus{},
		ServiceRegistry: reg,
		GetConfig: func() *models.Config {
			return &models.Config{
				Session: models.SessionConfig{
					ExpiresIn: 24 * time.Hour,
				},
			}
		},
	}

	if err := plugin.Init(pluginCtx); err != nil {
		t.Fatalf("plugin.Init failed: %v", err)
	}

	return plugin, userSvc, tokenSvc, verifSvc
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPluginInit(t *testing.T) {
	plugin, _, _, _ := buildTestPlugin(t)

	routes := plugin.Routes()
	if len(routes) != 7 {
		t.Fatalf("expected 7 routes, got %d", len(routes))
	}

	hooks := plugin.Hooks()
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}

	if hooks[0].Stage != models.HookAfter {
		t.Errorf("expected HookAfter stage, got %v", hooks[0].Stage)
	}
}

// Note: TestAfterHookPassesThroughWithout2FA and TestAfterHookIntercepts2FAUser
// were removed because the hook now uses repo.IsEnabled() which requires a real
// database connection. These scenarios should be covered by integration tests.

func TestVerifyTOTPHandlerSuccess(t *testing.T) {
	mockUC := &mockVerifyTOTPUseCase{
		VerifyFn: func(_ context.Context, pendingToken, code string, trustDevice bool, ipAddr, ua *string) (*types.VerifyResult, error) {
			return &types.VerifyResult{
				User: &models.User{
					ID:    "user-1",
					Email: "test@example.com",
				},
				Session: &models.Session{
					ID:        "session-1",
					UserID:    "user-1",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				},
				SessionToken: "session-token-xyz",
			}, nil
		},
	}

	handler := &handlers.VerifyTOTPHandler{
		UseCase:      mockUC,
		PluginConfig: &types.TwoFactorPluginConfig{},
	}

	body, _ := json.Marshal(types.VerifyTOTPRequest{
		Code: "123456",
	})

	req := httptest.NewRequest("POST", "/two-factor/verify-totp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "two_factor_pending",
		Value: "pending-token-abc",
	})

	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req

	handler.Handler().ServeHTTP(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Errorf("expected status 200, got %d", reqCtx.ResponseStatus)
	}

	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	if !ok || !authSuccess {
		t.Error("expected ContextAuthSuccess=true in values")
	}

	sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string)
	if !ok || sessionID == "" {
		t.Error("expected ContextSessionID to be set")
	}
	if sessionID != "session-1" {
		t.Errorf("expected session ID 'session-1', got '%s'", sessionID)
	}
}

func TestEnableHandlerSuccess(t *testing.T) {
	userID := "user-1"

	mockUC := &mockEnableUseCase{
		EnableFn: func(_ context.Context, uid, password, issuer string) (*types.EnableResult, error) {
			return &types.EnableResult{
				TotpURI:     "otpauth://totp/MyApp:test@example.com?secret=ABCDEF&issuer=MyApp",
				BackupCodes: []string{"code1", "code2", "code3"},
			}, nil
		},
	}

	handler := &handlers.EnableHandler{
		UseCase: mockUC,
	}

	body, _ := json.Marshal(types.EnableRequest{
		Password: "my-password",
	})

	req := httptest.NewRequest("POST", "/two-factor/enable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         make(map[string]any),
		UserID:         &userID,
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req

	handler.Handler().ServeHTTP(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Errorf("expected status 200, got %d", reqCtx.ResponseStatus)
	}

	var resp types.EnableResponse
	if err := json.Unmarshal(reqCtx.ResponseBody, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.TotpURI == "" {
		t.Error("expected totpURI in response")
	}
	if len(resp.BackupCodes) != 3 {
		t.Errorf("expected 3 backup codes, got %d", len(resp.BackupCodes))
	}
}

// Ensure usecases package is used (prevents "imported and not used" error).
var _ usecases.EnableUseCase = (*mockEnableUseCase)(nil)
var _ usecases.VerifyTOTPUseCase = (*mockVerifyTOTPUseCase)(nil)
