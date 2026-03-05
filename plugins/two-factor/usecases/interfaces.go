package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
)

type UseCases struct {
	Enable              EnableUseCase
	Disable             DisableUseCase
	GetTOTPURI          GetTOTPURIUseCase
	VerifyTOTP          VerifyTOTPUseCase
	GenerateBackupCodes GenerateBackupCodesUseCase
	VerifyBackupCode    VerifyBackupCodeUseCase
	ViewBackupCodes     ViewBackupCodesUseCase
}

type EnableUseCase interface {
	Enable(ctx context.Context, userID, password, issuer, email string) (*types.EnableResult, error)
}

type DisableUseCase interface {
	Disable(ctx context.Context, userID, password string) error
}

type GetTOTPURIUseCase interface {
	GetTOTPURI(ctx context.Context, userID, password, email string) (string, error)
}

type VerifyTOTPUseCase interface {
	Verify(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error)
}

type GenerateBackupCodesUseCase interface {
	Generate(ctx context.Context, userID, password string) ([]string, error)
}

type VerifyBackupCodeUseCase interface {
	Verify(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error)
}

type ViewBackupCodesUseCase interface {
	View(ctx context.Context, userID, password string) ([]string, error)
}
