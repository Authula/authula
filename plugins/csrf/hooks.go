package csrf

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type CSRFHookID string

const (
	HookIDCSRFGenerate CSRFHookID = "csrf.generate"
	HookIDCSRFProtect  CSRFHookID = "csrf.protect"
)

func (id CSRFHookID) String() string {
	return string(id)
}

func (p *CSRFPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Matcher: p.safeMethodMatcher,
			Handler: p.generateCSRFTokenHook,
			Order:   5,
		},
		{
			Stage:    models.HookBefore,
			PluginID: HookIDCSRFProtect.String(),
			Matcher:  p.unsafeMethodMatcher,
			Handler:  p.validateCSRFTokenHook,
			Order:    5,
		},
		{
			Stage:   models.HookAfter,
			Matcher: p.signedOutMatcher,
			Handler: p.clearCSRFTokenHook,
			Order:   5,
		},
	}
}

func (p *CSRFPlugin) safeMethodMatcher(reqCtx *models.RequestContext) bool {
	method := reqCtx.Method
	isValidMethod := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	return isValidMethod
}

func (p *CSRFPlugin) unsafeMethodMatcher(reqCtx *models.RequestContext) bool {
	method := reqCtx.Method
	isValidMethod := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete

	return isValidMethod
}

func (p *CSRFPlugin) signedOutMatcher(reqCtx *models.RequestContext) bool {
	signedOut, ok := reqCtx.Values[models.ContextAuthSignOut.String()].(bool)
	return ok && signedOut
}

func (p *CSRFPlugin) generateCSRFTokenHook(reqCtx *models.RequestContext) error {
	method := reqCtx.Method
	if method != http.MethodOptions && method != http.MethodHead && method != http.MethodGet {
		return nil
	}

	_, err := reqCtx.Request.Cookie(p.pluginConfig.CookieName)
	if err != http.ErrNoCookie {
		return nil
	}

	token, err := p.tokenService.Generate()
	if err != nil {
		reqCtx.SetJSONResponse(
			http.StatusInternalServerError,
			map[string]string{"message": "failed to generate csrf token"},
		)
		reqCtx.Handled = true
		return nil
	}
	p.setCSRFCookie(reqCtx, token)

	return nil
}

func (p *CSRFPlugin) validateCSRFTokenHook(reqCtx *models.RequestContext) error {
	method := reqCtx.Method

	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return nil
	}

	if err := p.validateHeaderProtection(reqCtx.Request); err != nil {
		reqCtx.SetJSONResponse(
			http.StatusForbidden,
			map[string]any{"message": "csrf validation failed"},
		)
		reqCtx.Handled = true
		return nil
	}

	if err := p.validateCSRFToken(reqCtx); err != nil {
		reqCtx.Handled = true
		return nil
	}

	return nil
}

func (p *CSRFPlugin) clearCSRFTokenHook(reqCtx *models.RequestContext) error {
	http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
		Name:     p.pluginConfig.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   p.pluginConfig.Secure,
		MaxAge:   -1,
	})
	return nil
}
