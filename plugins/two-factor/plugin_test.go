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

// mockVerificationServiceForHandler implements the handlers.VerificationService
// interface (FindByToken + Delete), which differs from the root VerificationService.
type mockVerificationServiceForHandler struct {
	FindByTokenFn func(ctx context.Context, hashedToken string, vType models.VerificationType) (*models.Verification, error)
	DeleteFn      func(ctx context.Context, id string) error
}

func (m *mockVerificationServiceForHandler) FindByToken(ctx context.Context, hashedToken string, vType models.VerificationType) (*models.Verification, error) {
	if m.FindByTokenFn != nil {
		return m.FindByTokenFn(ctx, hashedToken, vType)
	}
	return nil, nil
}

func (m *mockVerificationServiceForHandler) Delete(ctx context.Context, id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

// mockTokenServiceForHandler implements handlers.TokenService (Hash only).
type mockTokenServiceForHandler struct {
	HashFn func(token string) string
}

func (m *mockTokenServiceForHandler) Hash(token string) string {
	if m.HashFn != nil {
		return m.HashFn(token)
	}
	return "hashed-" + token
}

// mockEnableUseCase implements usecases.EnableUseCase for handler-level tests.
type mockEnableUseCase struct {
	EnableFn func(ctx context.Context, userID, password, issuer, email string) (*types.EnableResult, error)
}

func (m *mockEnableUseCase) Enable(ctx context.Context, userID, password, issuer, email string) (*types.EnableResult, error) {
	return m.EnableFn(ctx, userID, password, issuer, email)
}

// mockVerifyTOTPUseCase implements usecases.VerifyTOTPUseCase for handler-level tests.
type mockVerifyTOTPUseCase struct {
	VerifyFn func(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error)
}

func (m *mockVerifyTOTPUseCase) Verify(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	return m.VerifyFn(ctx, userID, code, trustDevice, ipAddress, userAgent)
}

// mockUserServiceForHandler implements handlers.UserService (GetByID only).
type mockUserServiceForHandler struct {
	GetByIDFn func(ctx context.Context, id string) (*models.User, error)
}

func (m *mockUserServiceForHandler) GetByID(ctx context.Context, id string) (*models.User, error) {
	return m.GetByIDFn(ctx, id)
}

// ---------------------------------------------------------------------------
// Helper: build a fully-initialized plugin with mock services
// ---------------------------------------------------------------------------

func buildTestPlugin(t *testing.T) (*TwoFactorPlugin, *tests.MockUserService, *tests.MockTokenService, *tests.MockVerificationService) {
	t.Helper()

	userSvc := tests.NewMockUserService(t)
	accountSvc := tests.NewMockAccountService(t)
	sessionSvc := tests.NewMockSessionService(t)
	verifSvc := tests.NewMockVerificationService(t)
	tokenSvc := tests.NewMockTokenService(t)
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

func TestAfterHookPassesThroughWithout2FA(t *testing.T) {
	plugin, userSvc, _, _ := buildTestPlugin(t)

	userSvc.GetByIDFn = func(_ context.Context, id string) (*models.User, error) {
		return &models.User{
			ID:               id,
			Email:            "test@example.com",
			TwoFactorEnabled: nil, // 2FA not enabled
		}, nil
	}

	hooks := plugin.Hooks()
	hook := hooks[0]

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/sign-in/email", nil)

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         make(map[string]any),
	}
	reqCtx.Values[models.ContextAuthSuccess.String()] = true
	reqCtx.Values[models.ContextUserID.String()] = "user-1"

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	reqCtx.Request = req.WithContext(ctx)

	// Matcher should match (auth success is true)
	if !hook.Matcher(reqCtx) {
		t.Fatal("expected matcher to return true for auth success")
	}

	err := hook.Handler(reqCtx)
	if err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	// Response should NOT have been overridden with twoFactorRedirect
	if reqCtx.ResponseReady {
		t.Error("expected ResponseReady to be false for non-2FA user")
	}

	// auth.success should still be present
	if _, ok := reqCtx.Values[models.ContextAuthSuccess.String()]; !ok {
		t.Error("expected ContextAuthSuccess to still be present")
	}
}

func TestAfterHookIntercepts2FAUser(t *testing.T) {
	plugin, userSvc, tokenSvc, verifSvc := buildTestPlugin(t)

	enabled := true
	userSvc.GetByIDFn = func(_ context.Context, id string) (*models.User, error) {
		return &models.User{
			ID:               id,
			Email:            "test@example.com",
			TwoFactorEnabled: &enabled,
		}, nil
	}

	tokenSvc.GenerateFn = func() (string, error) {
		return "pending-token-abc", nil
	}
	tokenSvc.HashFn = func(token string) string {
		return "hashed-" + token
	}

	verifSvc.CreateFn = func(_ context.Context, userID, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
		return &models.Verification{
			ID:     "verif-1",
			UserID: &userID,
			Token:  hashedToken,
			Type:   vType,
		}, nil
	}

	hooks := plugin.Hooks()
	hook := hooks[0]

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/sign-in/email", nil)

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         make(map[string]any),
	}
	reqCtx.Values[models.ContextAuthSuccess.String()] = true
	reqCtx.Values[models.ContextUserID.String()] = "user-1"

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	reqCtx.Request = req.WithContext(ctx)

	err := hook.Handler(reqCtx)
	if err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	// Should have overridden the response
	if !reqCtx.ResponseReady {
		t.Fatal("expected ResponseReady to be true")
	}

	var body map[string]any
	if err := json.Unmarshal(reqCtx.ResponseBody, &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["twoFactorRedirect"] != true {
		t.Errorf("expected twoFactorRedirect=true, got %v", body["twoFactorRedirect"])
	}

	// ContextAuthSuccess should have been deleted
	if _, ok := reqCtx.Values[models.ContextAuthSuccess.String()]; ok {
		t.Error("expected ContextAuthSuccess to be deleted from values")
	}

	// Check that a "two_factor_pending" cookie was set on the response writer
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "two_factor_pending" {
			found = true
			if c.Value != "pending-token-abc" {
				t.Errorf("expected cookie value 'pending-token-abc', got '%s'", c.Value)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'two_factor_pending' cookie to be set")
	}
}

func TestVerifyTOTPHandlerSuccess(t *testing.T) {
	userID := "user-1"

	tokenSvc := &mockTokenServiceForHandler{
		HashFn: func(token string) string {
			return "hashed-" + token
		},
	}

	verifSvc := &mockVerificationServiceForHandler{
		FindByTokenFn: func(_ context.Context, hashedToken string, vType models.VerificationType) (*models.Verification, error) {
			return &models.Verification{
				ID:        "verif-1",
				UserID:    &userID,
				Token:     hashedToken,
				Type:      vType,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}, nil
		},
		DeleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}

	enabledTrue := true
	mockUC := &mockVerifyTOTPUseCase{
		VerifyFn: func(_ context.Context, uid, code string, trustDevice bool, ipAddr, ua *string) (*types.VerifyResult, error) {
			return &types.VerifyResult{
				User: &models.User{
					ID:               uid,
					Email:            "test@example.com",
					TwoFactorEnabled: &enabledTrue,
				},
				Session: &models.Session{
					ID:        "session-1",
					UserID:    uid,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				},
				SessionToken: "session-token-xyz",
			}, nil
		},
	}

	handler := &handlers.VerifyTOTPHandler{
		UseCase:               mockUC,
		VerificationService:   verifSvc,
		TokenService:          tokenSvc,
		TrustedDeviceDuration: 30 * 24 * time.Hour,
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

	userSvc := &mockUserServiceForHandler{
		GetByIDFn: func(_ context.Context, id string) (*models.User, error) {
			return &models.User{ID: id, Email: "test@example.com"}, nil
		},
	}

	mockUC := &mockEnableUseCase{
		EnableFn: func(_ context.Context, uid, password, issuer, email string) (*types.EnableResult, error) {
			return &types.EnableResult{
				TotpURI:     "otpauth://totp/MyApp:test@example.com?secret=ABCDEF&issuer=MyApp",
				BackupCodes: []string{"code1", "code2", "code3"},
			}, nil
		},
	}

	handler := &handlers.EnableHandler{
		UseCase:     mockUC,
		UserService: userSvc,
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
