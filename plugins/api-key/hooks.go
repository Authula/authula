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
	if p.ctx == nil {
		return nil
	}

	return []models.Hook{
		{
			Stage: models.HookBefore,
			Matcher: func(reqCtx *models.RequestContext) bool {
				return reqCtx != nil && reqCtx.Request != nil && reqCtx.Request.Header.Get(p.config.ApiKeyHeader) != ""
			},
			Handler: p.handleApiKeyHook(),
			Order:   0,
		},
	}
}

func (p *ApiKeyPlugin) handleApiKeyHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		if reqCtx == nil || reqCtx.Request == nil {
			return nil
		}
		if p.Api == nil {
			return nil
		}

		logger := p.logger
		if logger == nil && p.ctx != nil {
			logger = p.ctx.Logger
		}

		apiKeyValue := reqCtx.Request.Header.Get(p.config.ApiKeyHeader)
		if apiKeyValue == "" {
			return nil
		}

		result, err := p.Api.Verify(reqCtx.Request.Context(), types.VerifyApiKeyRequest{Key: apiKeyValue})
		if err != nil {
			if logger != nil {
				logger.Error("failed to verify api key", "error", err)
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

		if !result.ApiKey.RateLimitEnabled {
			return nil
		}

		if result.ApiKey.RateLimitTimeWindow == nil || result.ApiKey.RateLimitMaxRequests == nil || *result.ApiKey.RateLimitTimeWindow <= 0 || *result.ApiKey.RateLimitMaxRequests <= 0 {
			return nil
		}

		var service rootservices.RateLimiterService
		if p.ctx != nil && p.ctx.ServiceRegistry != nil {
			if rawService := p.ctx.ServiceRegistry.Get(models.ServiceRateLimit.String()); rawService != nil {
				service, _ = rawService.(rootservices.RateLimiterService)
			}
		}
		if service == nil {
			if logger != nil {
				logger.Warn("rate limit service unavailable, allowing api key request")
			}
			return nil
		}

		allowed, count, resetAt, err := service.CheckAndIncrement(reqCtx.Request.Context(), result.ApiKey.ID, time.Duration(*result.ApiKey.RateLimitTimeWindow)*time.Second, *result.ApiKey.RateLimitMaxRequests)
		if err != nil {
			if logger != nil {
				logger.Error("failed to check api key rate limit", "error", err, "api_key_id", result.ApiKey.ID)
			}
			return nil
		}

		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(*result.ApiKey.RateLimitMaxRequests))
		remaining := *result.ApiKey.RateLimitMaxRequests - count
		if remaining < 0 {
			remaining = 0
		}
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(resetAt).Seconds())
			reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
			reqCtx.SetJSONResponse(http.StatusTooManyRequests, map[string]any{
				"message":     "rate limit exceeded",
				"retry_after": retryAfter,
				"limit":       *result.ApiKey.RateLimitMaxRequests,
				"remaining":   0,
			})
			reqCtx.Handled = true
		}

		return nil
	}
}
