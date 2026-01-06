package ratelimit

import (
	"github.com/GoBetterAuth/go-better-auth/models"
)

// buildHooks returns the configured hooks for this plugin
// Rate limiting is applied via HookOnRequest to check early
func (p *RateLimitPlugin) buildHooks() []models.Hook {
	if p.handler == nil {
		// Handler not initialized yet, return empty hooks
		// This will be called after Init() has completed handler initialization
		return []models.Hook{}
	}

	return []models.Hook{
		// Rate limiting hook: checks rate limits early in request lifecycle
		// Executes for all requests that have rate limiting enabled via config
		{
			Stage:   models.HookOnRequest,
			Handler: p.handleRateLimitHook,
			Order:   0, // Execute early, before other hooks
		},
	}
}

// handleRateLimitHook is the hook handler that delegates to the current handler
func (p *RateLimitPlugin) handleRateLimitHook(ctx *models.RequestContext) error {
	if p.handler == nil {
		return nil
	}
	return p.handler.Handle()(ctx)
}
