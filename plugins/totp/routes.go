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
	viewBackupCodesHandler := &handlers.ViewBackupCodesHandler{
		UseCase: uc.ViewBackupCodes,
	}

	return []models.Route{
		{Path: "/totp/enable", Method: http.MethodPost, Handler: enableHandler.Handler()},
		{Path: "/totp/disable", Method: http.MethodPost, Handler: disableHandler.Handler()},
		{Path: "/totp/get-uri", Method: http.MethodPost, Handler: getTOTPURIHandler.Handler()},
		{Path: "/totp/verify", Method: http.MethodPost, Handler: verifyTOTPHandler.Handler()},
		{Path: "/totp/generate-backup-codes", Method: http.MethodPost, Handler: generateBackupCodesHandler.Handler()},
		{Path: "/totp/verify-backup-code", Method: http.MethodPost, Handler: verifyBackupCodeHandler.Handler()},
		{Path: "/totp/view-backup-codes", Method: http.MethodPost, Handler: viewBackupCodesHandler.Handler()},
	}
}
