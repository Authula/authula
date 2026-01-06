package bearer

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// Constants for bearer plugin hook IDs and metadata
const (
	// HookIDBearerAuth identifies the bearer token authentication hook
	// Validates Authorization: Bearer <token> header and sets ctx.UserID
	HookIDBearerAuth = "bearer.auth"
)

// validateBearerToken hook validates a bearer token from the Authorization header
// This hook runs at HookBefore stage if "bearer.auth" is in route.Metadata["plugins"]
// Validates the token using the JWT service and sets ctx.UserID if valid.
func (p *BearerPlugin) validateBearerToken(reqCtx *models.RequestContext) error {
	p.logger.Debug("bearer auth hook running", "path", reqCtx.Path, "method", reqCtx.Method)

	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		p.logger.Debug("user already authenticated, skipping bearer validation")
		return nil
	}

	// Extract bearer token from Authorization header
	token, err := p.extractToken(reqCtx.Request)
	if err != nil {
		p.logger.Debug("bearer token extraction failed", "error", err)
		// Write 401 Unauthorized
		reqCtx.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		reqCtx.Handled = true
		return nil
	}

	// Validate token via JWT service (doesn't directly validate JWT, uses JWT service)
	p.logger.Debug("validating bearer token via JWT service")
	userID, err := p.jwtService.ValidateToken(token)
	if err != nil {
		p.logger.Debug("bearer token validation failed", "error", err)
		// Write 401 Unauthorized
		reqCtx.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		reqCtx.Handled = true
		return nil
	}

	reqCtx.SetUserIDInContext(userID)
	p.logger.Debug("bearer token validation successful", "user_id", userID)

	return nil
}

// buildHooks returns the configured hooks for this plugin
// Uses the new PluginID-based hook filtering for metadata-driven execution
func (p *BearerPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// Bearer authentication hook: validates Authorization header, sets UserID
		// Executes if "bearer.auth" is in route.Metadata["plugins"]
		{
			Stage:    models.HookBefore,
			PluginID: HookIDBearerAuth,
			Handler:  p.validateBearerToken,
			Order:    5, // Higher priority than session (which is 10)
		},
	}
}
