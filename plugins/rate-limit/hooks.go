package ratelimit

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/Authula/authula/models"
)

func (p *RateLimitPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookOnRequest,
			Handler: p.checkRateLimitHook(),
			Order:   0, // Execute early, before all other hooks
		},
		{
			Stage:   models.HookBefore,
			Handler: p.checkStoredRateLimitRuleHook(),
			Order:   0,
		},
	}
}

func (p *RateLimitPlugin) checkRateLimitHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		validHttpMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
		if !slices.Contains(validHttpMethods, reqCtx.Request.Method) {
			return nil
		}

		key := p.pluginConfig.Prefix + reqCtx.ClientIP
		window := p.pluginConfig.Window
		max := p.pluginConfig.Max

		if rule, exists := p.pluginConfig.CustomRules[reqCtx.Request.RequestURI]; exists {
			if rule.Disabled {
				return nil
			}
			if rule.Window > 0 {
				window = rule.Window
			}
			if rule.Max > 0 {
				max = rule.Max
			}
		}

		allowed, count, resetTime, err := p.provider.CheckAndIncrement(reqCtx.Request.Context(), key, window, max)
		if err != nil {
			p.logger.Error("failed to check rate limit", "error", err, "key", key)
			// On error, allow the request to proceed (fail-open)
			return nil
		}

		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(max))
		remaining := max - count
		if remaining < 0 {
			remaining = 0
		}
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
			payload := map[string]any{
				"message":     "rate limit exceeded",
				"retry_after": retryAfter,
				"limit":       max,
				"remaining":   0,
			}
			reqCtx.SetJSONResponse(http.StatusTooManyRequests, payload)
			reqCtx.Handled = true
		}

		return nil
	}
}

func (p *RateLimitPlugin) checkStoredRateLimitRuleHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		rawValue, ok := reqCtx.Values[models.ContextRateLimitRule.String()]
		if !ok || rawValue == nil {
			return nil
		}

		ruleCtx, ok := getRateLimitRuleContext(rawValue)
		if !ok || ruleCtx.Key == "" || ruleCtx.WindowSeconds <= 0 || ruleCtx.MaxRequests <= 0 {
			return nil
		}

		window := time.Duration(ruleCtx.WindowSeconds) * time.Second
		allowed, count, resetTime, err := p.provider.CheckAndIncrement(ctx, ruleCtx.Key, window, ruleCtx.MaxRequests)
		if err != nil {
			p.logger.Error("failed to check stored rate limit rule", "error", err, "key", ruleCtx.Key)
			return nil
		}

		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(ruleCtx.MaxRequests))
		remaining := max(ruleCtx.MaxRequests-count, 0)
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
			payload := map[string]any{
				"message":     "rate limit exceeded",
				"retry_after": retryAfter,
				"limit":       ruleCtx.MaxRequests,
				"remaining":   0,
			}
			reqCtx.SetJSONResponse(http.StatusTooManyRequests, payload)
			reqCtx.Handled = true
		}

		return nil
	}
}

func getRateLimitRuleContext(value any) (models.RateLimitRuleContext, bool) {
	switch typed := value.(type) {
	case models.RateLimitRuleContext:
		return typed, true
	case *models.RateLimitRuleContext:
		if typed == nil {
			return models.RateLimitRuleContext{}, false
		}
		return *typed, true
	default:
		return models.RateLimitRuleContext{}, false
	}
}
