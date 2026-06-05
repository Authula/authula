package totp

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/handlers"
)

func Routes(p *TOTPPlugin) []models.Route {
	uc := p.Api.useCases

	enableHandler := &handlers.EnableHandler{
		GlobalConfig: p.globalConfig,
		PluginConfig: p.pluginConfig,
		UseCase:      uc.Enable,
	}
	disableHandler := &handlers.DisableHandler{
		UseCase: uc.Disable,
	}
	getTOTPURIHandler := &handlers.GetTOTPURIHandler{
		GlobalConfig: p.globalConfig,
		UseCase:      uc.GetTOTPURI,
	}
	verifyTOTPHandler := &handlers.VerifyTOTPHandler{
		PluginConfig: p.pluginConfig,
		UseCase:      uc.VerifyTOTP,
	}
	generateBackupCodesHandler := &handlers.GenerateBackupCodesHandler{
		UseCase: uc.GenerateBackupCodes,
	}
	verifyBackupCodeHandler := &handlers.VerifyBackupCodeHandler{
		PluginConfig: p.pluginConfig,
		UseCase:      uc.VerifyBackupCode,
	}

	return []models.Route{
		{
			Method: http.MethodPost,
			Path:   "/totp/enable",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: enableHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/totp/disable",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: disableHandler.Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/totp/get-uri",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: getTOTPURIHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/totp/verify",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: verifyTOTPHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/totp/verify-backup-code",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: verifyBackupCodeHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/totp/generate-backup-codes",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: generateBackupCodesHandler.Handler(),
		},
	}
}
