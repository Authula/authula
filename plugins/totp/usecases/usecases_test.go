package usecases

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type mockTOTPRepo struct {
	mock.Mock
}

func (m *mockTOTPRepo) GetByUserID(ctx context.Context, userID string) (*types.TOTPRecord, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TOTPRecord), args.Error(1)
}

func (m *mockTOTPRepo) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockTOTPRepo) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	args := m.Called(ctx, userID, enabled)
	return args.Error(0)
}

func (m *mockTOTPRepo) UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error {
	args := m.Called(ctx, userID, backupCodes)
	return args.Error(0)
}

func (m *mockTOTPRepo) CompareAndSwapBackupCodes(ctx context.Context, userID, expectedBackupCodes, newBackupCodes string) (bool, error) {
	args := m.Called(ctx, userID, expectedBackupCodes, newBackupCodes)
	return args.Bool(0), args.Error(1)
}

func (m *mockTOTPRepo) Create(ctx context.Context, userID, secret, backupCodes string) (*types.TOTPRecord, error) {
	args := m.Called(ctx, userID, secret, backupCodes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TOTPRecord), args.Error(1)
}

func (m *mockTOTPRepo) CreateTrustedDevice(ctx context.Context, userID, token, userAgent string, expiresAt time.Time) (*types.TrustedDevice, error) {
	args := m.Called(ctx, userID, token, userAgent, expiresAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TrustedDevice), args.Error(1)
}

func (m *mockTOTPRepo) DeleteTrustedDevicesByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestDisableUseCase_UsesRepository(t *testing.T) {
	repo := &mockTOTPRepo{}
	eventBus := &tests.MockEventBus{}

	repo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
	repo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
	repo.On("DeleteTrustedDevicesByUserID", mock.Anything, "user-1").Return(nil).Once()
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

	uc := NewDisableUseCase(&tests.MockLogger{}, eventBus, repo)
	err := uc.Disable(context.Background(), "user-1")
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestGenerateBackupCodesUseCase_UpdatesRepository(t *testing.T) {
	passwordSvc := &tests.MockPasswordService{}
	repo := &mockTOTPRepo{}

	repo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: "[]"}, nil).Once()
	passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
	repo.On("UpdateBackupCodes", mock.Anything, "user-1", mock.AnythingOfType("string")).Return(nil).Once()

	backupSvc := services.NewBackupCodeService(2, passwordSvc)
	uc := NewGenerateBackupCodesUseCase(backupSvc, repo)
	codes, err := uc.Generate(context.Background(), "user-1")
	require.NoError(t, err)
	require.Len(t, codes, 2)

	var stored []string
	updateArgs := repo.Calls[len(repo.Calls)-1].Arguments
	require.NoError(t, json.Unmarshal([]byte(updateArgs.Get(2).(string)), &stored))
	require.Len(t, stored, 2)

	passwordSvc.AssertExpectations(t)
	repo.AssertExpectations(t)
}
