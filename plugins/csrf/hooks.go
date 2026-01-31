package csrf

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type CSRFHookID string

// Constants for CSRF plugin hook IDs and metadata
const (
	// HookIDCSRFGenerate identifies the CSRF token generation hook.
	// Generates CSRF tokens for safe requests
	HookIDCSRFGenerate CSRFHookID = "csrf.generate"

	// HookIDCSRFProtect identifies the CSRF protection hook.
	// Validates CSRF tokens on state-changing requests
	HookIDCSRFProtect CSRFHookID = "csrf.protect"
)

func (id CSRFHookID) String() string {
	return string(id)
}

// safeMethodMatcher returns true for safe HTTP methods (non-state-changing requests)
func (p *CSRFPlugin) safeMethodMatcher(reqCtx *models.RequestContext) bool {
	method := reqCtx.Method
	isValidMethod := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	return isValidMethod
}

// unsafeMethodMatcher returns true for unsafe HTTP methods (state-changing requests)
func (p *CSRFPlugin) unsafeMethodMatcher(reqCtx *models.RequestContext) bool {
	method := reqCtx.Method
	isValidMethod := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete

	return isValidMethod
}

// signedOutMatcher returns true if the request context indicates a sign-out action
func (p *CSRFPlugin) signedOutMatcher(reqCtx *models.RequestContext) bool {
	signedOut, ok := reqCtx.Values[models.ContextAuthSignOut.String()].(bool)
	return ok && signedOut
}

// generateCSRFTokenHook generates and sets CSRF tokens for safe methods
// This hook runs on all GET/HEAD/OPTIONS requests
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

// validateCSRFTokenHook validates CSRF tokens on state-changing requests
// This hook runs on unsafe methods (POST, PUT, PATCH, DELETE)
// First validates headers using Go 1.25 CrossOriginProtection (if enabled),
// then validates the token using Double-Submit Cookie pattern
func (p *CSRFPlugin) validateCSRFTokenHook(reqCtx *models.RequestContext) error {
	// Get method - from ctx.Method or from the request
	method := reqCtx.Method

	// Only validate on unsafe methods (POST, PUT, PATCH, DELETE)
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return nil
	}

	// Step 1: Validate header-based cross-origin protection (if enabled)
	if err := p.validateHeaderProtection(reqCtx.Request); err != nil {
		// The custom deny handler in the plugin's Init() method writes the response
		// but we need to set it here too for the hook system
		reqCtx.SetJSONResponse(
			http.StatusForbidden,
			map[string]any{"message": "csrf validation failed"},
		)
		reqCtx.Handled = true
		return nil
	}

	// Step 2: Validate CSRF token (Double-Submit Cookie pattern)
	if err := p.validateCSRFToken(reqCtx); err != nil {
		reqCtx.Handled = true
		return nil // Return nil to avoid propagating error through hook chain
	}

	return nil
}

// clearCSRFTokenHook clears the CSRF token cookie
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

// buildHooks returns the configured hooks for this plugin
// CSRF protection is provided through both token generation and validation hooks
// Uses PluginID-based filtering so CSRF only executes when explicitly configured in route metadata
func (p *CSRFPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// CSRF token generation hook: generates tokens for safe methods
		// Handler: generates token for safe methods
		{
			Stage:   models.HookBefore,
			Matcher: p.safeMethodMatcher,
			Handler: p.generateCSRFTokenHook,
			Order:   5, // Execute before auth but before main handler
		},
		// CSRF protection hook: validates tokens on state-changing requests
		// PluginID-based: only executes if "csrf.protect" is in route.Metadata["plugins"]
		// Matcher: unsafe methods (POST, PUT, PATCH, DELETE)
		{
			Stage:    models.HookBefore,
			PluginID: HookIDCSRFProtect.String(),
			Matcher:  p.unsafeMethodMatcher,
			Handler:  p.validateCSRFTokenHook,
			Order:    15,
		},
		// CSRF clear hook: clears CSRF cookie on sign-out
		{
			Stage:   models.HookAfter,
			Matcher: p.signedOutMatcher,
			Handler: p.clearCSRFTokenHook,
			Order:   15, // Execute after session clear
		},
	}
}
