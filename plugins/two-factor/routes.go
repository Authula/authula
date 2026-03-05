package twofactor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/handlers"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

// verificationServiceAdapter adapts rootservices.VerificationService to the
// handlers.VerificationService interface. The handler interface uses
// FindByToken(ctx, hashedToken, verificationType) while the root service
// uses GetByToken(ctx, hashedToken) and stores the type on the record.
type verificationServiceAdapter struct {
	svc rootservices.VerificationService
}

func (a *verificationServiceAdapter) FindByToken(ctx context.Context, hashedToken string, verificationType models.VerificationType) (*models.Verification, error) {
	v, err := a.svc.GetByToken(ctx, hashedToken)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	if v.Type != verificationType {
		return nil, fmt.Errorf("verification type mismatch")
	}
	return v, nil
}

func (a *verificationServiceAdapter) Delete(ctx context.Context, id string) error {
	return a.svc.Delete(ctx, id)
}

func Routes(p *TwoFactorPlugin) []models.Route {
	uc := p.Api.useCases

	verificationAdapter := &verificationServiceAdapter{svc: p.verificationService}

	enableHandler := &handlers.EnableHandler{
		UseCase:     uc.Enable,
		UserService: p.userService,
	}
	disableHandler := &handlers.DisableHandler{
		UseCase: uc.Disable,
	}
	getTOTPURIHandler := &handlers.GetTOTPURIHandler{
		UseCase:     uc.GetTOTPURI,
		UserService: p.userService,
	}
	verifyTOTPHandler := &handlers.VerifyTOTPHandler{
		UseCase:               uc.VerifyTOTP,
		VerificationService:   verificationAdapter,
		TokenService:          p.tokenService,
		TrustedDeviceDuration: p.pluginConfig.TrustedDeviceDuration,
	}
	generateBackupCodesHandler := &handlers.GenerateBackupCodesHandler{
		UseCase: uc.GenerateBackupCodes,
	}
	verifyBackupCodeHandler := &handlers.VerifyBackupCodeHandler{
		UseCase:               uc.VerifyBackupCode,
		VerificationService:   verificationAdapter,
		TokenService:          p.tokenService,
		TrustedDeviceDuration: p.pluginConfig.TrustedDeviceDuration,
	}
	viewBackupCodesHandler := &handlers.ViewBackupCodesHandler{
		UseCase: uc.ViewBackupCodes,
	}

	return []models.Route{
		{Path: "/two-factor/enable", Method: http.MethodPost, Handler: enableHandler.Handler(), Metadata: map[string]any{"plugins": []string{"session.auth"}}},
		{Path: "/two-factor/disable", Method: http.MethodPost, Handler: disableHandler.Handler(), Metadata: map[string]any{"plugins": []string{"session.auth"}}},
		{Path: "/two-factor/get-totp-uri", Method: http.MethodPost, Handler: getTOTPURIHandler.Handler(), Metadata: map[string]any{"plugins": []string{"session.auth"}}},
		{Path: "/two-factor/verify-totp", Method: http.MethodPost, Handler: verifyTOTPHandler.Handler()},
		{Path: "/two-factor/generate-backup-codes", Method: http.MethodPost, Handler: generateBackupCodesHandler.Handler(), Metadata: map[string]any{"plugins": []string{"session.auth"}}},
		{Path: "/two-factor/verify-backup-code", Method: http.MethodPost, Handler: verifyBackupCodeHandler.Handler()},
		{Path: "/two-factor/view-backup-codes", Method: http.MethodPost, Handler: viewBackupCodesHandler.Handler(), Metadata: map[string]any{"plugins": []string{"session.auth"}}},
	}
}
