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

// generateCSRFTokenHook generates and sets CSRF tokens for safe methods
// This hook runs on all GET/HEAD/OPTIONS requests for authenticated users
func (p *CSRFPlugin) generateCSRFTokenHook(ctx *models.RequestContext) error {
	method := ctx.Method
	if method == "" && ctx.Request != nil {
		method = ctx.Request.Method
	}

	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		return nil
	}

	_, err := ctx.Request.Cookie(p.pluginConfig.CookieName)
	if err != http.ErrNoCookie {
		return nil
	}

	token := p.generateToken()
	p.setCSRFCookie(ctx.ResponseWriter, ctx.Request, token)

	p.logger.Debug("csrf token generated", "path", ctx.Path)

	return nil
}

// combinedCSRFHook handles both token generation and validation
func (p *CSRFPlugin) combinedCSRFHook(ctx *models.RequestContext) error {
	method := ctx.Method
	if method == "" && ctx.Request != nil {
		method = ctx.Request.Method
	}

	isSafeMethod := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	if isSafeMethod {
		return p.generateCSRFTokenHook(ctx)
	}

	return p.validateCSRFTokenHook(ctx)
}

// safeMethodMatcher returns true for safe HTTP methods with authenticated users
func (p *CSRFPlugin) safeMethodMatcher(ctx *models.RequestContext) bool {
	// Get method - from ctx.Method or from the request
	method := ctx.Method
	if method == "" && ctx.Request != nil {
		method = ctx.Request.Method
	}

	// Only match safe methods (GET, HEAD, OPTIONS) for authenticated users
	isSafeMethod := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	return isSafeMethod && ctx.UserID != nil
}

// unsafeMethodMatcher returns true for unsafe HTTP methods (state-changing requests)
func (p *CSRFPlugin) unsafeMethodMatcher(ctx *models.RequestContext) bool {
	method := ctx.Method
	if method == "" && ctx.Request != nil {
		method = ctx.Request.Method
	}

	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

// validateCSRFTokenHook validates CSRF tokens on state-changing requests
// This hook runs on unsafe methods (POST, PUT, PATCH, DELETE)
// First validates headers using Go 1.25 CrossOriginProtection (if enabled),
// then validates the token using Double-Submit Cookie pattern
func (p *CSRFPlugin) validateCSRFTokenHook(reqCtx *models.RequestContext) error {
	// Get method - from ctx.Method or from the request
	method := reqCtx.Method
	if method == "" && reqCtx.Request != nil {
		method = reqCtx.Request.Method
	}

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
		err := reqCtx.SetJSONResponse(
			http.StatusForbidden,
			map[string]any{"message": "csrf validation failed"},
		)
		if err != nil {
			return err
		}
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
		// Combined CSRF hook: generates tokens for safe methods, validates for unsafe methods
		// PluginID-based: only executes if "csrf.generate" is in route.Metadata["plugins"]
		// Handler: generates token for safe methods, validates for unsafe methods
		{
			Stage:    models.HookBefore,
			PluginID: HookIDCSRFGenerate.String(),
			Matcher:  p.safeMethodMatcher,
			Handler:  p.combinedCSRFHook,
			Order:    15, // Execute after auth but before main handler
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
