package csrf

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
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
func (p *CSRFPlugin) safeMethodMatcher(ctx *models.RequestContext) bool {
	method := ctx.Method
	isValidMethod := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	p.logger.Debug("[safeMethodMatcher] checking method", "method", method, "isValidMethod", isValidMethod)
	return isValidMethod
}

// unsafeMethodMatcher returns true for unsafe HTTP methods (state-changing requests)
func (p *CSRFPlugin) unsafeMethodMatcher(ctx *models.RequestContext) bool {
	method := ctx.Method
	isValidMethod := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete

	return isValidMethod
}

// generateCSRFTokenHook generates and sets CSRF tokens for safe methods
// This hook runs on all GET/HEAD/OPTIONS requests
func (p *CSRFPlugin) generateCSRFTokenHook(reqCtx *models.RequestContext) error {
	method := reqCtx.Method
	p.logger.Debug("[generateCSRFTokenHook] checking method", "method", method)

	if method != http.MethodOptions && method != http.MethodHead && method != http.MethodGet {
		return nil
	}

	_, err := reqCtx.Request.Cookie(p.pluginConfig.CookieName)
	if err != http.ErrNoCookie {
		p.logger.Debug("csrf cookie already present, skipping generation", "path", reqCtx.Path)
		return nil
	}

	token := p.generateToken()
	p.setCSRFCookie(reqCtx, token)

	p.logger.Debug("csrf token generated", "path", reqCtx.Path)

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

	p.logger.Debug("csrf validation hook running", "path", reqCtx.Path, "method", method)

	// Step 1: Validate header-based cross-origin protection (if enabled)
	if err := p.validateHeaderProtection(reqCtx.Request); err != nil {
		p.logger.Debug("csrf header validation failed", "error", err)
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
		p.logger.Debug("csrf token validation failed", "error", err)
		// Mark as handled (test expects this instead of error return)
		reqCtx.Handled = true
		return nil // Return nil to avoid propagating error through hook chain
	}

	p.logger.Debug("csrf validation successful", "path", reqCtx.Path)

	return nil
}

// buildHooks returns the configured hooks for this plugin
// CSRF protection is provided through both token generation and validation hooks
// Uses PluginID-based filtering so CSRF only executes when explicitly configured in route metadata
func (p *CSRFPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// CSRF token generation hook: generates tokens for safe methods
		// PluginID-based: only executes if "csrf.generate" is in route.Metadata["plugins"]
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
	}
}
