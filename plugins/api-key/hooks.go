package apikey

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

func (p *ApiKeyPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Matcher: p.matchApiKeyHeader,
			Handler: p.validateApiKeyHook(),
			Order:   5,
		},
	}
}

func (p *ApiKeyPlugin) matchApiKeyHeader(reqCtx *models.RequestContext) bool {
	return reqCtx.Request.Header.Get(p.config.Header) != ""
}

func (p *ApiKeyPlugin) validateApiKeyHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		apiKeyValue := reqCtx.Request.Header.Get(p.config.Header)

		result, err := p.Api.Verify(ctx, types.VerifyApiKeyRequest{Key: apiKeyValue})
		if err != nil {
			if p.logger != nil {
				p.logger.Error("failed to verify api key", "error", err)
			}
			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "invalid api key"})
			reqCtx.Handled = true
			return nil
		}
		if result == nil || !result.Valid || result.ApiKey == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "invalid api key"})
			reqCtx.Handled = true
			return nil
		}

		reqCtx.UserID = &result.ApiKey.ReferenceID

		p.applyApiKeyRateLimit(reqCtx, result.ApiKey)

		return nil
	}
}

func (p *ApiKeyPlugin) applyApiKeyRateLimit(reqCtx *models.RequestContext, apiKey *types.ApiKey) {
	if !apiKey.RateLimitEnabled {
		return
	}

	if apiKey.RateLimitTimeWindow == nil || apiKey.RateLimitMaxRequests == nil || *apiKey.RateLimitTimeWindow <= 0 || *apiKey.RateLimitMaxRequests <= 0 {
		return
	}

	rateLimiterService, ok := p.pluginCtx.ServiceRegistry.Get(models.ServiceRateLimit.String()).(rootservices.RateLimiterService)
	if !ok {
			if p.logger != nil {
				p.logger.Warn("rate limit service unavailable, allowing api key request")
			}
		return
	}

	allowed, count, resetAt, err := rateLimiterService.CheckAndIncrement(reqCtx.Request.Context(), apiKey.ID, time.Duration(*apiKey.RateLimitTimeWindow)*time.Second, *apiKey.RateLimitMaxRequests)
	if err != nil {
		if p.logger != nil {
			p.logger.Error("failed to check api key rate limit", "error", err, "api_key_id", apiKey.ID)
		}
		return
	}

	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(*apiKey.RateLimitMaxRequests))
	remaining := max(*apiKey.RateLimitMaxRequests-count, 0)
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

	if !allowed {
		retryAfter := int(time.Until(resetAt).Seconds())
		reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
		reqCtx.SetJSONResponse(http.StatusTooManyRequests, map[string]any{
			"message":     "rate limit exceeded",
			"retry_after": retryAfter,
			"limit":       *apiKey.RateLimitMaxRequests,
			"remaining":   0,
		})
		reqCtx.Handled = true
	}
}
