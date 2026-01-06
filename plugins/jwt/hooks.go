package jwt

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// Constants for JWT plugin hook IDs and metadata
const (
	// HookIDJWTAuth identifies the JWT authentication hook
	// Validates JWT tokens in Authorization header and sets ctx.UserID
	HookIDJWTAuth = "jwt.auth"

	// HookIDJWTIssuance identifies the JWT issuance hook
	// Issues access and refresh tokens on successful authentication
	HookIDJWTIssuance = "jwt.issuance"
)

// issueTokensHook generates and stores JWT tokens for authenticated users
// This hook runs at HookAfter stage if "jwt.issuance" is in route.Metadata["plugins"]
// Expects ctx.Values["user_id"] and ctx.Values["session_id"] to be set
func (p *JWTPlugin) issueTokensHook(ctx *models.RequestContext) error {
	p.Logger.Debug("jwt issuance hook running", "path", ctx.Path)

	// Extract user_id and session_id from context values
	userID, ok := ctx.Values["user_id"].(string)
	if !ok || userID == "" {
		p.Logger.Debug("no user_id found in context values")
		return nil // No user ID, skip token issuance
	}

	sessionID, ok := ctx.Values["session_id"].(string)
	if !ok || sessionID == "" {
		p.Logger.Debug("jwt hook skipped due to missing session ID", "user_id", userID)
		return nil // No session ID, skip token issuance
	}

	// Generate tokens
	tokenPair, err := p.GenerateTokens(p.globalConfig.Secret, userID, sessionID)
	if err != nil {
		p.Logger.Error("failed to generate JWT tokens", "user_id", userID, "session_id", sessionID, "error", err)
		// Return error to fail the request - JWT generation should not silently fail
		return fmt.Errorf("failed to generate authentication tokens: %w", err)
	}

	// Store tokens in context for response handling
	ctx.Values["access_token"] = tokenPair.AccessToken
	ctx.Values["refresh_token"] = tokenPair.RefreshToken

	p.Logger.Debug("jwt tokens generated successfully", "user_id", userID)

	return nil
}

// buildHooks returns the configured hooks for this plugin
// Uses the new PluginID-based hook filtering for metadata-driven execution
func (p *JWTPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// JWT issuance hook: generates access and refresh tokens after authentication
		// Executes if "jwt.issuance" is in route.Metadata["plugins"]
		{
			Stage:    models.HookAfter,
			PluginID: HookIDJWTIssuance,
			Handler:  p.issueTokensHook,
			Order:    10, // Normal priority
		},
	}
}
