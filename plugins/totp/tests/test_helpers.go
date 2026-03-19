package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type MockTOTPRepo struct {
	mock.Mock
}

func (m *MockTOTPRepo) GetByUserID(ctx context.Context, userID string) (*types.TOTPRecord, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TOTPRecord), args.Error(1)
}

func (m *MockTOTPRepo) DeleteByUserID(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockTOTPRepo) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	return m.Called(ctx, userID, enabled).Error(0)
}

func (m *MockTOTPRepo) UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error {
	return m.Called(ctx, userID, backupCodes).Error(0)
}

func (m *MockTOTPRepo) CompareAndSwapBackupCodes(ctx context.Context, userID, expected, next string) (bool, error) {
	args := m.Called(ctx, userID, expected, next)
	return args.Bool(0), args.Error(1)
}

func (m *MockTOTPRepo) Create(ctx context.Context, userID, secret, backupCodes string) (*types.TOTPRecord, error) {
	args := m.Called(ctx, userID, secret, backupCodes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TOTPRecord), args.Error(1)
}

func (m *MockTOTPRepo) CreateTrustedDevice(ctx context.Context, userID, token, userAgent string, expiresAt time.Time) (*types.TrustedDevice, error) {
	args := m.Called(ctx, userID, token, userAgent, expiresAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TrustedDevice), args.Error(1)
}

func (m *MockTOTPRepo) DeleteTrustedDevicesByUserID(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

// CookieFromRecorder returns the named Set-Cookie value from the recorder, or nil.
func CookieFromRecorder(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// NewNoopEventBus returns a MockEventBus that silently accepts all Publish calls.
func NewNoopEventBus() *internaltests.MockEventBus {
	eb := &internaltests.MockEventBus{}
	eb.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
	return eb
}

func NewPluginConfig() *types.TOTPPluginConfig {
	return &types.TOTPPluginConfig{
		TrustedDeviceDuration: 30 * 24 * time.Hour,
		SecureCookie:          false,
		SameSite:              "lax",
	}
}
