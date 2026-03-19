package usecases

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type UseCases struct {
	Enable              *EnableUseCase
	Disable             *DisableUseCase
	GetTOTPURI          *GetTOTPURIUseCase
	VerifyTOTP          *VerifyTOTPUseCase
	GenerateBackupCodes *GenerateBackupCodesUseCase
	VerifyBackupCode    *VerifyBackupCodeUseCase
}

type TOTPReadRepository interface {
	GetByUserID(ctx context.Context, userID string) (*types.TOTPRecord, error)
}

type TOTPWriteRepository interface {
	DeleteByUserID(ctx context.Context, userID string) error
	SetEnabled(ctx context.Context, userID string, enabled bool) error
	UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error
	CompareAndSwapBackupCodes(ctx context.Context, userID, expectedBackupCodes, newBackupCodes string) (bool, error)
}

type TOTPCreateRepository interface {
	Create(ctx context.Context, userID, secret, backupCodes string) (*types.TOTPRecord, error)
	CreateTrustedDevice(ctx context.Context, userID, token, userAgent string, expiresAt time.Time) (*types.TrustedDevice, error)
}

type TOTPTrustedDeviceRepository interface {
	DeleteTrustedDevicesByUserID(ctx context.Context, userID string) error
}

type TOTPRepository interface {
	TOTPReadRepository
	TOTPWriteRepository
	TOTPCreateRepository
	TOTPTrustedDeviceRepository
}
