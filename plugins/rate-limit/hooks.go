package ratelimit

import (
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// buildHooks returns the configured hooks for this plugin
// Rate limiting is applied via HookOnRequest to check early
func (p *RateLimitPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// Rate limiting hook: checks rate limits early in request lifecycle
		// Executes for all requests that have rate limiting enabled via config
		{
			Stage:   models.HookOnRequest,
			Handler: p.handler.Handle(),
			Order:   0, // Execute early, before other hooks
		},
	}
}
