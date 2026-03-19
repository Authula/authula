package totp

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/handlers"
)

func Routes(p *TOTPPlugin) []models.Route {
	uc := p.Api.useCases

	enableHandler := &handlers.EnableHandler{
		UseCase:      uc.Enable,
		PluginConfig: p.pluginConfig,
	}
	disableHandler := &handlers.DisableHandler{
		UseCase: uc.Disable,
	}
	getTOTPURIHandler := &handlers.GetTOTPURIHandler{
		UseCase: uc.GetTOTPURI,
	}
	verifyTOTPHandler := &handlers.VerifyTOTPHandler{
		UseCase:      uc.VerifyTOTP,
		PluginConfig: p.pluginConfig,
	}
	generateBackupCodesHandler := &handlers.GenerateBackupCodesHandler{
		UseCase: uc.GenerateBackupCodes,
	}
	verifyBackupCodeHandler := &handlers.VerifyBackupCodeHandler{
		UseCase:      uc.VerifyBackupCode,
		PluginConfig: p.pluginConfig,
	}

	return []models.Route{
		{Method: http.MethodPost, Path: "/totp/enable", Handler: enableHandler.Handler()},
		{Method: http.MethodPost, Path: "/totp/disable", Handler: disableHandler.Handler()},
		{Method: http.MethodGet, Path: "/totp/get-uri", Handler: getTOTPURIHandler.Handler()},
		{Method: http.MethodPost, Path: "/totp/verify", Handler: verifyTOTPHandler.Handler()},
		{Method: http.MethodPost, Path: "/totp/verify-backup-code", Handler: verifyBackupCodeHandler.Handler()},
		{Method: http.MethodPost, Path: "/totp/generate-backup-codes", Handler: generateBackupCodesHandler.Handler()},
	}
}
